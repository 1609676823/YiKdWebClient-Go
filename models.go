package yikdwebclient

import (
	"net/http"
	"strings"
)

// LoginType identifies the authentication mode used by YiK3CloudClient.
type LoginType string

const (
	LoginTypeSignSHA256       LoginType = "LoginBySignSHA256"
	LoginTypeSignSHA1         LoginType = "LoginBySignSHA1"
	LoginTypeAppSecret        LoginType = "LoginByAppSecret"
	LoginTypeAPISignHeaders   LoginType = "ApiSignHeaders"
	LoginTypeValidateLogin    LoginType = "ValidateLogin"
	LoginTypeSimplePassport   LoginType = "LoginBySimplePassport"
	LoginTypeValidateUserCode LoginType = "ValidateUserEnDeCode"
)

// OperationType mirrors the operation names exposed by the C# client.
type OperationType string

const (
	OperationView               OperationType = "View"
	OperationSave               OperationType = "Save"
	OperationBatchSave          OperationType = "BatchSave"
	OperationSubmit             OperationType = "Submit"
	OperationAudit              OperationType = "Audit"
	OperationUnAudit            OperationType = "UnAudit"
	OperationDelete             OperationType = "Delete"
	OperationExecuteBillQuery   OperationType = "ExecuteBillQuery"
	OperationDraft              OperationType = "Draft"
	OperationAllocate           OperationType = "Allocate"
	OperationPush               OperationType = "Push"
	OperationGroupSave          OperationType = "GroupSave"
	OperationFlexSave           OperationType = "FlexSave"
	OperationSendMsg            OperationType = "SendMsg"
	OperationLogout             OperationType = "Logout"
	OperationExecuteOperation   OperationType = "ExecuteOperation"
	OperationSwitchOrg          OperationType = "SwitchOrg"
	OperationWorkflowAudit      OperationType = "WorkflowAudit"
	OperationDynamicFormService OperationType = "DynamicFormService"
)

// BySimplePassportType specifies where a simple-passport payload comes from.
type BySimplePassportType string

const (
	SimplePassportCnfFile   BySimplePassportType = "CnfFile"
	SimplePassportForBase64 BySimplePassportType = "ForBase64"
)

// AppSettingsModel contains the values from YiKdWebCfg/appsettings.xml.
type AppSettingsModel struct {
	XKDApiAcctID    string
	XKDApiAppID     string
	XKDApiAppSec    string
	XKDApiUserName  string
	XKDApiLCID      string
	XKDApiServerUrl string
	XKDApiOrgNum    string
}

// NormalizeServerURL returns an empty string for a blank value and otherwise
// makes sure that the value ends in a slash.
func NormalizeServerURL(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	if strings.HasSuffix(value, "/") {
		return value
	}
	return value + "/"
}

// Normalize normalizes URL-valued fields after direct struct assignment.
func (m *AppSettingsModel) Normalize() {
	if m != nil {
		m.XKDApiServerUrl = NormalizeServerURL(m.XKDApiServerUrl)
	}
}

// CustomServicesStubpath describes a custom Kingdee service stub.
// Projet* keeps the original project's public spelling for source familiarity.
type CustomServicesStubpath struct {
	ProjetNamespace   string
	ProjetClassName   string
	ProjetClassMethod string
}

// RemoveSpaces removes every ASCII space, matching the original client.
func RemoveSpaces(value string) string { return strings.ReplaceAll(value, " ", "") }

// GetCustomServicesStubpathURL builds the complete custom service stub path.
func (m CustomServicesStubpath) GetCustomServicesStubpathURL() string {
	namespace := RemoveSpaces(m.ProjetNamespace)
	return namespace + "." + RemoveSpaces(m.ProjetClassName) + "." +
		RemoveSpaces(m.ProjetClassMethod) + "," + namespace + ".common.kdsvc"
}

// GetCustomServicesStubpathUrl is the C#-style spelling kept for migration.
func (m CustomServicesStubpath) GetCustomServicesStubpathUrl() string {
	return m.GetCustomServicesStubpathURL()
}

// ValidateLoginSettingsModel configures password-based validation login.
type ValidateLoginSettingsModel struct {
	Url      string
	DbId     string
	UserName string
	Password string
	Lcid     int
}

// NewValidateLoginSettings creates validation settings with C# defaults.
func NewValidateLoginSettings(serverURL string) *ValidateLoginSettingsModel {
	return &ValidateLoginSettingsModel{Url: NormalizeServerURL(serverURL), Lcid: 2052}
}

// Normalize normalizes URL-valued fields and applies zero-value defaults.
func (m *ValidateLoginSettingsModel) Normalize() {
	if m == nil {
		return
	}
	m.Url = NormalizeServerURL(m.Url)
	if m.Lcid == 0 {
		m.Lcid = 2052
	}
}

// LoginBySimplePassportModel configures simple-passport login.
type LoginBySimplePassportModel struct {
	Url                     string
	CnfFilePath             string
	SimplePassportForBase64 string
	Lcid                    int
	BySimplePassportType    BySimplePassportType
}

// NewLoginBySimplePassportModel creates simple-passport settings with C# defaults.
func NewLoginBySimplePassportModel(serverURL string) *LoginBySimplePassportModel {
	return &LoginBySimplePassportModel{
		Url:                  NormalizeServerURL(serverURL),
		Lcid:                 2052,
		BySimplePassportType: SimplePassportCnfFile,
	}
}

// Normalize normalizes URL-valued fields and applies zero-value defaults.
func (m *LoginBySimplePassportModel) Normalize() {
	if m == nil {
		return
	}
	m.Url = NormalizeServerURL(m.Url)
	if m.Lcid == 0 {
		m.Lcid = 2052
	}
	if m.BySimplePassportType == "" {
		m.BySimplePassportType = SimplePassportCnfFile
	}
}

// RequestWebModel records the actual URL, body, response, and cookie jar.
type RequestWebModel struct {
	Cookie           http.CookieJar
	RequestUrl       string
	RealRequestBody  string
	RealResponseBody string
}

// UploadModel is the wire envelope expected by AttachmentUpLoad.
type UploadModel struct {
	Data UploadModelData `json:"data"`
}

// UploadModelData describes one attachment chunk.
type UploadModelData struct {
	FileName      string `json:"FileName"`
	FormId        string `json:"FormId"`
	IsLast        bool   `json:"IsLast"`
	InterId       string `json:"InterId"`
	Entrykey      string `json:"Entrykey"`
	EntryinterId  string `json:"EntryinterId"`
	BillNO        string `json:"BillNO"`
	AliasFileName string `json:"AliasFileName"`
	FileId        string `json:"FileId"`
	SendByte      string `json:"SendByte"`
}

// NewUploadModel creates an upload model with the original EntryinterId default.
func NewUploadModel() *UploadModel {
	return &UploadModel{Data: UploadModelData{EntryinterId: "-1"}}
}

// SimplePassportLoginArg is the payload embedded in SSO V2-V4 URLs.
type SimplePassportLoginArg struct {
	DbID       string  `json:"dbid"`
	Username   string  `json:"username"`
	AppID      string  `json:"appid"`
	SignedData string  `json:"signeddata"`
	Timestamp  string  `json:"timestamp"`
	Lcid       string  `json:"lcid"`
	OriginType string  `json:"origintype"`
	EntryRole  string  `json:"entryrole"`
	FormID     string  `json:"formid"`
	FormType   string  `json:"formtype"`
	PKID       string  `json:"pkid"`
	OtherArgs  string  `json:"otherargs"`
	OpenMode   *string `json:"openmode"`
	LoginThen  *string `json:"loginthen"`
}

// NewSimplePassportLoginArg creates an SSO payload with the original defaults.
func NewSimplePassportLoginArg() SimplePassportLoginArg {
	return SimplePassportLoginArg{Lcid: "2052", OriginType: "SimPas"}
}

// SSOLoginURLObject contains the three client-specific SSO URLs.
type SSOLoginURLObject struct {
	SilverlightURL string
	HTML5URL       string
	WPFURL         string
}

// SSOLoginUrlObject is a compatibility alias for the C# type spelling.
type SSOLoginUrlObject = SSOLoginURLObject

// SSOLogoutObject contains the logout endpoint and its ap0 form value.
type SSOLogoutObject struct {
	RequestLogoutURL string
	AP0              string
}
