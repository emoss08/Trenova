package recordcase

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	MaxVerdictLength = 600
	MaxTrustLength   = 240
	highlightMark    = "=="
	sentenceStops    = ".!?"
	secondsPerHour   = 3600
	maxSegments      = 40
)

type Segment struct {
	Text      string `json:"text,omitempty"`
	Highlight bool   `json:"highlight,omitempty"`
	Cite      int    `json:"cite,omitempty"`
}

type Reading struct {
	keel.Table `keel:"record_case_readings"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:rcr_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	RecordKind  recordkind.Kind `json:"recordKind"  keel:"record_kind"`
	RecordID    pulid.ID        `json:"recordId"    keel:"record_id"`
	Fingerprint string          `json:"fingerprint" keel:"fingerprint"`
	Status      ReadingStatus   `json:"status"      keel:"status"`

	Segments         []Segment `json:"segments"         keel:"segments,jsonb"`
	WhyItMatters     string    `json:"whyItMatters"     keel:"why_it_matters,nullzero"`
	IfNothingChanges string    `json:"ifNothingChanges" keel:"if_nothing_changes,nullzero"`

	ModelIdentifier string   `json:"modelIdentifier" keel:"model_identifier,nullzero"`
	ProviderID      pulid.ID `json:"providerId"      keel:"provider_id,nullzero"`
	RequestedAt     int64    `json:"requestedAt"     keel:"requested_at"`
	WrittenAt       int64    `json:"writtenAt"       keel:"written_at,nullzero"`

	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`
}

func (r *Reading) GetID() pulid.ID { return r.ID }

func (r *Reading) GetTableName() string { return "record_case_readings" }

func (r *Reading) Current(fingerprint string) bool {
	return r != nil && r.Fingerprint == fingerprint
}

func (r *Reading) Text() string {
	var builder strings.Builder
	for _, segment := range r.Segments {
		builder.WriteString(segment.Text)
	}

	return builder.String()
}

func ParseVerdict(marked string, exhibits int) []Segment {
	marked = stringutils.Ellipsize(strings.Join(strings.Fields(marked), " "), MaxVerdictLength)
	marked = spreadStackedCitations(marked)
	segments := make([]Segment, 0, 8)
	var text strings.Builder
	highlight := false
	flush := func() {
		if text.Len() == 0 {
			return
		}
		segments = append(segments, Segment{Text: text.String(), Highlight: highlight})
		text.Reset()
	}

	for i := 0; i < len(marked); {
		if strings.HasPrefix(marked[i:], highlightMark) {
			flush()
			highlight = !highlight
			i += len(highlightMark)

			continue
		}
		if marked[i] == '[' {
			if end := strings.IndexByte(marked[i:], ']'); end > 1 {
				if n, err := strconv.Atoi(marked[i+1 : i+end]); err == nil {
					if n >= 1 && n <= exhibits {
						trimTrailingSpace(&text)
						stop := trailingStop(&text)
						flush()
						segments = append(segments, Segment{Cite: n})
						text.WriteString(stop)
					}
					i += end + 1

					continue
				}
			}
		}
		text.WriteByte(marked[i])
		i++
	}
	flush()
	segments = citeRunsOnce(segments)
	if len(segments) > maxSegments {
		segments = segments[:maxSegments]
	}

	return segments
}

var stackedCitations = regexp.MustCompile(`(?:\s*\[\d{1,2}\]){2,}`)

var oneCitation = regexp.MustCompile(`\[\d{1,2}\]`)

// spreadStackedCitations puts each of several citations written side by side
// at the clause it belongs to. A model that gathers its sources at the end of
// a sentence, "past due, having been due Oct 27 [1][2].", has made its claims
// in the order it cites them, so the citations are dealt back over the
// clauses before them, last to last: "past due [1], having been due Oct 27
// [2]." A clause ends at a comma, a semicolon or a full stop, and only the
// text since the citation before is looked at. Citations with no clause of
// their own stay where they were written.
func spreadStackedCitations(marked string) string {
	for from := 0; from < len(marked); {
		run := stackedCitations.FindStringIndex(marked[from:])
		if run == nil {
			break
		}
		start, end := from+run[0], from+run[1]
		cites := oneCitation.FindAllString(marked[start:end], -1)

		since := strings.LastIndexByte(marked[:start], ']') + 1
		breaks := clauseBreaks(marked[since:start])
		moved := min(len(cites)-1, len(breaks))
		if moved == 0 {
			from = end
			continue
		}

		var out strings.Builder
		out.WriteString(marked[:since])
		span := marked[since:start]
		last := 0
		for i, at := range breaks[len(breaks)-moved:] {
			out.WriteString(span[last:at])
			out.WriteString(" " + cites[len(cites)-1-moved+i])
			last = at
		}
		out.WriteString(span[last:])
		kept := strings.Join(cites[:len(cites)-1-moved], "")
		out.WriteString(" " + kept + cites[len(cites)-1])
		from = out.Len()
		out.WriteString(marked[end:])
		marked = out.String()
	}

	return marked
}

func clauseBreaks(span string) []int {
	breaks := make([]int, 0, 4)
	for i := 0; i+1 < len(span); i++ {
		if strings.IndexByte(",;.", span[i]) >= 0 && span[i+1] == ' ' {
			breaks = append(breaks, i)
		}
	}

	return breaks
}

func citeRunsOnce(segments []Segment) []Segment {
	next := 0
	kept := make([]Segment, 0, len(segments))
	following := make([]int, len(segments))
	for i := len(segments) - 1; i >= 0; i-- {
		following[i] = next
		if segments[i].Cite > 0 {
			next = segments[i].Cite
		}
	}
	for i, segment := range segments {
		if segment.Cite > 0 && segment.Cite == following[i] {
			continue
		}
		last := len(kept) - 1
		if segment.Cite == 0 && last >= 0 && kept[last].Cite == 0 &&
			kept[last].Highlight == segment.Highlight {
			kept[last].Text += segment.Text
			continue
		}
		kept = append(kept, segment)
	}

	return kept
}

func trailingStop(text *strings.Builder) string {
	current := text.String()
	if current == "" || !strings.ContainsRune(sentenceStops, rune(current[len(current)-1])) {
		return ""
	}
	stop := current[len(current)-1:]
	text.Reset()
	text.WriteString(current[:len(current)-1])

	return stop
}

func trimTrailingSpace(text *strings.Builder) {
	current := text.String()
	trimmed := strings.TrimRight(current, " ")
	if len(trimmed) == len(current) {
		return
	}
	text.Reset()
	text.WriteString(trimmed)
}

type Signal struct {
	Kind     string
	SourceID string
	Severity string
	Title    string
	DueAt    *int64
}

func ReadingFingerprint(status string, signals []Signal) string {
	parts := make([]string, 0, len(signals)+1)
	for _, signal := range signals {
		due := ""
		if signal.DueAt != nil {
			due = strconv.FormatInt(*signal.DueAt/secondsPerHour, 10)
		}
		parts = append(parts, strings.Join([]string{
			signal.Kind, signal.SourceID, signal.Severity, signal.Title, due,
		}, "|"))
	}
	parts = append(parts, "status|"+status)

	return Fingerprint(parts)
}

func Fingerprint(parts []string) string {
	sorted := slices.Clone(parts)
	slices.Sort(sorted)
	sum := sha256.Sum256([]byte(strings.Join(sorted, "\x1f")))

	return hex.EncodeToString(sum[:16])
}
