package detentionnoticerender

import (
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
)

// NoticePDF is the rendered notice on paper, carried from the render to the
// send and then to storage.
//
// It exists because the bytes are needed in three places and must be identical
// in all of them: the email attachment the customer opens, the shipment document
// the claim file cites, and the generated-documents row that says which template
// version produced them.
type NoticePDF struct {
	Content  []byte
	FileName string

	// Provenance is the render's source, ids, and content hash with the bytes
	// stripped out. RecordGeneratedDocument wants exactly this, and the workflow
	// that files the document carries it through a Temporal payload where
	// megabytes of PDF have no business being.
	Provenance *documenttemplateports.RenderedDocument
}
