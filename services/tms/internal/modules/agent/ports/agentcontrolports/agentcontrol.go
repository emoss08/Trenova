package agentcontrolports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/agentcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
)

type UpdateAgentControlRequest struct {
	ShadowMode bool
	// EarnedAutonomy and PromotionThreshold are optional so a client that
	// only knows the pause switch leaves them as they are.
	EarnedAutonomy     *bool
	PromotionThreshold *int
	AITrainingConsent  *bool
	// PersonMonthlyMessages is absent to leave the allowance as it is.
	PersonMonthlyMessages *int
	LearningOff           *bool
	AutoChaptersOff       *bool
	AgentMoney            AgentMoneyLimitsUpdate
	// Version is the version the person loaded. When it is given and someone
	// saved since, the update is refused with what they changed; absent, the
	// update applies to whatever is there.
	Version    *int64
	TenantInfo pagination.TenantInfo
}

type AgentControlService interface {
	Get(ctx context.Context, tenantInfo pagination.TenantInfo) (*agentcontrol.AgentControl, error)
	Update(
		ctx context.Context,
		req *UpdateAgentControlRequest,
		actor *permissionports.RequestActor,
	) (*agentcontrol.AgentControl, error)
}

type AgentMoneyLimitsUpdate struct {
	CallCeilingMinor *int64
	RunCalls         *int
	RunAmountMinor   *int64
	DayCalls         *int
	DayAmountMinor   *int64
}
