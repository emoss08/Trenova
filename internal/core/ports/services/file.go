package services

import (
	"context"
	"net/http"
	"time"

	"github.com/rotisserie/eris"
)

var (
	ErrOperationTimeout     = eris.New("operation timeout")
	ErrInvalidConfiguration = eris.New("invalid configuration")

	ErrFileSizeExceedsMaxSize  = eris.New("file size exceeds the maximum allowed size")
	ErrFileExtensionNotAllowed = eris.New("file extension not allowed")
)

// FileClassification represents the security level of files
type FileClassification string

const (
	ClassificationPublic     = FileClassification("public")    // Publicly accessible files
	ClassificationPrivate    = FileClassification("private")   // Internal files
	ClassificationSensitive  = FileClassification("sensitive") // Sensitive files
	ClassificationRegulatory = FileClassification("regulatory")
)

func (fc FileClassification) String() string {
	return string(fc)
}

type FileCategory string

const (
	CategoryShipment   = FileCategory("shipment")   // BOL, POD, etc...
	CategoryWorker     = FileCategory("worker")     // Worker docs, licenses
	CategoryRegulatory = FileCategory("regulatory") // Regulatory docs, certificates, etc...
	CategoryProfile    = FileCategory("profile")    // Profile photos, etc...
	CategoryBranding   = FileCategory("branding")   // Branding files, etc...
)

func (fc FileCategory) String() string {
	return string(fc)
}

type FileType string

const (
	ImageFile = FileType("image")
	DocFile   = FileType("document")
	PDFFile   = FileType("pdf")
)

func (ft FileType) String() string {
	return string(ft)
}

type Metadata struct {
	OrgID       string
	UserID      string
	FileType    FileType
	Tags        map[string]string
	ContentType string
}

type SaveFileRequest struct {
	File           []byte
	FileName       string
	OrgID          string
	FileType       FileType
	Classification FileClassification
	Category       FileCategory
	BucketName     string
	UserID         string
	Metadata       http.Header
	Tags           map[string]string
	VersionComment string // Optional comment for version history
}

type SaveFileResponse struct {
	Key            string      `json:"key"`
	Location       string      `json:"location"`
	Etag           string      `json:"etag"`
	Checksum       string      `json:"checksum"`
	BucketName     string      `json:"bucketName"`
	Size           int64       `json:"size"`
	Expiration     time.Time   `json:"expiration"`
	ContentType    string      `json:"contentType"`
	Metadata       http.Header `json:"metadata"`
	VersionComment string      // Optional comment for version history
}

type VersionInfo struct {
	VersionID      string      `json:"versionId"`
	LastModified   time.Time   `json:"lastModified"`
	CreatedBy      string      `json:"createdBy"`
	Comment        string      `json:"comment,omitempty"`
	Size           int64       `json:"size"`
	Checksum       string      `json:"checksum"`
	Metadata       http.Header `json:"metadata"`
	IsLatest       bool        `json:"isLatest"`
	Classification string      `json:"classification"`
}

type ClassificationPolicy struct {
	RetentionPeriod    time.Duration
	RequiresEncryption bool
	AllowedCategories  []FileCategory
	MaxFileSize        int64  // Override default if needed
	RequireVersioning  bool   // Whether versioning is required for this classification
	MaxVersions        int    // Maxiumum number of versions to keep (0 = unlimited)
	VersionRetention   string // Version retention policy (none, all, latest-n)
}

type FileService interface {
	SaveFile(ctx context.Context, req *SaveFileRequest) (*SaveFileResponse, error)
	SaveFileVersion(ctx context.Context, req *SaveFileRequest) (*SaveFileResponse, error)
	GetFileVersion(ctx context.Context, bucketName, objectName string) ([]VersionInfo, error)
	GetSpecificVersion(ctx context.Context, bucketName, objectName, versionID string) ([]byte, *VersionInfo, error)
	RestoreVersion(ctx context.Context, req *SaveFileRequest, versionID string) (*SaveFileResponse, error)
	ValidateFile(filename string, size int64, fileType FileType) error
	GetFileURL(ctx context.Context, bucketName, objectName string, expiry time.Duration) (string, error)
	DeleteFile(ctx context.Context, bucketName, objectName string) error
}
