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

	brief, err := s.repo.GetBrief(ctx, tenantInfo)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return &billingqueueports.QueueBriefView{Available: true}, nil
		}
		return nil, err
	}

	return &billingqueueports.QueueBriefView{
		Available:   true,
		Text:        brief.Text,
		GeneratedAt: brief.GeneratedAt,
	}, nil
}

func (s *Service) RefreshBrief(ctx context.Context, tenantInfo pagination.TenantInfo) error {
	available, err := s.Availability(ctx, tenantInfo)
	if err != nil || !available.Brief {
		return err
	}

	facts, err := s.briefFacts(ctx, tenantInfo)
	if err != nil {
		return err
	}
	hash, err := factsHash(facts)
	if err != nil {
		return err
	}

	current, err := s.repo.GetBrief(ctx, tenantInfo)
	if err != nil && !errortypes.IsNotFoundError(err) {
		return err
	}
	if current != nil && current.InputHash == hash {
		return nil
	}

	text := emptyQueueText
	if openCount(facts) > 0 {
		text, err = s.writeBrief(ctx, tenantInfo, facts)
		if err != nil {
			return err
		}
		if text == "" {
			return nil
		}
	}

	return s.repo.SaveBrief(ctx, &billingqueue.QueueBrief{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		Text:           text,
		InputHash:      hash,
		GeneratedAt:    timeutils.NowUnix(),
	})
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

func (s *Service) writeBrief(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	facts *briefFacts,
) (string, error) {
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
		MaxTokens: briefMaxTokens,
		Context: llmports.DelimitedContext{Sections: []llmports.ContextSection{
			{Title: "The queue now", Trusted: true, Content: briefLines(facts)},
		}},
		TenantInfo: tenantInfo,
	}, &answer); err != nil {
		return "", fmt.Errorf("write billing queue brief: %w", err)
	}
	text := strings.TrimSpace(answer.Text)
	if text == "" || len([]rune(text)) > briefRunes {
		return "", nil
	}
	if bad := unknownNumbers(text, allowedNumbers(facts)); len(bad) > 0 {
		s.l.Warn("discarded a billing queue brief with figures it was not given",
			zap.Strings("figures", bad))
		return "", nil
	}

	return text, nil
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

func openCount(facts *briefFacts) int {
	total := 0
	for _, lane := range facts.Lanes {
		total += lane.Count
	}

	return total
}

func factsHash(facts *briefFacts) (string, error) {
	raw, err := sonic.Marshal(facts)
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
