package notification

import (
	"maps"
	"strings"
	"time"
)

const (
	DataKeyOccurrence = "occurrence"
	DataKeyRecent     = "recent"
	DataKeyLink       = "link"

	NoNewOccurrence      = -1
	MaxRecentOccurrences = 3
	TeamBroadcastTTL     = 72 * time.Hour
)

const (
	ResolutionCleared  = "cleared"
	ResolutionResolved = "resolved"
	ResolutionDecided  = "decided"
)

type Occurrence struct {
	Label string `json:"label"`
	Ref   string `json:"ref,omitempty"`
	Link  string `json:"link,omitempty"`
	At    int64  `json:"at"`
}

func SubjectKey(parts ...string) string {
	return strings.Join(parts, ":")
}

func DaySubjectKey(now int64, parts ...string) string {
	day := time.Unix(now, 0).UTC().Format(time.DateOnly)

	return SubjectKey(append(parts, day)...)
}

func (o Occurrence) asData() map[string]any {
	out := map[string]any{"label": o.Label, "at": o.At}
	if o.Ref != "" {
		out["ref"] = o.Ref
	}
	if o.Link != "" {
		out["link"] = o.Link
	}

	return out
}

func (n *Notification) Collapses() bool {
	return n.SubjectKey != "" && n.TargetUserID != nil && n.TargetUserID.IsNotNil()
}

func (n *Notification) WithOccurrence(occurrence Occurrence) *Notification {
	if n.Data == nil {
		n.Data = make(map[string]any, 1)
	}
	n.Data[DataKeyOccurrence] = occurrence.asData()

	return n
}

func (n *Notification) FoldOccurrence(previous map[string]any) {
	occurrence, ok := n.Data[DataKeyOccurrence]
	if !ok {
		return
	}

	data := maps.Clone(n.Data)
	delete(data, DataKeyOccurrence)

	recent := make([]any, 0, MaxRecentOccurrences)
	recent = append(recent, occurrence)
	if earlier, has := previous[DataKeyRecent].([]any); has {
		recent = append(recent, earlier...)
	}
	if len(recent) > MaxRecentOccurrences {
		recent = recent[:MaxRecentOccurrences]
	}
	data[DataKeyRecent] = recent
	n.Data = data
}

func (n *Notification) Absorb(next *Notification, now int64) {
	next.FoldOccurrence(n.Data)

	n.EventType = next.EventType
	n.Priority = next.Priority
	n.Title = next.Title
	n.Message = next.Message
	n.Data = next.Data
	n.RelatedEntities = next.RelatedEntities
	n.Actions = next.Actions
	n.ExpiresAt = next.ExpiresAt
	n.CorrelationID = next.CorrelationID
	n.JobID = next.JobID
	n.Source = next.Source
	n.Tags = next.Tags
	n.DeliveryStatus = next.DeliveryStatus
	n.DeliveredAt = next.DeliveredAt
	if next.Reason != "" {
		n.Reason = next.Reason
	}

	switch {
	case next.OccurrenceCount == NoNewOccurrence:
	case next.OccurrenceCount <= 0:
		n.OccurrenceCount++
	default:
		n.OccurrenceCount += next.OccurrenceCount
	}
	n.ReadAt = nil
	if n.SurfacesAt <= now || next.Priority == PriorityCritical {
		n.SurfacesAt = now
	}
}

func (n *Notification) Snoozed(now int64) bool {
	return n.DismissedAt == nil && n.SurfacesAt > now
}
