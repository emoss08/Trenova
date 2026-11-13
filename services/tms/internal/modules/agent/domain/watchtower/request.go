package watchtower

import (
	"strings"

	"github.com/emoss08/trenova/shared/pulid"
)

// Request is the last play a person sent for an item: who asked, when, and
// what they asked in a sentence. An item keeps it so the next person to
// open the case sees the question is already out.
type Request struct {
	At     int64    `json:"at"`
	ByID   pulid.ID `json:"byId"`
	ByName string   `json:"byName"`
	// Summary is what was asked of whom and how, as one sentence.
	Summary string `json:"summary"`
}

// RecordRequest notes on the item that a play was sent for it and puts the
// item in progress: someone is working it and waits on an answer.
func (i *Item) RecordRequest(by pulid.ID, byName, summary string, now int64) error {
	if err := i.requireOpen(); err != nil {
		return err
	}
	i.LastRequest = &Request{
		At:      now,
		ByID:    by,
		ByName:  strings.TrimSpace(byName),
		Summary: strings.TrimSpace(summary),
	}
	if i.Status != StatusInProgress {
		i.Status = StatusInProgress
		i.SnoozedUntil = nil
		if i.AcknowledgedAt == nil {
			i.AcknowledgedAt = &now
		}
	}

	return nil
}
