package artifactrecorder

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/assistantservice/artifactpayload"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/shared/pulid"
)

// AttachedExtraction keeps what document intelligence read off a file the
// person attached as its extraction card beside the conversation. It reads
// the draft through intake_get_draft as the person, so the card shows only
// what they may read, in the same shape as the card the agent opens. It is
// nil while nothing has been read off the file, and for a file that does not
// give a shipment draft.
func (r *Recorder) AttachedExtraction(documentID pulid.ID) *assistantartifact.Artifact {
	if r == nil || r.reads == nil || r.reads.Runtime == nil || documentID.IsNil() {
		return nil
	}

	read, ok := r.reads.Runtime.ReadRecordWith(
		r.ctx,
		r.reader,
		artifactpayload.ToolGetShipmentDraft,
		documentID.String(),
	)
	if !ok {
		return nil
	}
	card := r.extraction(AttachmentCallID+documentID.String(), read.Data)
	if card == nil {
		return nil
	}
	saved, err := r.save(card)
	if err != nil {
		return nil
	}

	return saved
}

const (
	// AttachmentCallID names the extraction card kept for an attached file.
	// It is one card per file and thread, so reading the file again after
	// its extraction changed revises the card rather than adding another.
	AttachmentCallID = "attachment:"
)
