package inboundmessage

import "github.com/emoss08/trenova/shared/pulid"

// ApplyReview settles the message as a person, or an approved write, decided
// it: its status, who settled it and when, and the note left with it. An
// empty note leaves the one already on the message.
func (m *InboundMessage) ApplyReview(status Status, reviewerID pulid.ID, note string, now int64) {
	m.Status = status
	m.ReviewedBy = reviewerID
	m.ReviewedAt = now
	if note != "" {
		m.ReviewNote = note
	}
}
