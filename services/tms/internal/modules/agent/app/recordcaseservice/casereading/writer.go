package casereading

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/modules/agent/app/recordcaseservice"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/domain/recordcase"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/recordcaseports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/numberguard"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	RealtimeResource  = "desk-cases"
	maxOutputTokens   = 500
	maxCitations      = 99
	maxAttempts       = 2
	highlightMark     = "=="
	maxDecidingLength = 120
	maxPhraseWords    = 8
	digits            = "0123456789"
	structuralRunes   = "{}[]<>=\"`"
)

type Params struct {
	fx.In

	Logger   *zap.Logger
	Nova     llmports.TaskRunner
	Composer *recordcaseservice.Composer
	Repo     recordcaseports.RecordCaseRepository
	Realtime realtimeports.RealtimeService
}

type Service struct {
	l        *zap.Logger
	nova     llmports.TaskRunner
	composer *recordcaseservice.Composer
	repo     recordcaseports.RecordCaseRepository
	realtime realtimeports.RealtimeService
	now      func() int64
}

func New(p Params) recordcaseports.ReadingWriter {
	return &Service{
		l:        p.Logger.Named("service.case-reading"),
		nova:     p.Nova,
		composer: p.Composer,
		repo:     p.Repo,
		realtime: p.Realtime,
		now:      timeutils.NowUnix,
	}
}

type draft struct {
	Verdict          string  `json:"verdict"`
	DecidingFact     string  `json:"decidingFact"`
	WhyItMatters     *string `json:"whyItMatters"`
	IfNothingChanges *string `json:"ifNothingChanges"`
}

func (s *Service) WriteReading(
	ctx context.Context,
	req *recordcaseports.WriteReadingRequest,
) error {
	tenant := req.TenantInfo
	tenant.UserID = req.UserID
	actor := permissionports.UserActor(tenant)
	ctx = dbscope.WithTenant(ctx, actor.DBTenant())

	composed, err := s.composer.Compose(ctx, &recordcaseports.CaseRequest{
		TenantInfo: tenant,
		Kind:       req.RecordKind,
		ID:         req.RecordID,
		Timezone:   req.Timezone,
	}, actor)
	if err != nil {
		if errortypes.IsNotFoundError(err) || errortypes.IsAuthorizationError(err) {
			return s.save(
				ctx,
				req,
				&recordcase.Reading{Status: recordcase.ReadingStatusUnavailable},
			)
		}

		return err
	}
	if composed.Fingerprint != req.Fingerprint {
		return nil
	}

	reading := s.write(ctx, req, composed)

	return s.save(ctx, req, reading)
}

func (s *Service) write(
	ctx context.Context,
	req *recordcaseports.WriteReadingRequest,
	composed *recordcaseservice.Reading,
) *recordcase.Reading {
	unavailable := &recordcase.Reading{Status: recordcase.ReadingStatusUnavailable}
	if len(composed.Exhibits) == 0 {
		return unavailable
	}

	supported := supportedBy(composed)
	evidence := evidenceOf(composed)
	correction := ""
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		reading, redo, err := s.attempt(ctx, req, composed, supported, evidence, correction)
		if err != nil {
			if errors.Is(err, llmports.ErrNoProviderConfigured) {
				s.l.Info("no provider configured for case readings; the case shows what was computed")
			} else {
				s.l.Warn("a case reading could not be written", zap.Error(err))
			}

			return unavailable
		}
		if reading != nil {
			return reading
		}
		if redo == "" {
			return unavailable
		}
		s.l.Warn("refused a case reading",
			zap.Int("attempt", attempt),
			zap.String("correction", redo),
		)
		correction = "\n\n" + redo
	}

	return unavailable
}

func (s *Service) attempt(
	ctx context.Context,
	req *recordcaseports.WriteReadingRequest,
	composed *recordcaseservice.Reading,
	supported []decimal.Decimal,
	evidence []string,
	correction string,
) (reading *recordcase.Reading, redo string, err error) {
	var answer draft
	completion, err := s.nova.Run(ctx, &llmports.NovaTask{
		Name:         aitask.TaskCaseReading,
		Feature:      aiusage.FeatureCaseReading,
		Instructions: systemPrompt + correction,
		Context:      buildContext(composed),
		Schema:       outputSchema(),
		SchemaName:   "case_reading",
		MaxTokens:    maxOutputTokens,
		TenantInfo:   req.TenantInfo,
		UserID:       req.UserID,
	}, &answer)
	if err != nil {
		return nil, "", err
	}
	if trailsOff(answer.Verdict) {
		return nil, unfinishedCorrection, nil
	}

	verdict := marked(answer.Verdict, answer.DecidingFact, evidence)
	if verdict == "" || utf8.RuneCountInString(verdict) > recordcase.MaxVerdictLength ||
		strings.ContainsAny(verdict, "{}<>\"`") {
		return nil, "", nil
	}
	if check := numberguard.CheckEvery(unmarked(verdict), supported); !check.OK {
		return nil, unsupportedCorrection(check.Unsupported), nil
	}

	segments := recordcase.ParseVerdict(verdict, len(composed.Exhibits))
	// Asked once: a second answer that still leaves a message out is kept,
	// since a reading without it is better than no reading.
	if correction == "" {
		if unread := unreadMessages(segments, composed.Exhibits); len(unread) > 0 {
			return nil, unreadCorrection(unread), nil
		}
	}

	reading = &recordcase.Reading{
		Status:          recordcase.ReadingStatusReady,
		Segments:        segments,
		ModelIdentifier: completion.ModelIdentifier,
		ProviderID:      completion.ProviderID,
		WrittenAt:       s.now(),
	}
	reading.WhyItMatters = guarded(answer.WhyItMatters, supported)
	reading.IfNothingChanges = guarded(answer.IfNothingChanges, supported)

	return reading, "", nil
}

const unfinishedCorrection = "Your last answer was refused because the verdict trailed off " +
	"in an ellipsis. Write the verdict again as whole sentences that end in a full stop, and " +
	"fill in whyItMatters and ifNothingChanges."

func unsupportedCorrection(figures []string) string {
	return "Your last answer was refused because it stated these figures, which no " +
		"exhibit holds: " + strings.Join(figures, ", ") + ". You worked them out or spelled " +
		"them as words. Write the verdict again using only figures printed in the exhibits, " +
		"in digits, and leave out any claim that needs a figure you were not given."
}

// unreadMessages are the numbers of the message exhibits the verdict cites
// nowhere. What a customer or a carrier wrote is the one kind of evidence a
// reading may not pass over: it is what the company was told, and a person
// deciding the case without it decides on less than the company knows.
func unreadMessages(segments []recordcase.Segment, exhibits []recordcase.Exhibit) []int {
	cited := make(map[int]struct{}, len(segments))
	for _, segment := range segments {
		if segment.Cite > 0 {
			cited[segment.Cite] = struct{}{}
		}
	}

	unread := make([]int, 0, len(exhibits))
	for idx := range exhibits {
		if exhibits[idx].Kind != recordcase.ExhibitMessage {
			continue
		}
		if _, ok := cited[exhibits[idx].Number]; !ok {
			unread = append(unread, exhibits[idx].Number)
		}
	}

	return unread
}

func unreadCorrection(unread []int) string {
	numbers := make([]string, 0, len(unread))
	for _, n := range unread {
		numbers = append(numbers, "["+strconv.Itoa(n)+"]")
	}

	return "Your last answer left out what a customer or a carrier wrote, in exhibit " +
		strings.Join(numbers, ", ") + ". Write the verdict again and say in a clause what " +
		"each of those messages tells the company, with its citation, keeping every other rule."
}

func trailsOff(verdict string) bool {
	verdict = strings.TrimSpace(verdict)

	return strings.HasSuffix(verdict, "...") || strings.HasSuffix(verdict, "…")
}

func marked(verdict, deciding string, evidence []string) string {
	verdict = strings.Join(strings.Fields(strings.ReplaceAll(verdict, highlightMark, "")), " ")
	deciding = strings.Trim(strings.Join(strings.Fields(deciding), " "), " .,;:!?")
	if utf8.RuneCountInString(deciding) > maxDecidingLength ||
		strings.ContainsAny(deciding, structuralRunes) {
		deciding = ""
	}
	at, length := foldedIndex(verdict, deciding), len(deciding)
	if at < 0 {
		at, length = evidencePhrase(verdict, evidence)
	}
	if at < 0 {
		return verdict
	}

	return verdict[:at] + highlightMark + verdict[at:at+length] + highlightMark +
		verdict[at+length:]
}

func foldedIndex(text, phrase string) int {
	if phrase == "" {
		return -1
	}
	lowered, wanted := strings.ToLower(text), strings.ToLower(phrase)
	if len(lowered) != len(text) || len(wanted) != len(phrase) {
		return strings.Index(text, phrase)
	}

	return strings.Index(lowered, wanted)
}

func evidencePhrase(verdict string, evidence []string) (at, length int) {
	lowered := make([]string, 0, len(evidence))
	for _, text := range evidence {
		lowered = append(lowered, strings.ToLower(text))
	}

	type word struct{ start, end int }
	words := make([]word, 0, 48)
	for i := 0; i < len(verdict); {
		if verdict[i] == ' ' {
			i++
			continue
		}
		end := i
		for end < len(verdict) && verdict[end] != ' ' {
			end++
		}
		words = append(words, word{start: i, end: end})
		i = end
	}

	at, length = -1, 0
	for first := range words {
		for last := min(first+maxPhraseWords, len(words)) - 1; last > first; last-- {
			phrase := strings.TrimRight(verdict[words[first].start:words[last].end], ".,;:!?")
			if len(phrase) <= length || strings.ContainsAny(phrase, "[]") ||
				!strings.ContainsAny(phrase, digits) ||
				slices.Contains(looseEnds, strings.ToLower(verdict[words[last].start:words[last].end])) ||
				(first > 0 && hasDigit(verdict[words[first-1].start:words[first-1].end])) ||
				(last+1 < len(words) && hasDigit(verdict[words[last+1].start:words[last+1].end])) {
				continue
			}
			folded := strings.ToLower(phrase)
			if len(folded) != len(phrase) {
				continue
			}
			if slices.ContainsFunc(lowered, func(text string) bool {
				return strings.Contains(text, folded)
			}) {
				at, length = words[first].start, len(phrase)
			}
		}
	}

	return at, length
}

var looseEnds = []string{
	"is", "are", "was", "were", "has", "have", "and", "or", "of", "the", "a", "an", "at", "to",
	"for", "with", "in", "on", "by", "its", "still",
}

func hasDigit(word string) bool {
	return strings.ContainsAny(word, digits)
}

func evidenceOf(composed *recordcaseservice.Reading) []string {
	evidence := make([]string, 0, len(composed.Exhibits)*4+len(composed.Signals))
	for i := range composed.Exhibits {
		evidence = append(evidence, composed.Exhibits[i].Claims...)
	}

	return append(evidence, composed.Signals...)
}

func aboutTheEvidence(text string) bool {
	lowered := strings.ToLower(text)
	for _, word := range []string{"the facts", "exhibit", "evidence", "does not say"} {
		if strings.Contains(lowered, word) {
			return true
		}
	}

	return false
}

func firstSentence(text string) string {
	for i, r := range text {
		if !strings.ContainsRune(".!?", r) {
			continue
		}
		rest := text[i+utf8.RuneLen(r):]
		if rest == "" || rest[0] == ' ' || !isWordRune(rest) {
			return text[:i+utf8.RuneLen(r)]
		}
		if next, _ := utf8.DecodeRuneInString(rest); unicode.IsLetter(next) && i > 0 &&
			unicode.IsLower(rune(text[i-1])) && unicode.IsLower(next) && len(rest) > 2 &&
			rest[1] != '.' {
			return text[:i+utf8.RuneLen(r)]
		}
	}

	return text
}

func isWordRune(text string) bool {
	r, _ := utf8.DecodeRuneInString(text)

	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

func guarded(written *string, supported []decimal.Decimal) string {
	if written == nil {
		return ""
	}
	text := *written
	text = firstSentence(strings.Join(strings.Fields(text), " "))
	if text == "" || aboutTheEvidence(text) || utf8.RuneCountInString(text) > recordcase.MaxTrustLength ||
		strings.ContainsAny(text, structuralRunes) ||
		!numberguard.CheckEvery(text, supported).OK {
		return ""
	}

	return text
}

func unmarked(verdict string) string {
	segments := recordcase.ParseVerdict(verdict, maxCitations)
	var builder strings.Builder
	for _, segment := range segments {
		builder.WriteString(segment.Text)
		if segment.Cite > 0 {
			builder.WriteByte(' ')
		}
	}

	return builder.String()
}

func supportedBy(composed *recordcaseservice.Reading) []decimal.Decimal {
	texts := make([]string, 0, len(composed.Exhibits)*4+len(composed.Signals)+3)
	texts = append(texts, composed.Label, composed.Title, composed.Status, composed.Proposal)
	texts = append(texts, composed.Signals...)
	for i := range composed.Exhibits {
		texts = append(texts, composed.Exhibits[i].Claims...)
	}

	return numberguard.SupportedFromText(texts...)
}

func (s *Service) save(
	ctx context.Context,
	req *recordcaseports.WriteReadingRequest,
	reading *recordcase.Reading,
) error {
	reading.RecordKind = req.RecordKind
	reading.RecordID = req.RecordID
	reading.Fingerprint = req.Fingerprint
	if reading.Segments == nil {
		reading.Segments = []recordcase.Segment{}
	}
	if err := s.repo.SaveReading(ctx, reading); err != nil {
		return err
	}

	if err := realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		Resource:       RealtimeResource,
		Action:         "updated",
		RecordID:       req.RecordID,
	}); err != nil {
		s.l.Warn("a written case reading could not be announced", zap.Error(err))
	}

	return nil
}
