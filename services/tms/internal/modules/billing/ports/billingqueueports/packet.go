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

type PacketPartKind string

const (
	PacketPartInvoice  PacketPartKind = "Invoice"
	PacketPartDocument PacketPartKind = "Document"
)

type PacketPart struct {
	Kind       PacketPartKind
	DocumentID pulid.ID
	Name       string
	FirstPage  int
	LastPage   int
	Unsigned   bool
}

// PacketPreview is the one PDF an item's invoice goes out as: the invoice
// approving it would make, followed by the documents it carries.
type PacketPreview struct {
	FileName  string
	Content   []byte
	Pages     int
	Documents int
	Build     string
	Parts     []PacketPart
}

type PacketPageRequest struct {
	ItemID     pulid.ID
	TenantInfo pagination.TenantInfo
	Page       int
}

type PacketPage struct {
	Content     []byte
	ContentType string
}

type PacketChecksRequest struct {
	ItemID     pulid.ID
	TenantInfo pagination.TenantInfo
}

type PacketChecks struct {
	Build      string
	BlankPages []int
}

type PacketPreviewer interface {
	PreviewPacket(ctx context.Context, req *PacketPreviewRequest) (*PacketPreview, error)
	PreviewPacketPage(ctx context.Context, req *PacketPageRequest) (*PacketPage, error)
	PreviewPacketThumbnail(ctx context.Context, req *PacketPageRequest) (*PacketPage, error)
	PreviewPacketChecks(ctx context.Context, req *PacketChecksRequest) (*PacketChecks, error)
}
