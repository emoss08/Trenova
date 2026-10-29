package inboundreplyports

import (
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/shared/pulid"
)

type ReplyAttachable struct {
	ID           pulid.ID
	FileName     string
	ContentType  string
	DocumentType string
	ByteSize     int64
	CreatedAt    int64
	Source       triage.AttachableSource
}
