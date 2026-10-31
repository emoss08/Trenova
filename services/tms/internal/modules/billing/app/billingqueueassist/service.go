package billingqueueassist

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/billingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentcommentports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/accessorialchargeports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         billingqueueports.BillingQueueRepository
	Queue        billingqueueports.BillingQueueService
	Review       billingqueueports.BillingQueueReviewService
	Controls     tenantports.BillingControlRepository
	Customers    customerports.CustomerRepository                   `optional:"true"`
	Nova         llmports.TaskRunner                                `optional:"true"`
	Documents    documentports.DocumentRepository                   `optional:"true"`
	Content      documentports.DocumentContentService               `optional:"true"`
	Accessorials accessorialchargeports.AccessorialChargeRepository `optional:"true"`
	Comments     shipmentcommentports.ShipmentCommentRepository     `optional:"true"`
	Batches      llmports.TaskBatcher                               `optional:"true"`
	Generator    billingqueueports.NovaGenerator                    `optional:"true"`
}

type Service struct {
	l            *zap.Logger
	repo         billingqueueports.BillingQueueRepository
	queue        billingqueueports.BillingQueueService
	review       billingqueueports.BillingQueueReviewService
	controls     tenantports.BillingControlRepository
	customers    customerports.CustomerRepository
	nova         llmports.TaskRunner
	documents    documentports.DocumentRepository
	content      documentports.DocumentContentService
	accessorials accessorialchargeports.AccessorialChargeRepository
	comments     shipmentcommentports.ShipmentCommentRepository
	batches      llmports.TaskBatcher
	generator    billingqueueports.NovaGenerator
}

var _ billingqueueports.BillingQueueAssistService = (*Service)(nil)

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) billingqueueports.BillingQueueAssistService {
	return &Service{
		l:            p.Logger.Named("service.billing-queue-assist"),
		repo:         p.Repo,
		queue:        p.Queue,
		review:       p.Review,
		controls:     p.Controls,
		customers:    p.Customers,
		nova:         p.Nova,
		documents:    p.Documents,
		content:      p.Content,
		accessorials: p.Accessorials,
		comments:     p.Comments,
		batches:      p.Batches,
		generator:    p.Generator,
	}
}

func (s *Service) Availability(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*billingqueueports.AssistAvailability, error) {
	control, err := s.controls.GetByOrgID(ctx, tenantInfo.OrgID)
	if err != nil {
		return nil, err
	}

	reviewing := control.QueueAgentReviewEnabled
	review, err := s.canServe(ctx, aitask.TaskBillingDiagnosis, tenantInfo)
	if err != nil {
		return nil, err
	}
	brief, err := s.canServe(ctx, aitask.TaskDailyBriefing, tenantInfo)
	if err != nil {
		return nil, err
	}
	search, err := s.canServe(ctx, aitask.TaskQueryCompose, tenantInfo)
	if err != nil {
		return nil, err
	}

	return &billingqueueports.AssistAvailability{
		Review: reviewing && review,
		Brief:  reviewing && brief,
		Search: search,
	}, nil
}

func (s *Service) canServe(
	ctx context.Context,
	task aitask.Task,
	tenantInfo pagination.TenantInfo,
) (bool, error) {
	if s.nova == nil {
		return false, nil
	}

	return s.nova.Available(ctx, task, tenantInfo)
}

func (s *Service) AutoApprove(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	itemID pulid.ID,
) (*billingqueueports.AutoApproveOutcome, error) {
	control, err := s.controls.GetByOrgID(ctx, tenantInfo.OrgID)
	if err != nil {
		return nil, err
	}
	if control.QueueAutoApproveMode == "" ||
		control.QueueAutoApproveMode == billingcontrol.QueueAutoApproveModeOff {
		return &billingqueueports.AutoApproveOutcome{Reason: "Auto-approve is off"}, nil
	}

	item, err := s.queue.GetByID(ctx, &billingqueueports.GetBillingQueueItemByIDRequest{
		ItemID:     itemID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if item.Lane != billingqueue.LaneReady || item.BillType != billingqueue.BillTypeInvoice {
		return &billingqueueports.AutoApproveOutcome{Reason: "Not ready"}, nil
	}

	agrees := item.AgentVerdict != nil && *item.AgentVerdict == billingqueue.AgentVerdictAgrees
	if !control.AutoApproves(item.AllocatedTotalAmount, agrees) {
		return &billingqueueports.AutoApproveOutcome{Reason: "Outside the auto-approve rule"}, nil
	}

	result, err := s.review.ApproveIfReady(ctx, &billingqueueports.ApproveIfReadyRequest{
		ItemID:     itemID,
		TenantInfo: tenantInfo,
	}, systemActor(tenantInfo))
	if err != nil {
		return nil, err
	}

	return &billingqueueports.AutoApproveOutcome{
		Approved: result.Approved,
		Reason:   result.Reason,
	}, nil
}

func systemActor(tenantInfo pagination.TenantInfo) *permissionports.RequestActor {
	return &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeSystem,
		PrincipalID:    permissionports.SystemPrincipalID,
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
	}
}
