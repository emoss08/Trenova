package inboundmessage

import (
	"strings"

	"github.com/emoss08/trenova/shared/pulid"
)

const (
	MaxThreadKeyLength = 500
	QuietAfterSeconds  = 3 * 24 * 60 * 60
	MaxSnoozeSeconds   = 90 * 24 * 60 * 60
	MaxAwaitSeconds    = 30 * 24 * 60 * 60
)

type Snooze struct {
	Until          int64
	Event          SnoozeEvent
	WatchID        pulid.ID
	DocumentTypeID pulid.ID
	By             pulid.ID
}

type SourceHealth string

const (
	SourceHealthy       = SourceHealth("Healthy")
	SourceQuiet         = SourceHealth("Quiet")
	SourceNeverReceived = SourceHealth("NeverReceived")
	SourcePaused        = SourceHealth("Paused")
)

func (h SourceHealth) Unhealthy() bool {
	return h == SourceQuiet || h == SourceNeverReceived
}

type SourceHealthInput struct {
	Status         MailboxStatus
	CreatedAt      int64
	LastReceivedAt int64
	Now            int64
}

func HealthOf(in SourceHealthInput) SourceHealth {
	if in.Status == MailboxInactive {
		return SourcePaused
	}
	if in.LastReceivedAt == 0 {
		if in.Now-in.CreatedAt > QuietAfterSeconds {
			return SourceNeverReceived
		}

		return SourceHealthy
	}
	if in.Now-in.LastReceivedAt > QuietAfterSeconds {
		return SourceQuiet
	}

	return SourceHealthy
}

func NormalizeMessageID(raw string) string {
	trimmed := strings.TrimSpace(raw)
	trimmed = strings.TrimPrefix(trimmed, "<")
	trimmed = strings.TrimSuffix(trimmed, ">")
	trimmed = strings.ToLower(strings.TrimSpace(trimmed))
	if len(trimmed) > MaxThreadKeyLength {
		trimmed = trimmed[:MaxThreadKeyLength]
	}

	return trimmed
}

func DeriveThreadKey(messageIDHeader string, id pulid.ID) string {
	if normalized := NormalizeMessageID(messageIDHeader); normalized != "" {
		return normalized
	}

	return id.String()
}

func ThreadCandidates(inReplyTo string, references []string) []string {
	candidates := make([]string, 0, len(references)+1)
	seen := make(map[string]struct{}, len(references)+1)
	add := func(raw string) {
		normalized := NormalizeMessageID(raw)
		if normalized == "" {
			return
		}
		if _, ok := seen[normalized]; ok {
			return
		}
		seen[normalized] = struct{}{}
		candidates = append(candidates, normalized)
	}

	add(inReplyTo)
	for i := len(references) - 1; i >= 0; i-- {
		add(references[i])
	}

	return candidates
}
