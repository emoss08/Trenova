package agentmessage

import (
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/textcase"
	"github.com/emoss08/trenova/shared/textfence"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	anchorsOpenTag     = "<records_in_play>"
	anchorsCloseTag    = "</records_in_play>"
	maxAnchorValueLen  = 240
	anchorReadAtLayout = "2006-01-02 15:04 MST"
)

const anchorsLead = "These are the records this conversation is about, read again just now, before " +
	"this message. Anything said about them earlier in the conversation, a status, an ETA, " +
	"an amount, a driver, is older than this: where the two differ, this is current, and " +
	"say so if you said otherwise before. A fact marked \"reported … ago\" is what its " +
	"source last reported, not what is true this minute. Read a record with its get tool " +
	"for anything not shown here."

func DescribeAnchors(anchors []agentturn.RuntimeAnchor, now int64, timezone string) string {
	if len(anchors) == 0 {
		return ""
	}
	loc, _ := timeutils.ResolveZone(timezone)

	var b strings.Builder
	b.WriteString(anchorsOpenTag)
	b.WriteString("\nRead at ")
	b.WriteString(time.Unix(now, 0).In(loc).Format(anchorReadAtLayout))
	b.WriteString(". ")
	b.WriteString(anchorsLead)
	b.WriteString("\n")
	for idx := range anchors {
		writeAnchor(&b, &anchors[idx], now)
	}
	b.WriteString(anchorsCloseTag)
	b.WriteString("\n\n")

	return b.String()
}

func writeAnchor(b *strings.Builder, anchor *agentturn.RuntimeAnchor, now int64) {
	b.WriteString("- ")
	b.WriteString(textcase.CapitalizeFirst(recordkind.Kind(anchor.Kind).Noun()))
	if label := stringutils.OneLine(anchor.Label, maxAnchorValueLen); label != "" {
		b.WriteString(" ")
		b.WriteString(textfence.FenceText(label, anchorsCloseTag))
	}
	b.WriteString(" (")
	b.WriteString(anchor.ID)
	b.WriteString(")")
	if anchor.Note != "" {
		b.WriteString(": ")
		b.WriteString(textfence.FenceText(anchor.Note, anchorsCloseTag))
	}
	b.WriteString("\n")
	for _, fact := range anchor.Facts {
		value := stringutils.OneLine(fact.Value, maxAnchorValueLen)
		if value == "" {
			continue
		}
		b.WriteString("  - ")
		b.WriteString(fact.Name)
		b.WriteString(": ")
		b.WriteString(textfence.FenceText(value, anchorsCloseTag))
		if ago := timeutils.DescribeAgo(fact.SeenAt, now); ago != "" {
			b.WriteString(" (reported ")
			b.WriteString(ago)
			if fact.Stale {
				b.WriteString(", too old to plan on")
			}
			b.WriteString(")")
		}
		b.WriteString("\n")
	}
}
