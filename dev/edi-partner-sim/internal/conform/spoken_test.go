package conform

import (
	"testing"

	"github.com/emoss08/trenova/edi-partner-sim/internal/partner/reply"
)

func TestCarrierRepliesReadAsSentences(t *testing.T) {
	t.Parallel()

	for _, c := range []struct {
		reply    reply.Reply
		scenario reply.Scenario
		want     string
	}{
		{reply.Reply{Set: "990"}, reply.Scenario{Decline: "No driver"},
			"Read Quarrystone Freight Lines' 990 declining the load"},
		{reply.Reply{Set: "214", Want: reply.OutcomeRejected}, reply.Scenario{},
			"Read Quarrystone Freight Lines' 214 for a load Trenova never tendered"},
		{reply.Reply{Set: "210"}, reply.Scenario{DeltaCents: -15000},
			"Read Quarrystone Freight Lines' 210, $150.00 under the tender"},
		{reply.Reply{Set: "210"}, reply.Scenario{},
			"Read Quarrystone Freight Lines' 210 at the tendered total"},
	} {
		if got := replyText("Quarrystone Freight Lines", &c.reply, c.scenario); got != c.want {
			t.Errorf("got %q, want %q", got, c.want)
		}
	}
	if got := (reply.Scenario{DeltaCents: -15000}).Describe(); got !=
		"Invoice $150.00 under the tender" {
		t.Errorf("the case title: %q", got)
	}
	if got := possessive("Kestrel Valve Works"); got != "Kestrel Valve Works'" {
		t.Errorf("possessive: %q", got)
	}
}
