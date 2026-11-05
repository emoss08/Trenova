package documentservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config/storageconfig"
	"github.com/stretchr/testify/assert"
)

func uploadValidator() *Validator {
	return NewValidator(ValidatorParams{StorageConfig: &storageconfig.Config{MaxFileSize: 1 << 20}})
}

func TestValidateUploadMetadata_RefusesADangerousNameHoweverItEnds(t *testing.T) {
	t.Parallel()

	for _, name := range []string{
		"run.exe",
		"run.EXE",
		"run.exe.",
		"run.exe...",
		"run.exe ",
		"run.exe. .",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.NotNil(
				t,
				uploadValidator().ValidateUploadMetadata(name, 10, "application/pdf"),
			)
		})
	}
}

func TestValidateUploadMetadata_HoldsAnUndeclaredTypeToTheAllowlist(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		fileName string
		refused  bool
	}{
		{name: "pdf by its extension", fileName: "pod.pdf"},
		{name: "heic by its extension", fileName: "photo.HEIC"},
		{name: "html has no allowed type", fileName: "page.html", refused: true},
		{name: "svg has no allowed type", fileName: "logo.svg", refused: true},
		{name: "no extension at all", fileName: "upload", refused: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := uploadValidator().ValidateUploadMetadata(tt.fileName, 10, "")

			if tt.refused {
				assert.NotNil(t, err)
				return
			}
			assert.Nil(t, err)
		})
	}
}

func TestValidateUploadMetadata_ADeclaredTypeIsStillHeldToTheAllowlist(t *testing.T) {
	t.Parallel()

	assert.NotNil(t, uploadValidator().ValidateUploadMetadata("pod.pdf", 10, "text/html"))
	assert.Nil(t, uploadValidator().ValidateUploadMetadata("pod.pdf", 10, "application/pdf"))
}

func TestResolvedContentType_NamesTheTypeStoredForAnUndeclaredFile(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "application/pdf", resolvedContentType("pod.pdf", ""))
	assert.Equal(t, "image/png", resolvedContentType("scan.png", "image/png"))
	assert.Empty(t, resolvedContentType("upload", ""))
}
