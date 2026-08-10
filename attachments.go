package yikdwebclient

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const DefaultChunkSize int64 = 1024 * 1024

// FileChunk describes one binary attachment chunk.
type FileChunk struct {
	Chunkindex  int64
	Filename    string
	IsLast      bool
	Chunkbyte   []byte
	ChunkBase64 string
}

// SetChunkBytes copies bytes into the chunk and updates ChunkBase64.
func (c *FileChunk) SetChunkBytes(data []byte) {
	c.Chunkbyte = append(c.Chunkbyte[:0], data...)
	c.ChunkBase64 = base64.StdEncoding.EncodeToString(c.Chunkbyte)
}

// ChunkAction handles one chunk and may stop iteration by returning an error.
type ChunkAction func(*FileChunk) error

// ProgressAction receives a successfully submitted chunk.
type ProgressAction func(*FileChunk, *YiK3CloudClient) error

func validateChunkArguments(action ChunkAction, chunkSize int64) error {
	if action == nil {
		return fmt.Errorf("chunkAction 不能为空")
	}
	if chunkSize <= 0 || chunkSize > int64(^uint32(0)>>1) {
		return fmt.Errorf("分块大小必须大于 0 且不能超过 Int32.MaxValue")
	}
	return nil
}

// ReadFileInChunksByAction streams a file without loading the complete file.
func ReadFileInChunksByAction(filePath string, action ChunkAction, chunkSize ...int64) error {
	size := DefaultChunkSize
	if len(chunkSize) > 0 {
		size = chunkSize[0]
	}
	if err := validateChunkArguments(action, size); err != nil {
		return err
	}
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open attachment %q: %w", filePath, err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("stat attachment %q: %w", filePath, err)
	}
	buffer := make([]byte, int(size))
	var position, chunkIndex int64
	for {
		read, readErr := file.Read(buffer)
		if read > 0 {
			position += int64(read)
			chunk := &FileChunk{
				Chunkindex: chunkIndex, Filename: filepath.Base(filePath), IsLast: position >= info.Size(),
			}
			chunk.SetChunkBytes(buffer[:read])
			if err := action(chunk); err != nil {
				return err
			}
			chunkIndex++
		}
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			return fmt.Errorf("read attachment %q: %w", filePath, readErr)
		}
	}
	return nil
}

// ReadBase64ChunksByAction decodes Base64 and visits fixed-size binary chunks.
func ReadBase64ChunksByAction(
	base64Data, fileName string, action ChunkAction, chunkSize ...int64,
) error {
	size := DefaultChunkSize
	if len(chunkSize) > 0 {
		size = chunkSize[0]
	}
	if err := validateChunkArguments(action, size); err != nil {
		return err
	}
	data, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return fmt.Errorf("decode attachment Base64: %w", err)
	}
	var chunkIndex int64
	for offset := int64(0); offset < int64(len(data)); offset += size {
		end := offset + size
		if end > int64(len(data)) {
			end = int64(len(data))
		}
		chunk := &FileChunk{
			Chunkindex: chunkIndex, Filename: fileName, IsLast: end >= int64(len(data)),
		}
		chunk.SetChunkBytes(data[int(offset):int(end)])
		if err := action(chunk); err != nil {
			return err
		}
		chunkIndex++
	}
	return nil
}

// UploadResponseError reports an unsuccessful or malformed upload response.
type UploadResponseError struct{ Response string }

func (e *UploadResponseError) Error() string {
	return "attachment upload was not accepted: " + e.Response
}

func uploadResponse(response string) (bool, string, error) {
	var root struct {
		Result struct {
			ResponseStatus struct {
				IsSuccess any `json:"IsSuccess"`
			} `json:"ResponseStatus"`
			FileID any `json:"FileId"`
		} `json:"Result"`
	}
	if err := json.Unmarshal([]byte(response), &root); err != nil {
		return false, "", fmt.Errorf("parse attachment response: %w", err)
	}
	success := false
	switch value := root.Result.ResponseStatus.IsSuccess.(type) {
	case bool:
		success = value
	case string:
		success, _ = strconv.ParseBool(value)
	}
	fileID := ""
	switch value := root.Result.FileID.(type) {
	case string:
		fileID = value
	case float64:
		fileID = strconv.FormatFloat(value, 'f', -1, 64)
	case nil:
	default:
		fileID = fmt.Sprint(value)
	}
	return success, fileID, nil
}

func uploadChunks(
	readChunks func(ChunkAction) error,
	client *YiK3CloudClient,
	template *UploadModel,
	progress ProgressAction,
) (string, error) {
	if client == nil {
		return "", fmt.Errorf("YiK3CloudClient must not be nil")
	}
	if template == nil {
		return "", fmt.Errorf("UploadModel must not be nil")
	}
	response := ""
	err := readChunks(func(chunk *FileChunk) error {
		template.Data.FileName = chunk.Filename
		template.Data.SendByte = chunk.ChunkBase64
		template.Data.IsLast = chunk.IsLast
		payload, err := marshalCompatibleJSON(
			template, client.UnsafeRelaxedJsonEscaping, true,
		)
		if err != nil {
			return fmt.Errorf("marshal attachment chunk: %w", err)
		}
		response, err = client.AttachmentUpLoad(payload)
		if err != nil {
			return err
		}
		if progress != nil {
			if err := progress(chunk, client); err != nil {
				return err
			}
		}
		success, fileID, parseErr := uploadResponse(response)
		if parseErr != nil {
			return &UploadResponseError{Response: response}
		}
		if !success {
			return &UploadResponseError{Response: response}
		}
		template.Data.FileId = fileID
		return nil
	})
	return response, err
}

// AttachmentUploadByFilePath uploads a file in chunks.
func AttachmentUploadByFilePath(
	filePath string,
	client *YiK3CloudClient,
	template *UploadModel,
	chunkSize int64,
	progress ProgressAction,
) (string, error) {
	if chunkSize == 0 {
		chunkSize = DefaultChunkSize
	}
	return uploadChunks(
		func(action ChunkAction) error {
			return ReadFileInChunksByAction(filePath, action, chunkSize)
		},
		client, template, progress,
	)
}

// AttachmentUploadByBase64 uploads decoded Base64 data in chunks.
func AttachmentUploadByBase64(
	base64Data, fileName string,
	client *YiK3CloudClient,
	template *UploadModel,
	chunkSize int64,
	progress ProgressAction,
) (string, error) {
	if chunkSize == 0 {
		chunkSize = DefaultChunkSize
	}
	return uploadChunks(
		func(action ChunkAction) error {
			return ReadBase64ChunksByAction(base64Data, fileName, action, chunkSize)
		},
		client, template, progress,
	)
}

// CheckUploadModelData validates all fields required by an already-built chunk.
func CheckUploadModelData(template *UploadModel) error {
	if template == nil {
		return fmt.Errorf("UploadModel 不能为空")
	}
	data := template.Data
	required := []struct {
		value   string
		message string
	}{
		{data.FileName, "文件名不能为空"},
		{data.FormId, "表单ID不能为空"},
		{data.InterId, "单据内码不能为空"},
		{data.FileId, "文件ID不能为空"},
		{data.SendByte, "文件字节流不能为空"},
	}
	for _, field := range required {
		if strings.TrimSpace(field.value) == "" {
			return errors.New(field.message)
		}
	}
	hasEntryKey := strings.TrimSpace(data.Entrykey) != ""
	hasEntryID := strings.TrimSpace(data.EntryinterId) != "" && data.EntryinterId != "-1"
	if hasEntryKey != hasEntryID {
		return fmt.Errorf("Entrykey 和 EntryinterId 要么全有，要么全没有")
	}
	return nil
}
