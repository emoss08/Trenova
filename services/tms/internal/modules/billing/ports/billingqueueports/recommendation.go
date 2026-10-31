package billingqueueports

import (
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type RecommendationKind string

const (
	RecommendApproveReady        = RecommendationKind("ApproveReady")
	RecommendAssignToMe          = RecommendationKind("AssignToMe")
	RecommendPostApproved        = RecommendationKind("PostApproved")
	RecommendRequestPaperwork    = RecommendationKind("RequestPaperwork")
	RecommendReviewUnbilled      = RecommendationKind("ReviewUnbilled")
	RecommendHoldDuplicates      = RecommendationKind("HoldDuplicates")
	RecommendApproveAgentAgreed  = RecommendationKind("ApproveAgentAgreed")
	RecommendReviewAgentConcerns = RecommendationKind("ReviewAgentConcerns")
)

type RecommendationSource string

const (
	RecommendationSourceRules = RecommendationSource("Rules")
	RecommendationSourceAgent = RecommendationSource("Agent")
)

const MaxRecommendationItems = 500

type Recommendation struct {
	Kind    RecommendationKind   `json:"kind"`
	Source  RecommendationSource `json:"source"`
	Count   int                  `json:"count"`
	Total   decimal.Decimal      `json:"total"`
	ItemIDs []pulid.ID           `json:"itemIds"`
}

type ListRecommendationsRequest struct {
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
	CanPost    bool
	CanAssign  bool
}

// RecordAgentReviewRequest records what Nova made of the evidence named by
// InputHash, and what it read to get there. An empty Verdict records that
// this evidence was reviewed and the answer was unusable, so the same evidence
// is not paid for again.
type RecordAgentReviewRequest struct {
	TenantInfo pagination.TenantInfo
	ItemID     pulid.ID
	Verdict    billingqueue.AgentVerdict
	Summary    string
	Reads      []billingqueue.AgentRead
	InputHash  string
	ReviewedAt int64
}

type OpenIssueCount struct {
	Code  billingqueue.IssueCode `json:"code"`
	Count int                    `json:"count"`
}

// AgentConcern is an item Nova questioned, named by its shipment's PRO number
// (the item's own number when it has no shipment), as the queue shows it.
type AgentConcern struct {
	ItemID   pulid.ID `json:"itemId"`
	Shipment string   `json:"shipment"`
	Summary  string   `json:"summary"`
}
