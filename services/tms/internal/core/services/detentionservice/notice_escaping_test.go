package detentionservice_test

import (
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/services/detentionservice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The send path used to build HTML by wrapping the stored plain text in <pre>
// with no escaping, so an angle bracket in a facility or customer name — both
// operator-entered — became markup in a customer's inbox.
//
// Rendering through html/template closes that. This asserts it stays closed.
func TestNoticeHTMLEscapesOperatorEnteredNames(t *testing.T) {
	t.Parallel()

	content, err := detentionservice.RenderBuiltInNotice(
		t.Context(),
		&detentionservice.BuildNoticeParams{
			Occurrence:   noticeOccurrence(),
			Kind:         detention.NoticeKindFinal,
			FacilityName: `Memphis <script>alert(1)</script> DC`,
			ShipmentRef:  `SHP-<img src=x onerror=alert(2)>`,
			Location:     time.UTC,
		},
	)
	require.NoError(t, err)

	// What matters is that no tag the operator typed survives as a tag. The
	// escaped text still contains the characters "onerror=", which is inert —
	// asserting on that substring would be testing the wrong thing.
	assert.NotContains(t, content.HTML, "<script>",
		"a facility name must never reach the inbox as markup")
	assert.NotContains(t, content.HTML, "<img ",
		"a shipment reference must never reach the inbox as a tag")
	assert.Contains(t, content.HTML, "&lt;script&gt;",
		"the name should still be legible, just escaped")
	assert.Contains(t, content.HTML, "&lt;img src=x onerror=alert(2)&gt;",
		"the reference survives as visible text rather than as an element")

	// The plain-text alternative carries the raw characters, which is correct:
	// text/plain has no markup to escape into.
	assert.Contains(t, content.Text, "<script>")

	// And nothing landed somewhere html/template could not escape it.
	assert.NotContains(t, content.HTML, "ZgotmplZ")
}

// A notice queued before body_html existed has no snapshot to send. The
// reconstruction must escape rather than reproduce the old bug.
func TestLegacyNoticeWithoutASnapshotIsEscapedOnSend(t *testing.T) {
	t.Parallel()

	legacy := &detention.DetentionNotice{
		Body: "Detention at Memphis <script>alert(1)</script> DC",
	}

	html := detentionservice.NoticeHTMLForTest(legacy)

	assert.NotContains(t, html, "<script>")
	assert.Contains(t, html, "&lt;script&gt;")
	assert.True(t, strings.HasPrefix(html, "<pre"))
}
