package billingqueueports

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type PacketPreviewRequest struct {
	ItemID     pulid.ID
	TenantInfo pagination.TenantInfo
}

// PacketPreview is the one PDF an item's invoice goes out as: the invoice
// approving it would make, followed by the documents it carries.
type PacketPreview struct {
	FileName  string
	Content   []byte
	Documents int
}

type PacketPreviewer interface {
	PreviewPacket(ctx context.Context, req *PacketPreviewRequest) (*PacketPreview, error)
}
