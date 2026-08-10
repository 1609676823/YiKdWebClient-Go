package yikdwebclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type requestEnvelope struct {
	Format     int       `json:"format"`
	UserAgent  string    `json:"useragent"`
	RID        string    `json:"rid"`
	Parameters string    `json:"parameters"`
	Timestamp  time.Time `json:"timestamp"`
	Version    string    `json:"v"`
}

func standardEnvelope(parameters string) requestEnvelope {
	return requestEnvelope{
		Format: 1, UserAgent: "ApiClient", RID: newRID(), Parameters: parameters,
		Timestamp: time.Now(), Version: "1.0",
	}
}

func marshalCompatibleJSON(value any, unsafeRelaxed, indented bool) (string, error) {
	var output bytes.Buffer
	encoder := json.NewEncoder(&output)
	encoder.SetEscapeHTML(!unsafeRelaxed)
	if indented {
		encoder.SetIndent("", "  ")
	}
	if err := encoder.Encode(value); err != nil {
		return "", err
	}
	result := strings.TrimSuffix(output.String(), "\n")
	if !unsafeRelaxed {
		result = strings.NewReplacer(
			`\u003c`, `\u003C`, `\u003e`, `\u003E`, `\u0026`, `\u0026`,
			"'", `\u0027`,
		).Replace(result)
	}
	return result, nil
}

// GetRequestBodyString builds the standard DynamicFormService request envelope.
func GetRequestBodyString(formID, payload string, unsafeRelaxed bool, opNumber string) (string, error) {
	parameters := make([]string, 0, 3)
	if strings.TrimSpace(formID) != "" {
		parameters = append(parameters, formID)
	}
	if strings.TrimSpace(opNumber) != "" {
		parameters = append(parameters, opNumber)
	}
	parameters = append(parameters, payload)
	content, err := marshalCompatibleJSON(parameters, unsafeRelaxed, false)
	if err != nil {
		return "", fmt.Errorf("marshal request parameters: %w", err)
	}
	result, err := marshalCompatibleJSON(standardEnvelope(content), unsafeRelaxed, true)
	if err != nil {
		return "", fmt.Errorf("marshal request envelope: %w", err)
	}
	return result, nil
}

// GetLoginRequestBodyString builds the full login request envelope.
func GetLoginRequestBodyString(payload string, unsafeRelaxed, writeIndented bool) (string, error) {
	result, err := marshalCompatibleJSON(standardEnvelope(payload), unsafeRelaxed, writeIndented)
	if err != nil {
		return "", fmt.Errorf("marshal login envelope: %w", err)
	}
	return result, nil
}

// GetLoginRequestBodyStringByParameters builds the compact LoginBySign envelope.
func GetLoginRequestBodyStringByParameters(payload string, unsafeRelaxed, writeIndented bool) (string, error) {
	result, err := marshalCompatibleJSON(struct {
		Parameters string `json:"parameters"`
	}{Parameters: payload}, unsafeRelaxed, writeIndented)
	if err != nil {
		return "", fmt.Errorf("marshal login parameters: %w", err)
	}
	return result, nil
}
