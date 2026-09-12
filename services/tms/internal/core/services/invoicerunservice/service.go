package invoicerunservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/invoicerun"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/services/invoiceservice"
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
	Repo              repositories.InvoiceRunRepository
	BillingQueueRepo  repositories.BillingQueueRepository
	CustomerRepo      repositories.CustomerRepository
	ShipmentRepo      repositories.ShipmentRepository
	InvoiceService    servicesports.InvoiceService
	ConsolidatedMaker *invoiceservice.Service
	SequenceGenerator seqgen.Generator
	AuditService      servicesports.AuditService
	Validator         *Validator
}

type Service struct {
	l                 *zap.Logger
	repo              repositories.InvoiceRunRepository
	billingQueueRepo  repositories.BillingQueueRepository
	customerRepo      repositories.CustomerRepository
	shipmentRepo      repositories.ShipmentRepository
	consolidatedMaker *invoiceservice.Service
	sequenceGenerator seqgen.Generator
	auditService      servicesports.AuditService
	validator         *Validator
}

func New(p Params) *Service {
	return &Service{
		l:                 p.Logger.Named("invoice-run-service"),
		repo:              p.Repo,
		billingQueueRepo:  p.BillingQueueRepo,
		customerRepo:      p.CustomerRepo,
		shipmentRepo:      p.ShipmentRepo,
		consolidatedMaker: p.ConsolidatedMaker,
		sequenceGenerator: p.SequenceGenerator,
		auditService:      p.AuditService,
		validator:         p.Validator,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListInvoiceRunsRequest,
) (*pagination.ListResult[*invoicerun.InvoiceRun], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) ListConnection(
	ctx context.Context,
	req *repositories.ListInvoiceRunConnectionRequest,
) (*pagination.CursorListResult[*invoicerun.InvoiceRun], error) {
	return s.repo.ListConnection(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	req repositories.GetInvoiceRunByIDRequest,
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
	req *servicesports.PreviewInvoiceRunRequest,
	actor *servicesports.RequestActor,
) (*invoicerun.InvoiceRun, error) {
	if req == nil || actor == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request and actor are required",
		)
	}

	customerIDs := req.CustomerIDs
	if len(customerIDs) == 0 {
		return nil, errortypes.NewValidationError(
			"customerIds",
			errortypes.ErrRequired,
			"Select at least one customer to bill",
		)
	}

	number, err := s.sequenceGenerator.Generate(ctx, &seqgen.GenerateRequest{
		Type:  tenant.SequenceTypeInvoiceRun,
		OrgID: req.TenantInfo.OrgID,
		BuID:  req.TenantInfo.BuID,
	})
	if err != nil {
		return nil, err
	}

	invoiceDate := req.InvoiceDate
	if invoiceDate == 0 {
		invoiceDate = timeutils.NowUnix()
	}

	source := req.Source
	if source == "" {
		source = invoicerun.SourceManual
	}

	run := &invoicerun.InvoiceRun{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		Number:         number,
		Status:         invoicerun.StatusBuilding,
		Source:         source,
		Cycle:          req.Cycle,
		PeriodStart:    req.PeriodStart,
		PeriodEnd:      req.PeriodEnd,
		InvoiceDate:    invoiceDate,
		CustomerIDs:    idsToStrings(customerIDs),
		BuiltByID:      actor.UserID,
	}

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

	if err = s.repo.ReplaceGroups(ctx, &repositories.ReplaceGroupsRequest{
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

	return s.repo.GetByID(ctx, repositories.GetInvoiceRunByIDRequest{
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
		&repositories.ListConsolidationCandidatesRequest{
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
	req *servicesports.CancelInvoiceRunRequest,
	actor *servicesports.RequestActor,
) (*invoicerun.InvoiceRun, error) {
	run, err := s.repo.GetByID(ctx, repositories.GetInvoiceRunByIDRequest{
		ID:         req.RunID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	if !invoicerun.CanTransition(run.Status, invoicerun.StatusCanceled) {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			fmt.Sprintf("An invoice run that is %s cannot be canceled", run.Status),
		)
	}

	canceledAt := timeutils.NowUnix()
	run.Status = invoicerun.StatusCanceled
	run.CanceledByID = actor.UserID
	run.CanceledAt = &canceledAt
	run.FailureReason = req.Reason

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
	actor *servicesports.RequestActor,
	op permission.Operation,
	comment string,
) {
	if s.auditService == nil || actor == nil {
		return
	}

	if err := s.auditService.LogAction(
		&servicesports.LogActionParams{
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
