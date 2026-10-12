package triageports

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

var ErrMessageChanged = errors.New("the message changed while it was being written")

type ClaimRuleRequest struct {
	TenantInfo pagination.TenantInfo
	MessageID  pulid.ID
	RuleID     pulid.ID
	Action     inboundmessage.NovaAction
	At         int64
}

type ClaimedRule struct {
	Claimed bool
	Version int64
}

type CorrectClassificationRequest struct {
	TenantInfo     pagination.TenantInfo
	MessageID      pulid.ID
	Classification inboundmessage.Classification
	TargetSeconds  int64
	At             int64
}

type SaveTriageFieldsRequest struct {
	TenantInfo pagination.TenantInfo
	MessageID  pulid.ID
	Version    int64
	Facts      *[]inboundmessage.Fact
	Conflicts  *[]inboundmessage.Conflict
}

type FirstCorrectionRequest struct {
	TenantInfo pagination.TenantInfo
	MessageID  pulid.ID
}

type TriageMessageRepository interface {
	ClaimRule(ctx context.Context, req *ClaimRuleRequest) (*ClaimedRule, error)
	CorrectClassification(ctx context.Context, req *CorrectClassificationRequest) (bool, error)
	SaveTriageFields(ctx context.Context, req *SaveTriageFieldsRequest) error
	OriginalClassification(
		ctx context.Context,
		req FirstCorrectionRequest,
	) (inboundmessage.Classification, bool, error)
}

type UpsertSenderHintRequest struct {
	TenantInfo     pagination.TenantInfo
	SenderDomain   string
	Classification inboundmessage.Classification
	At             int64
}

type ListSenderHintsRequest struct {
	TenantInfo   pagination.TenantInfo
	SenderDomain string
	Limit        int
}

type SenderHintRepository interface {
	Upsert(ctx context.Context, req *UpsertSenderHintRequest) error
	ListByDomain(
		ctx context.Context,
		req ListSenderHintsRequest,
	) ([]*triage.SenderHint, error)
}
