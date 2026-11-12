package recordcaseservice

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/recordcase"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

type subject struct {
	head      recordcase.Head
	exhibits  []recordcase.Exhibit
	absorbs   map[watchtower.SourceKind]struct{}
	starters  []string
	unchanged string
	customer  string
}

type subjectRead struct {
	tenant pagination.TenantInfo
	id     pulid.ID
	zone   *time.Location
	site   *time.Location
	marked bool
	now    int64
	items  []*watchtower.Item
	chosen *recordcase.Option
	done   bool
	// stopped is a plan that began and did not finish: what it proposed is
	// no longer a proposal, so the record is drawn as it stands.
	stopped bool
}

type standReader func(
	ctx context.Context,
	tenant pagination.TenantInfo,
	id pulid.ID,
) (recordkind.Kind, pulid.ID, error)

type subjectReader func(ctx context.Context, read *subjectRead) (*subject, error)

func plainSubject(_ context.Context, _ *subjectRead) (*subject, error) {
	return &subject{}, nil
}

func (r *subjectRead) at(stopZone string) *subjectRead {
	site := r.site
	if site == nil {
		site = r.zone
	}
	zone, _ := timeutils.ResolveZone(stopZone, site.String())
	there := *r
	there.zone = zone
	now := time.Unix(r.now, 0)
	_, here := now.In(r.zone).Zone()
	_, away := now.In(zone).Zone()
	there.marked = here != away

	return &there
}

func (r *subjectRead) clock(ts int64) string {
	return r.bare(ts) + r.mark(ts)
}

func (r *subjectRead) mark(ts int64) string {
	if !r.marked {
		return ""
	}

	return " " + time.Unix(ts, 0).In(r.zone).Format("MST")
}

func (r *subjectRead) bare(ts int64) string {
	at := time.Unix(ts, 0).In(r.zone)
	now := time.Unix(r.now, 0).In(r.zone)
	if sameDay(at, now) {
		return at.Format("3:04 PM")
	}
	if diff := at.Sub(now); diff > -6*24*time.Hour && diff < 6*24*time.Hour {
		return at.Format("Mon 3:04 PM")
	}

	return at.Format("Jan 2, 3:04 PM")
}

func (r *subjectRead) day(ts int64) string {
	at := time.Unix(ts, 0).In(r.zone)
	if at.Year() == time.Unix(r.now, 0).In(r.zone).Year() {
		return at.Format("Jan 2")
	}

	return at.Format("Jan 2, 2006")
}

func (r *subjectRead) window(start, end int64) string {
	if end <= 0 || end == start {
		return r.clock(start)
	}
	from := time.Unix(start, 0).In(r.zone)
	to := time.Unix(end, 0).In(r.zone)
	if sameDay(from, to) && from.Format("PM") == to.Format("PM") {
		return strings.TrimSuffix(r.bare(start), from.Format(" PM")) + "–" + to.Format("3:04 PM") +
			r.mark(end)
	}
	if sameDay(from, to) {
		return r.bare(start) + "–" + to.Format("3:04 PM") + r.mark(end)
	}

	return r.bare(start) + "–" + r.clock(end)
}

func sameDay(a, b time.Time) bool {
	return a.Year() == b.Year() && a.YearDay() == b.YearDay()
}

func span(minutes int64) string {
	if minutes < 0 {
		minutes = 0
	}
	hours, rest := minutes/60, minutes%60
	switch {
	case hours == 0:
		return fmt.Sprintf("%dm", rest)
	case rest == 0:
		return fmt.Sprintf("%dh", hours)
	default:
		return fmt.Sprintf("%dh %02dm", hours, rest)
	}
}

func joined(parts ...string) string {
	kept := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			kept = append(kept, part)
		}
	}

	return strings.Join(kept, " · ")
}

func fraction(part, whole int64) float64 {
	if whole <= 0 || part <= 0 {
		return 0
	}
	if part >= whole {
		return 1
	}

	return float64(part) / float64(whole)
}
