package detentionnoticerender

import (
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/operations/app/detentionengine"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionevidence"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionnotice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detentionpolicy"
	"github.com/emoss08/trenova/shared/pulid"
)

// NoticeContent is the rendered notice, kept separate from delivery so the same
// text can be previewed in the UI before anyone commits to sending it.
type NoticeContent struct {
	Subject string
	Text    string
	HTML    string
	// VersionID names the template version that produced this wording, so a
	// disputed charge can be traced to the exact notice text that was sent. It
	// is nil when the built-in rendered.
	VersionID *pulid.ID
	// PDF is the same notice on paper, rendered from the same context. It is nil
	// unless the governing policy attaches one.
	PDF *NoticePDF
}

// BuildNoticeParams carries everything the notice cites. Every number in the
// message comes from the frozen snapshot rather than live config, so the notice
// and the eventual invoice quote identical terms.
type BuildNoticeParams struct {
	Occurrence    *detention.DetentionOccurrence
	Kind          detentionnotice.NoticeKind
	FacilityName  string
	ShipmentRef   string
	ArrivalSource detentionevidence.EvidenceSource
	Location      *time.Location
	CustomerName  string
	// CustomerID lets a customer with an assigned notice template receive it.
	CustomerID *pulid.ID
	// AttachPDF asks for the notice on paper as well. The caller resolves it from
	// the governing policy rather than the service reading the policy again,
	// because the send path has already loaded it.
	AttachPDF bool
}

func NoticeRateLabel(snap *detentionpolicy.PolicySnapshot) string {
	if snap.RateSource == detentionpolicy.RateSourceTiers && len(snap.Tiers) > 0 {
		parts := make([]string, 0, len(snap.Tiers))
		for _, tier := range snap.Tiers {
			parts = append(parts, fmt.Sprintf(
				"%s at %s",
				detentionengine.TierBandTerm(
					tier,
				),
				detentionengine.RateLabel(tier.Rate, tier.RateUnit),
			))
		}
		return strings.Join(parts, "; ")
	}

	return detentionengine.RateLabel(snap.FlatRate, snap.FlatRateUnit)
}

func formatStamp(ts int64, loc *time.Location) string {
	return time.Unix(ts, 0).In(loc).Format("Mon 02 Jan 2006 15:04 MST")
}
