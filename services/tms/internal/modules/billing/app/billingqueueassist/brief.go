package billingqueueassist

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueuereview"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

const (
	briefMaxTokens = 400
	briefRunes     = 700
	briefConcerns  = 5
	briefAttempts  = 2
	emptyQueueText = "Nothing is waiting to be billed."
)

const briefSystem = `Your task: write the one-paragraph brief at the top of a billing queue.

Write two to four plain sentences in the first person for the biller who just opened the queue: how much
is waiting, what is ready, what needs them and why, and anything you questioned. Lead with what they can
clear fastest. Use only the figures you are given, written exactly as given; never compute a new number,
total or percentage. Name a shipment only by the number you are given for it. No headings, lists,
markdown or greetings.`

var numberToken = regexp.MustCompile(`\$?\d[\d,]*(?:\.\d+)?`)

type briefFacts struct {
	Lanes    []*billingqueueports.BillingQueueLaneStat `json:"lanes"`
	Issues   []*billingqueueports.OpenIssueCount       `json:"issues"`
	Concerns []*billingqueueports.AgentConcern         `json:"concerns"`
	Batches  []*billingqueueports.Recommendation       `json:"batches"`
}

type briefAnswer struct {
	Text string `json:"text"`
}

func (s *Service) GetBrief(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*billingqueueports.QueueBriefView, error) {
	available, err := s.Availability(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	if !available.Brief {
		return &billingqueueports.QueueBriefView{}, nil
	}

	lanes, err := s.repo.GetLaneStats(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	lanesNow, err := lanesHash(lanes)
	if err != nil {
		return nil, err
	}
	brief, err := s.repo.GetBrief(ctx, tenantInfo)
	if err != nil && !errortypes.IsNotFoundError(err) {
		return nil, err
	}

	view, current := briefView(brief, lanes, lanesNow, timeutils.NowUnix())
	if !current {
		s.briefs.RequestBrief(ctx, tenantInfo)
	}

	return view, nil
}

// briefView is the brief as the queue stands now, and whether the stored one
// was written for these lanes. An empty queue says so whatever was stored; a
// brief written for other lanes is shown marked stale until its rewrite lands.
func briefView(
	brief *billingqueue.QueueBrief,
	lanes []*billingqueueports.BillingQueueLaneStat,
	lanesNow string,
	now int64,
) (*billingqueueports.QueueBriefView, bool) {
	if brief != nil && brief.LanesHash == lanesNow {
		return &billingqueueports.QueueBriefView{
			Available:   true,
			Text:        brief.Text,
			GeneratedAt: brief.GeneratedAt,
		}, true
	}
	if laneCount(lanes) == 0 {
		generatedAt := now
		if brief != nil {
			generatedAt = brief.GeneratedAt
		}
		return &billingqueueports.QueueBriefView{
			Available:   true,
			Text:        emptyQueueText,
			GeneratedAt: generatedAt,
		}, false
	}
	if brief == nil {
		return &billingqueueports.QueueBriefView{Available: true}, false
	}

	return &billingqueueports.QueueBriefView{
		Available:   true,
		Text:        brief.Text,
		GeneratedAt: brief.GeneratedAt,
		Stale:       true,
	}, false
}

// RefreshBrief writes the brief for the queue as it stands, when its figures
// moved since the last one, and reports whether it wrote one.
func (s *Service) RefreshBrief(ctx context.Context, tenantInfo pagination.TenantInfo) (bool, error) {
	available, err := s.Availability(ctx, tenantInfo)
	if err != nil || !available.Brief {
		return false, err
	}

	facts, err := s.briefFacts(ctx, tenantInfo)
	if err != nil {
		return false, err
	}
	hash, err := factsHash(facts)
	if err != nil {
		return false, err
	}
	lanes, err := lanesHash(facts.Lanes)
	if err != nil {
		return false, err
	}

	current, err := s.repo.GetBrief(ctx, tenantInfo)
	if err != nil && !errortypes.IsNotFoundError(err) {
		return false, err
	}
	if current != nil && current.InputHash == hash && current.LanesHash == lanes {
		return false, nil
	}

	brief := &billingqueue.QueueBrief{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		InputHash:      hash,
		LanesHash:      lanes,
		GeneratedAt:    timeutils.NowUnix(),
	}
	switch {
	case current != nil && current.InputHash == hash:
		brief.Text, brief.GeneratedAt = current.Text, current.GeneratedAt
	case laneCount(facts.Lanes) == 0:
		brief.Text = emptyQueueText
	default:
		brief.Text, err = s.writeBrief(ctx, tenantInfo, facts)
		if err != nil || brief.Text == "" {
			return false, err
		}
	}

	if err = s.repo.SaveBrief(ctx, brief); err != nil {
		return false, err
	}

	return true, nil
}

func (s *Service) briefFacts(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*briefFacts, error) {
	lanes, err := s.repo.GetLaneStats(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	issues, err := s.repo.ListOpenIssueCounts(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	concerns, err := s.repo.ListAgentConcerns(ctx, tenantInfo, briefConcerns)
	if err != nil {
		return nil, err
	}
	batches, err := s.repo.ListRecommendations(ctx, &billingqueueports.ListRecommendationsRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	for _, batch := range batches {
		batch.ItemIDs = nil
	}

	return &briefFacts{Lanes: lanes, Issues: issues, Concerns: concerns, Batches: batches}, nil
}

// writeBrief asks Nova for the brief, and once more with what was wrong when a
// draft uses a figure it was not given or runs long. A brief that still fails is
// written from the figures alone, so the queue never keeps a brief for figures it
// no longer has.
func (s *Service) writeBrief(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	facts *briefFacts,
) (string, error) {
	allowed := allowedNumbers(facts)
	problem := ""
	for range briefAttempts {
		text, err := s.draftBrief(ctx, tenantInfo, facts, problem)
		if err != nil {
			return "", err
		}
		problem = draftProblem(text, allowed)
		if problem == "" {
			return text, nil
		}
		s.l.Warn("rejected a billing queue brief draft", zap.String("problem", problem))
	}

	return factsBrief(facts), nil
}

func (s *Service) draftBrief(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	facts *briefFacts,
	problem string,
) (string, error) {
	sections := []llmports.ContextSection{
		{Title: "The queue now", Trusted: true, Content: briefLines(facts)},
	}
	if problem != "" {
		sections = append(sections, llmports.ContextSection{
			Title:   "What was wrong with your last draft",
			Trusted: true,
			Content: problem,
		})
	}

	var answer briefAnswer
	if _, err := s.nova.Run(ctx, &llmports.NovaTask{
		Name:         aitask.TaskDailyBriefing,
		Feature:      aiusage.FeatureBillingQueueBrief,
		Instructions: briefSystem,
		SchemaName:   "billing_queue_brief",
		Schema: map[string]any{
			"type":                 "object",
			"properties":           map[string]any{"text": map[string]any{"type": "string"}},
			"required":             []string{"text"},
			"additionalProperties": false,
		},
		MaxTokens:  briefMaxTokens,
		Context:    llmports.DelimitedContext{Sections: sections},
		TenantInfo: tenantInfo,
	}, &answer); err != nil {
		return "", fmt.Errorf("write billing queue brief: %w", err)
	}

	return strings.TrimSpace(answer.Text), nil
}

// draftProblem is what keeps a draft from being the brief, said so Nova can fix
// it, or empty when the draft will do.
func draftProblem(text string, allowed map[string]struct{}) string {
	if text == "" {
		return "The draft was empty. Write two to four sentences."
	}
	if len([]rune(text)) > briefRunes {
		return fmt.Sprintf("The draft was too long. Keep it under %d characters.", briefRunes)
	}
	if bad := unknownNumbers(text, allowed); len(bad) > 0 {
		return "The draft used figures you were not given: " + strings.Join(bad, ", ") +
			". Use only the figures listed, written exactly as given; never count, add or compute one."
	}

	return ""
}

// factsBrief is the brief written from the figures alone, for when Nova cannot
// write one that keeps to them.
func factsBrief(facts *briefFacts) string {
	parts := make([]string, 0, len(facts.Lanes))
	for _, lane := range facts.Lanes {
		parts = append(parts, fmt.Sprintf("%s: %s, %s.",
			laneName(lane.Lane), itemCount(lane.Count), billingqueuereview.FormatMoney(lane.Total)))
	}

	return strings.Join(parts, " ")
}

func itemCount(n int) string {
	if n == 1 {
		return "1 item"
	}

	return fmt.Sprintf("%d items", n)
}

func briefLines(facts *briefFacts) string {
	lines := make([]string, 0, 16)
	for _, lane := range facts.Lanes {
		lines = append(lines, fmt.Sprintf("%s: %d items, %s",
			laneName(lane.Lane), lane.Count, billingqueuereview.FormatMoney(lane.Total)))
	}
	for _, issue := range facts.Issues {
		lines = append(lines, fmt.Sprintf("%s: %d items", issueName(issue.Code), issue.Count))
	}
	for _, batch := range facts.Batches {
		lines = append(lines, fmt.Sprintf("Can be done in one step, %s: %d items, %s",
			batchName(batch.Kind), batch.Count, billingqueuereview.FormatMoney(batch.Total)))
	}
	for _, concern := range facts.Concerns {
		lines = append(lines, "You questioned shipment "+concern.Shipment+": "+concern.Summary)
	}

	return strings.Join(lines, "\n")
}

// issueName is what an open check is, said the way a biller would.
func issueName(code billingqueue.IssueCode) string {
	switch code {
	case billingqueue.IssueAccessorialNotOnRateCon:
		return "A charge not on the rate confirmation"
	case billingqueue.IssueChargeOverRateCon:
		return "A charge above the rate confirmation"
	case billingqueue.IssuePODMissing:
		return "No proof of delivery"
	case billingqueue.IssuePODUnsigned:
		return "An unsigned proof of delivery"
	case billingqueue.IssueDetentionELDMismatch:
		return "Detention the ELD does not support"
	case billingqueue.IssueDetentionAwaitingApproval:
		return "Detention waiting on an approver"
	case billingqueue.IssueRatingAwaitingReview:
		return "A rating held for review"
	case billingqueue.IssueChargesUnsplittable:
		return "Charges that cannot be split between payers"
	case billingqueue.IssueBillToCreditHold:
		return "A payer on credit hold"
	case billingqueue.IssuePossibleDuplicate:
		return "A possible duplicate invoice"
	case billingqueue.IssueUnbilledAccessorial:
		return "A receipt no charge bills"
	case billingqueue.IssuePONumberMissing:
		return "A missing PO number"
	default:
		return string(code)
	}
}

// batchName is what one of the queue's one-step batches does.
func batchName(kind billingqueueports.RecommendationKind) string {
	switch kind {
	case billingqueueports.RecommendApproveReady:
		return "approve every item that is ready"
	case billingqueueports.RecommendApproveAgentAgreed:
		return "approve the items you agreed with"
	case billingqueueports.RecommendAssignToMe:
		return "take the unassigned items"
	case billingqueueports.RecommendPostApproved:
		return "post the approved items"
	case billingqueueports.RecommendRequestPaperwork:
		return "ask for the missing paperwork"
	case billingqueueports.RecommendReviewUnbilled:
		return "review receipts no charge bills"
	case billingqueueports.RecommendHoldDuplicates:
		return "hold the possible duplicates"
	case billingqueueports.RecommendReviewAgentConcerns:
		return "review the items you questioned"
	default:
		return string(kind)
	}
}

func laneName(lane billingqueue.Lane) string {
	switch lane {
	case billingqueue.LaneChecking:
		return "Being checked"
	case billingqueue.LaneNeedsYou:
		return "Needs the biller"
	case billingqueue.LaneWaiting:
		return "Waiting on someone else"
	case billingqueue.LaneReady:
		return "Ready to approve"
	case billingqueue.LaneHeld:
		return "On hold"
	case billingqueue.LaneApproved:
		return "Approved, not posted"
	case billingqueue.LaneClosed:
		return "Closed"
	default:
		return string(lane)
	}
}

func laneCount(lanes []*billingqueueports.BillingQueueLaneStat) int {
	total := 0
	for _, lane := range lanes {
		total += lane.Count
	}

	return total
}

func factsHash(facts *briefFacts) (string, error) {
	return hashOf(facts)
}

func lanesHash(lanes []*billingqueueports.BillingQueueLaneStat) (string, error) {
	return hashOf(lanes)
}

func hashOf(v any) (string, error) {
	raw, err := sonic.Marshal(v)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)

	return hex.EncodeToString(sum[:]), nil
}

func allowedNumbers(facts *briefFacts) map[string]struct{} {
	allowed := map[string]struct{}{}
	add := func(text string) {
		for _, token := range numberToken.FindAllString(text, -1) {
			allowed[normalizeNumber(token)] = struct{}{}
		}
	}
	add(briefLines(facts))
	for _, lane := range facts.Lanes {
		add(lane.Total.Round(0).String())
	}
	for _, batch := range facts.Batches {
		add(batch.Total.Round(0).String())
	}

	return allowed
}

func unknownNumbers(text string, allowed map[string]struct{}) []string {
	var bad []string
	for _, token := range numberToken.FindAllString(text, -1) {
		if _, ok := allowed[normalizeNumber(token)]; !ok {
			bad = append(bad, token)
		}
	}

	return bad
}

func normalizeNumber(token string) string {
	clean := strings.NewReplacer("$", "", ",", "").Replace(token)
	if value, err := strconv.ParseFloat(clean, 64); err == nil {
		return strconv.FormatFloat(value, 'f', -1, 64)
	}

	return clean
}
