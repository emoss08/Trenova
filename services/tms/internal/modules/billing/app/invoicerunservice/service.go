package invoicerunservice

import (
	"context"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"

	"github.com/emoss08/trenova/internal/modules/billing/app/invoiceservice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoicerun"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoicerunports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/seqgen"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger            *zap.Logger
	Repo              invoicerunports.InvoiceRunRepository
	BillingQueueRepo  billingqueueports.BillingQueueRepository
	CustomerRepo      customerports.CustomerRepository
	ShipmentRepo      shipmentports.ShipmentRepository
	InvoiceService    invoiceports.InvoiceService
	ConsolidatedMaker *invoiceservice.Service
	SequenceGenerator seqgen.Generator
	AuditService      auditports.AuditService
	DetentionBilling  detentionports.DetentionBillingService
	Validator         *Validator
	Notifications     *notificationservice.Service `optional:"true"`
}

type Service struct {
	l                 *zap.Logger
	repo              invoicerunports.InvoiceRunRepository
	billingQueueRepo  billingqueueports.BillingQueueRepository
	customerRepo      customerports.CustomerRepository
	shipmentRepo      shipmentports.ShipmentRepository
	consolidatedMaker *invoiceservice.Service
	sequenceGenerator seqgen.Generator
	auditService      auditports.AuditService
	detentionBilling  detentionports.DetentionBillingService
	validator         *Validator
	notifier          permittedNotifier
}

func New(p Params) *Service {
	svc := &Service{
		l:                 p.Logger.Named("invoice-run-service"),
		repo:              p.Repo,
		billingQueueRepo:  p.BillingQueueRepo,
		customerRepo:      p.CustomerRepo,
		shipmentRepo:      p.ShipmentRepo,
		consolidatedMaker: p.ConsolidatedMaker,
		sequenceGenerator: p.SequenceGenerator,
		auditService:      p.AuditService,
		detentionBilling:  p.DetentionBilling,
		validator:         p.Validator,
	}
	if p.Notifications != nil {
		svc.notifier = p.Notifications
	}

	return svc
}

func (s *Service) List(
	ctx context.Context,
	req *invoicerunports.ListInvoiceRunsRequest,
) (*pagination.ListResult[*invoicerun.InvoiceRun], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) ListConnection(
	ctx context.Context,
	req *invoicerunports.ListInvoiceRunConnectionRequest,
) (*pagination.CursorListResult[*invoicerun.InvoiceRun], error) {
	return s.repo.ListConnection(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	req invoicerunports.GetInvoiceRunByIDRequest,
) (*invoicerun.InvoiceRun, error) {
	return s.repo.GetByID(ctx, req)
}

// Preview builds the proposal an operator reviews.
//
// The run is written first and its groups replaced wholesale, so a re-preview of
// the same period cannot leave a stale group behind, and so a failure partway
// leaves a run somebody can look at rather than nothing at all.
func (s *Service) Preview(
	ctx context.Context,
	req *invoicerunports.PreviewInvoiceRunRequest,
	actor *permissionports.RequestActor,
) (*invoicerun.InvoiceRun, error) {
	if req == nil || actor == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request and actor are required",
		)
	}

	run, customerIDs, err := planRunHeader(req, actor)
	if err != nil {
		return nil, err
	}

	number, err := s.sequenceGenerator.Generate(ctx, &seqgen.GenerateRequest{
		Type:  tenant.SequenceTypeInvoiceRun,
		OrgID: req.TenantInfo.OrgID,
		BuID:  req.TenantInfo.BuID,
	})
	if err != nil {
		return nil, err
	}
	run.Number = number

	if multiErr := s.validator.ValidateCreate(run); multiErr != nil {
		return nil, multiErr
	}

	created, err := s.repo.Create(ctx, run)
	if err != nil {
		return nil, err
	}

	groups, err := s.buildGroups(ctx, created, customerIDs)
	if err != nil {
		return nil, s.failRun(ctx, created, err)
	}

	if err = s.repo.ReplaceGroups(ctx, &invoicerunports.ReplaceGroupsRequest{
		TenantInfo: req.TenantInfo,
		RunID:      created.ID,
		Groups:     groups,
	}); err != nil {
		return nil, s.failRun(ctx, created, err)
	}

	created.Groups = groups
	created.SyncTotals()
	created.Status = invoicerun.StatusReady
	builtAt := timeutils.NowUnix()
	created.BuiltAt = &builtAt

	updated, err := s.repo.Update(ctx, created)
	if err != nil {
		return nil, err
	}

	s.audit(ctx, updated, actor, permission.OpCreate, "Invoice run built")

	return s.repo.GetByID(ctx, invoicerunports.GetInvoiceRunByIDRequest{
		ID:            updated.ID,
		TenantInfo:    req.TenantInfo,
		IncludeGroups: true,
		IncludeItems:  true,
	})
}

// buildGroups turns the eligible queue items into proposed invoices.
func (s *Service) buildGroups(
	ctx context.Context,
	run *invoicerun.InvoiceRun,
	customerIDs []pulid.ID,
) ([]*invoicerun.InvoiceRunGroup, error) {
	candidates, err := s.billingQueueRepo.ListConsolidationCandidates(
		ctx,
		&billingqueueports.ListConsolidationCandidatesRequest{
			TenantInfo:  pagination.TenantInfo{OrgID: run.OrganizationID, BuID: run.BusinessUnitID},
			CustomerIDs: customerIDs,
			PeriodEnd:   run.PeriodEnd,
		},
	)
	if err != nil {
		return nil, err
	}

	proposed := GroupCandidates(candidates)

	groups := make([]*invoicerun.InvoiceRunGroup, 0, len(proposed))
	for _, group := range proposed {
		groups = append(groups, s.newGroup(run, group))
	}

	return groups, nil
}

func (s *Service) newGroup(
	run *invoicerun.InvoiceRun,
	proposed CandidateGroup,
) *invoicerun.InvoiceRunGroup {
	first := proposed.First
	members := proposed.Members

	group := &invoicerun.InvoiceRunGroup{
		OrganizationID: run.OrganizationID,
		BusinessUnitID: run.BusinessUnitID,
		RunID:          run.ID,
		CustomerID:     first.CustomerID,
		GroupKey:       proposed.Key,
		GroupLabel:     proposed.Label,
		SplitBy:        first.SplitBy,
		Status:         invoicerun.GroupStatusPending,
		CurrencyCode:   first.CurrencyCode,
		MinimumAmount:  first.MinConsolidatedAmount,
		AutoBill:       first.AutoBill,
		Items:          make([]*invoicerun.InvoiceRunGroupItem, 0, len(members)),
	}

	for i, member := range members {
		group.Items = append(group.Items, &invoicerun.InvoiceRunGroupItem{
			OrganizationID:     run.OrganizationID,
			BusinessUnitID:     run.BusinessUnitID,
			RunID:              run.ID,
			BillingQueueItemID: member.BillingQueueItemID,
			ShipmentID:         member.ShipmentID,
			OrderID:            member.OrderID,
			ProNumber:          member.ProNumber,
			BOL:                member.ShipmentBOL,
			PONumber:           member.OrderPONumber,
			ServiceDate:        member.ServiceDate,
			SortKey:            i,
			Amount:             member.TotalChargeAmount.Decimal,
		})
	}

	group.SyncTotals()

	return group
}

// Cancel discards a proposal that will never be billed.
func (s *Service) Cancel(
	ctx context.Context,
	req *invoicerunports.CancelInvoiceRunRequest,
	actor *permissionports.RequestActor,
) (*invoicerun.InvoiceRun, error) {
	run, err := s.repo.GetByID(ctx, invoicerunports.GetInvoiceRunByIDRequest{
		ID:         req.RunID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	if err = planCancel(run, req, actor, timeutils.NowUnix()); err != nil {
		return nil, err
	}

	updated, err := s.repo.Update(ctx, run)
	if err != nil {
		return nil, err
	}

	s.audit(ctx, updated, actor, permission.OpCancel, "Invoice run canceled")

	return updated, nil
}

// failRun records why a build could not finish, so the operator sees the reason
// rather than an empty run.
func (s *Service) failRun(
	ctx context.Context,
	run *invoicerun.InvoiceRun,
	cause error,
) error {
	run.Status = invoicerun.StatusFailed
	run.FailureReason = cause.Error()
	if _, err := s.repo.Update(ctx, run); err != nil {
		s.l.Error("failed to record invoice run failure", zap.Error(err))
	}

	return cause
}

func (s *Service) audit(
	ctx context.Context,
	run *invoicerun.InvoiceRun,
	actor *permissionports.RequestActor,
	op permission.Operation,
	comment string,
) {
	if s.auditService == nil || actor == nil {
		return
	}

	if err := s.auditService.LogAction(
		&auditports.LogActionParams{
			Resource:       permission.ResourceInvoiceRun,
			ResourceID:     run.ID.String(),
			Operation:      op,
			UserID:         actor.UserID,
			APIKeyID:       actor.APIKeyID,
			PrincipalType:  actor.PrincipalType,
			PrincipalID:    actor.PrincipalID,
			CurrentState:   jsonutils.MustToJSON(run),
			OrganizationID: run.OrganizationID,
			BusinessUnitID: run.BusinessUnitID,
		},
		auditservice.WithComment(comment),
		auditservice.WithMetadata(map[string]any{
			"runNumber":    run.Number,
			"status":       run.Status,
			"groupCount":   run.GroupCount,
			"invoiceCount": run.InvoiceCount,
		}),
	); err != nil {
		s.l.Error("failed to log invoice run action", zap.Error(err))
	}
}

func tenantOf(run *invoicerun.InvoiceRun) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: run.OrganizationID, BuID: run.BusinessUnitID}
}

func pulidFromString(raw string) (pulid.ID, error) {
	return pulid.MustParse(raw)
}

func idsToStrings(ids []pulid.ID) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, id.String())
	}

	return out
}
