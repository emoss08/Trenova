package assistantports

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// MessageFileReader reads a file a person attached to a conversation, for a
// model that reads files itself.
type MessageFileReader interface {
	ReadMessageFile(ctx context.Context, req *ReadMessageFileRequest) (*MessageFileContent, error)
}

type ReadMessageFileRequest struct {
	TenantInfo pagination.TenantInfo
	DocumentID pulid.ID
}

// MessageFileContent is an attached file's bytes as stored, decrypted.
type MessageFileContent struct {
	Data        []byte
	ContentType string
	// PageCount is the pages of a PDF, counted from the bytes; zero for any
	// other file, or when they could not be counted.
	PageCount int
	// Unreadable marks a PDF that would not open, encrypted or damaged, which
	// a model would refuse too.
	Unreadable bool
}
