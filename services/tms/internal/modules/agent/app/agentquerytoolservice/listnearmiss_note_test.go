package agentquerytoolservice

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

/*
"What was delivered today?" found nothing early in the morning, and the note
told the agent to loosen the date. It dropped the date and answered another
question, then proposed keeping that as a lesson. The term ruling every row
out is named, and the note says that when the person asked for it, none
matching is the answer.
*/
func TestNearMissNote_SaysNoneIsTheAnswerWhenThePersonAskedForIt(t *testing.T) {
	t.Parallel()

	note := nearMissNote("shipments", "actualDeliveryDate is today",
		[]string{"actualDeliveryDate is today"})

	assert.Equal(t,
		"No shipments matched actualDeliveryDate is today. Without just one of these, some "+
			"do: actualDeliveryDate is today. If the person asked for it, none matching is the "+
			"answer: say so rather than searching without it. Loosen it only when you chose "+
			"it yourself, and loosen it rather than trying new values one by one.",
		note,
	)

	together := nearMissNote("shipments", "status one of New and actualDeliveryDate is today", nil)
	assert.Contains(t, together, "together they rule everything out")
	assert.Contains(t, together, "If the person asked for all of these, none matching is the answer")
}
