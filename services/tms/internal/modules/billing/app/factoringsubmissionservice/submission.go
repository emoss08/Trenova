package factoringsubmissionservice

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/billing/app/factoringaction"
	"github.com/emoss08/trenova/internal/modules/billing/app/factoringtransportservice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringreadiness"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringsubmission"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringsubmissionports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/sequenceconfig"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/factoringcompanyports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/seqgen"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	submissionKeyPrefix     = "facsub-"
	maxPlanProblemsInError  = 10
	submissionResource      = permission.ResourceFactoringSubmission
	submissionRealtimeEvent = "updated"
)

type SubmissionParams struct {
	fx.In

	Repo             factoringsubmissionports.SubmissionRepository
	Transports       factoringsubmissionports.TransportRepository
	States           factoringports.InvoiceStateRepository
	Agreements       factoringports.AgreementRepository
	Invoices         invoiceports.InvoiceRepository
	InvoiceService   invoiceports.InvoiceService `optional:"true"`
	InvoiceDocuments documentports.InvoiceDocumentService
	Documents        documentports.DocumentRepository
	Store            documentports.DocumentStore
	Imaging          kernelports.Imaging
	Factors          factoringcompanyports.FactoringCompanyRepository
	Customers        customerports.CustomerRepository
	Organizations    tenantports.OrganizationRepository
	Templates        documenttemplateports.DocumentTemplateResolver
	Inliner          documenttemplateports.AssetInliner `optional:"true"`
	Sequences        seqgen.Generator
	Cipher           kernelports.SecretCipher
	Senders          []factoringsubmissionports.FactoringTransport  `                group:"factoring_transports"`
	Starter          factoringsubmissionports.SubmissionSendStarter `optional:"true"`
	Requester        factoringports.EvaluationRequester             `optional:"true"`
	Events           agentports.AgentEventPublisher                 `optional:"true"`
	Audit            auditports.AuditService
	Realtime         realtimeports.RealtimeService `optional:"true"`
	Logger           *zap.Logger
}

type SubmissionService struct {
	repo             factoringsubmissionports.SubmissionRepository
	transports       factoringsubmissionports.TransportRepository
	states           factoringports.InvoiceStateRepository
	agreements       factoringports.AgreementRepository
	invoices         invoiceports.InvoiceRepository
	invoiceService   invoiceports.InvoiceService
	invoiceDocuments documentports.InvoiceDocumentService
	documents        documentports.DocumentRepository
	store            documentports.DocumentStore
	imaging          kernelports.Imaging
	factors          factoringcompanyports.FactoringCompanyRepository
	customers        customerports.CustomerRepository
	organizations    tenantports.OrganizationRepository
	templates        documenttemplateports.DocumentTemplateResolver
	inliner          documenttemplateports.AssetInliner
	sequences        seqgen.Generator
	cipher           kernelports.SecretCipher
	senders          map[factoringsubmission.TransportMethod]factoringsubmissionports.FactoringTransport
	starter          factoringsubmissionports.SubmissionSendStarter
	requester        factoringports.EvaluationRequester
	events           agentports.AgentEventPublisher
	rec              factoringaction.Recorder
	l                *zap.Logger
	now              func() int64
}

var _ factoringsubmissionports.SubmissionService = (*SubmissionService)(nil)

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func NewSubmissionService(p SubmissionParams) factoringsubmissionports.SubmissionService {
	l := p.Logger.Named("service.factoring-submission")
	return &SubmissionService{
		repo:             p.Repo,
		transports:       p.Transports,
		states:           p.States,
		agreements:       p.Agreements,
		invoices:         p.Invoices,
		invoiceService:   p.InvoiceService,
		invoiceDocuments: p.InvoiceDocuments,
		documents:        p.Documents,
		store:            p.Store,
		imaging:          p.Imaging,
		factors:          p.Factors,
		customers:        p.Customers,
		organizations:    p.Organizations,
		templates:        p.Templates,
		inliner:          p.Inliner,
		sequences:        p.Sequences,
		cipher:           p.Cipher,
		senders:          factoringtransportservice.TransportsByMethod(p.Senders),
		starter:          p.Starter,
		requester:        p.Requester,
		events:           p.Events,
		rec:              factoringaction.Recorder{Audit: p.Audit, Realtime: p.Realtime, Logger: l},
		l:                l,
		now:              timeutils.NowUnix,
	}
}

func (s *SubmissionService) Get(
	ctx context.Context,
	req factoringsubmissionports.GetSubmissionRequest,
) (*factoringsubmission.Submission, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *SubmissionService) List(
	ctx context.Context,
	req *factoringsubmissionports.ListSubmissionConnectionRequest,
) (*pagination.CursorListResult[*factoringsubmission.Submission], error) {
	return s.repo.ListConnection(ctx, req)
}

func (s *SubmissionService) ListItems(
	ctx context.Context,
	req *factoringsubmissionports.ListItemsBySubmissionIDsRequest,
) (map[pulid.ID][]*factoringsubmission.SubmissionItem, error) {
	return s.repo.ListItemsBySubmissionIDs(ctx, req)
}

func (s *SubmissionService) LiveItems(
	ctx context.Context,
	req *factoringsubmissionports.ListLiveItemsRequest,
) (map[pulid.ID]*factoringsubmissionports.LiveItem, error) {
	return s.repo.ListLiveItems(ctx, req)
}

type plannedSubmission struct {
	plan      *factoringsubmissionports.SubmissionPlan
	factor    *factoringcompany.FactoringCompany
	transport *factoringsubmission.Transport
	invoices  map[pulid.ID]*invoice.Invoice
}

func (s *SubmissionService) Plan(
	ctx context.Context,
	req *factoringsubmissionports.PlanSubmissionRequest,
) (*factoringsubmissionports.SubmissionPlan, error) {
	planned, err := s.plan(ctx, req)
	if err != nil {
		return nil, err
	}
	return planned.plan, nil
}

func (s *SubmissionService) plan(
	ctx context.Context,
	req *factoringsubmissionports.PlanSubmissionRequest,
) (*plannedSubmission, error) {
	if err := factoringaction.RequireRequest(req != nil); err != nil {
		return nil, err
	}
	ids, err := submissionInvoiceIDs(req.InvoiceIDs)
	if err != nil {
		return nil, err
	}
	factor, err := s.factors.GetByID(ctx, factoringcompanyports.GetFactoringCompanyByIDRequest{
		ID:         req.FactoringCompanyID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if !factor.IsActive() {
		return nil, errortypes.NewBusinessError("{0} is inactive", factor.Name)
	}
	transport, err := s.transports.Get(ctx, factoringsubmissionports.GetTransportRequest{
		TenantInfo:         req.TenantInfo,
		FactoringCompanyID: factor.ID,
	})
	if errortypes.IsNotFoundError(err) {
		return nil, errortypes.NewBusinessError(
			"Set up how schedules reach {0} on the factor's page before submitting",
			factor.Name,
		)
	}
	if err != nil {
		return nil, err
	}
	now := s.now()
	agreement, err := s.agreements.GetInForce(ctx, factoringports.GetAgreementInForceRequest{
		TenantInfo:         req.TenantInfo,
		FactoringCompanyID: factor.ID,
		At:                 now,
	})
	if errortypes.IsNotFoundError(err) {
		return nil, errortypes.NewBusinessError(
			"{0} has no agreement in force today; activate one before submitting",
			factor.Name,
		)
	}
	if err != nil {
		return nil, err
	}

	states, err := s.states.ListByInvoiceIDs(
		ctx,
		&factoringports.ListInvoiceStatesByInvoiceIDsRequest{
			TenantInfo: req.TenantInfo,
			InvoiceIDs: ids,
		},
	)
	if err != nil {
		return nil, err
	}
	invoices, err := s.loadInvoices(ctx, req.TenantInfo, ids)
	if err != nil {
		return nil, err
	}
	names, err := s.customerNames(ctx, req.TenantInfo, invoices)
	if err != nil {
		return nil, err
	}

	plan := &factoringsubmissionports.SubmissionPlan{
		FactoringCompanyID:   factor.ID,
		FactoringCompanyName: factor.Name,
		Agreement:            agreement,
		TransportMethod:      transport.Method,
		TransportTarget:      transport.Target(),
		AttachPackets: transport.Method == factoringsubmission.TransportMethodSftp ||
			transport.AttachPackets,
		Items: make([]*factoringsubmissionports.SubmissionPlanItem, 0, len(ids)),
	}
	keyParts, problems := fillPlanItems(plan, &planInputs{
		ids:       ids,
		invoices:  invoices,
		states:    states,
		names:     names,
		agreement: agreement,
		factorID:  factor.ID,
		now:       now,
	})
	if len(problems) > 0 {
		return nil, planError(problems)
	}
	plan.InvoiceCount = len(plan.Items)
	plan.IdempotencyKey = submissionKey(keyParts)

	return &plannedSubmission{
		plan:      plan,
		factor:    factor,
		transport: transport,
		invoices:  invoices,
	}, nil
}

type planInputs struct {
	ids       []pulid.ID
	invoices  map[pulid.ID]*invoice.Invoice
	states    map[pulid.ID]*factoringreadiness.InvoiceState
	names     map[pulid.ID]string
	agreement *factoring.Agreement
	factorID  pulid.ID
	now       int64
}

func fillPlanItems(
	plan *factoringsubmissionports.SubmissionPlan,
	in *planInputs,
) (keyParts, problems []string) {
	problems = make([]string, 0, 4)
	keyParts = make([]string, 0, len(in.ids)+1)
	keyParts = append(keyParts, in.factorID.String())
	for _, id := range in.ids {
		inv := in.invoices[id]
		state := in.states[id]
		if problem := planProblem(id, inv, state, in.factorID); problem != "" {
			problems = append(problems, problem)
			continue
		}
		if plan.CurrencyCode == "" {
			plan.CurrencyCode = inv.CurrencyCode
		}
		if inv.CurrencyCode != plan.CurrencyCode {
			problems = append(problems, inv.Number+" is in "+inv.CurrencyCode+
				", not "+plan.CurrencyCode)
			continue
		}
		funding := factoring.ExpectedFunding(
			inv.OpenBalanceMinor(),
			in.agreement,
			factoring.DaysToRun(inv.DueDate, in.now),
		)
		plan.Items = append(plan.Items, &factoringsubmissionports.SubmissionPlanItem{
			InvoiceID:     inv.ID,
			InvoiceNumber: inv.Number,
			CustomerID:    inv.CustomerID,
			CustomerName:  in.names[inv.CustomerID],
			InvoiceDate:   inv.InvoiceDate,
			DueDate:       inv.DueDate,
			Funding:       funding,
		})
		plan.FaceMinor += funding.FaceMinor
		plan.ExpectedAdvanceMinor += funding.AdvanceMinor
		plan.ExpectedFeeMinor += funding.FeeMinor
		plan.ExpectedReserveMinor += funding.ReserveMinor
		keyParts = append(keyParts, id.String()+"@"+strconv.FormatInt(*state.ReadySince, 10))
	}
	return keyParts, problems
}

func (s *SubmissionService) Submit(
	ctx context.Context,
	req *factoringsubmissionports.PlanSubmissionRequest,
	actor *permissionports.RequestActor,
) (*factoringsubmission.Submission, error) {
	if err := factoringaction.RequireActor(actor, "Submitting invoices to a factor"); err != nil {
		return nil, err
	}
	planned, err := s.plan(ctx, req)
	if err != nil {
		return nil, err
	}
	plan := planned.plan

	existing, err := s.repo.GetByIdempotencyKey(
		ctx,
		factoringsubmissionports.GetSubmissionByKeyRequest{
			TenantInfo:     req.TenantInfo,
			IdempotencyKey: plan.IdempotencyKey,
		},
	)
	if err == nil {
		return existing, nil
	}
	if !errortypes.IsNotFoundError(err) {
		return nil, err
	}

	number, err := s.sequences.Generate(ctx, &seqgen.GenerateRequest{
		Type:  sequenceconfig.SequenceTypeFactoringSubmission,
		OrgID: req.TenantInfo.OrgID,
		BuID:  req.TenantInfo.BuID,
	})
	if err != nil {
		return nil, err
	}

	entity := &factoringsubmission.Submission{
		OrganizationID:     req.TenantInfo.OrgID,
		BusinessUnitID:     req.TenantInfo.BuID,
		Number:             number,
		FactoringCompanyID: plan.FactoringCompanyID,
		AgreementID:        plan.Agreement.ID,
		Status:             factoringsubmission.SubmissionStatusSending,
		CurrencyCode:       plan.CurrencyCode,
		TransportMethod:    plan.TransportMethod,
		TransportTarget:    plan.TransportTarget,
		IdempotencyKey:     plan.IdempotencyKey,
		SubmittedByID:      actor.UserID,
		AgentRunID:         req.AgentRunID,
		SendAttemptGroup:   1,
		Items:              itemsFromPlan(plan),
	}
	entity.Totalize()
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if err = factoringaction.ValidationOf(multiErr); err != nil {
		return nil, err
	}

	created, err := s.repo.Create(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.rec.Log(actor, &factoringaction.AuditEntry{
		Resource: submissionResource,
		Entity:   created,
		Op:       permission.OpSubmit,
		Comment: "Schedule " + created.Number + " of " + strconv.Itoa(created.InvoiceCount) +
			" invoices submitted to " + planned.factor.Name,
	})
	s.rec.Publish(ctx, submissionResource, created, actor, "created")
	s.requestInvoices(ctx, req.TenantInfo, created)
	s.startSend(ctx, req.TenantInfo, created)
	return created, nil
}

func (s *SubmissionService) PlanRetry(
	ctx context.Context,
	req *factoringsubmissionports.SubmissionActionRequest,
) (*factoringsubmissionports.SubmissionActionPlan, error) {
	before, err := s.loadForAction(ctx, req)
	if err != nil {
		return nil, err
	}
	transport, err := s.transports.Get(ctx, factoringsubmissionports.GetTransportRequest{
		TenantInfo:         req.TenantInfo,
		FactoringCompanyID: before.FactoringCompanyID,
	})
	if errortypes.IsNotFoundError(err) {
		return nil, errortypes.NewBusinessError(
			"The factor has no delivery settings; set them up before retrying",
		)
	}
	if err != nil {
		return nil, err
	}
	after := factoringaction.CloneSubmission(before)
	if err = after.Retry(); err != nil {
		return nil, err
	}
	after.TransportMethod = transport.Method
	after.TransportTarget = transport.Target()
	return &factoringsubmissionports.SubmissionActionPlan{Before: before, After: after}, nil
}

func (s *SubmissionService) Retry(
	ctx context.Context,
	req *factoringsubmissionports.SubmissionActionRequest,
	actor *permissionports.RequestActor,
) (*factoringsubmission.Submission, error) {
	if err := factoringaction.RequireActor(actor, "Retrying a factoring schedule"); err != nil {
		return nil, err
	}
	plan, err := s.PlanRetry(ctx, req)
	if err != nil {
		return nil, err
	}
	updated, err := s.repo.Update(ctx, plan.After)
	if err != nil {
		return nil, err
	}
	s.rec.Log(actor, &factoringaction.AuditEntry{
		Resource: submissionResource,
		Entity:   updated,
		Previous: plan.Before,
		Op:       permission.OpUpdate,
		Comment:  "Schedule " + updated.Number + " sent again to " + updated.TransportTarget,
	})
	s.rec.Publish(ctx, submissionResource, updated, actor, submissionRealtimeEvent)
	s.startSend(ctx, req.TenantInfo, updated)
	return updated, nil
}

func (s *SubmissionService) PlanCancel(
	ctx context.Context,
	req *factoringsubmissionports.SubmissionActionRequest,
) (*factoringsubmissionports.SubmissionActionPlan, error) {
	before, err := s.loadForAction(ctx, req)
	if err != nil {
		return nil, err
	}
	after := factoringaction.CloneSubmission(before)
	if err = after.Cancel(s.now(), req.Reason); err != nil {
		return nil, err
	}
	return &factoringsubmissionports.SubmissionActionPlan{Before: before, After: after}, nil
}

func (s *SubmissionService) Cancel(
	ctx context.Context,
	req *factoringsubmissionports.SubmissionActionRequest,
	actor *permissionports.RequestActor,
) (*factoringsubmission.Submission, error) {
	if err := factoringaction.RequireActor(actor, "Cancelling a factoring schedule"); err != nil {
		return nil, err
	}
	plan, err := s.PlanCancel(ctx, req)
	if err != nil {
		return nil, err
	}
	updated, err := s.repo.Update(ctx, plan.After)
	if err != nil {
		return nil, err
	}
	s.rec.Log(actor, &factoringaction.AuditEntry{
		Resource: submissionResource,
		Entity:   updated,
		Previous: plan.Before,
		Op:       permission.OpCancel,
		Comment:  "Schedule " + updated.Number + " cancelled: " + updated.CancelReason,
	})
	s.rec.Publish(ctx, submissionResource, updated, actor, submissionRealtimeEvent)
	s.requestInvoices(ctx, req.TenantInfo, updated)
	return updated, nil
}

func (s *SubmissionService) loadForAction(
	ctx context.Context,
	req *factoringsubmissionports.SubmissionActionRequest,
) (*factoringsubmission.Submission, error) {
	if err := factoringaction.RequireRequest(req != nil); err != nil {
		return nil, err
	}
	current, err := s.repo.GetByID(ctx, factoringsubmissionports.GetSubmissionRequest{
		ID:           req.SubmissionID,
		TenantInfo:   req.TenantInfo,
		IncludeItems: true,
	})
	if err != nil {
		return nil, err
	}
	if current.Version != req.Version {
		return nil, errortypes.NewConflictError(
			"Schedule {0} changed since you opened it; reload and try again",
			current.Number,
		)
	}
	return current, nil
}

func (s *SubmissionService) loadInvoices(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	ids []pulid.ID,
) (map[pulid.ID]*invoice.Invoice, error) {
	found, err := s.invoices.GetByIDs(ctx, invoiceports.GetInvoicesByIDsRequest{
		TenantInfo: tenantInfo,
		InvoiceIDs: ids,
	})
	if err != nil {
		return nil, err
	}
	out := make(map[pulid.ID]*invoice.Invoice, len(found))
	for _, inv := range found {
		if inv != nil {
			out[inv.ID] = inv
		}
	}
	return out, nil
}

func (s *SubmissionService) customerNames(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	invoices map[pulid.ID]*invoice.Invoice,
) (map[pulid.ID]string, error) {
	ids := make([]pulid.ID, 0, len(invoices))
	seen := make(map[pulid.ID]struct{}, len(invoices))
	for _, inv := range invoices {
		if _, dup := seen[inv.CustomerID]; dup {
			continue
		}
		seen[inv.CustomerID] = struct{}{}
		ids = append(ids, inv.CustomerID)
	}
	names := make(map[pulid.ID]string, len(ids))
	if len(ids) == 0 {
		return names, nil
	}
	found, err := s.customers.GetByIDs(ctx, customerports.GetCustomersByIDsRequest{
		TenantInfo:  tenantInfo,
		CustomerIDs: ids,
	})
	if err != nil {
		return nil, err
	}
	for _, cus := range found {
		if cus != nil {
			names[cus.ID] = cus.DisplayName()
		}
	}
	return names, nil
}

func (s *SubmissionService) requestInvoices(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	sub *factoringsubmission.Submission,
) {
	if s.requester == nil {
		return
	}
	s.requester.RequestEvaluation(ctx, tenantInfo, &factoringports.EvaluationTargets{
		InvoiceIDs: sub.InvoiceIDs(),
	})
}

func (s *SubmissionService) startSend(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	sub *factoringsubmission.Submission,
) {
	if s.starter == nil {
		s.l.Warn("factoring schedule sending is not available in this process",
			zap.String("submissionId", sub.ID.String()))
		return
	}
	s.starter.StartSend(ctx, tenantInfo, sub)
}

func (s *SubmissionService) publishEvent(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	sub *factoringsubmission.Submission,
	kind agenttrigger.EventKind,
	detail string,
) {
	agentports.PublishAgentEvent(ctx, s.events, agentports.AgentEvent{
		Kind:       kind,
		SubjectID:  sub.ID,
		TenantInfo: tenantInfo,
		Related:    []pulid.ID{sub.FactoringCompanyID},
		Detail:     detail,
	})
}

func submissionInvoiceIDs(raw []pulid.ID) ([]pulid.ID, error) {
	ids := factoringaction.UniqueIDs(raw)
	if len(ids) == 0 {
		return nil, errortypes.NewValidationError(
			"invoiceIds",
			errortypes.ErrRequired,
			"Choose at least one invoice",
		)
	}
	if len(ids) > factoringsubmission.MaxSubmissionInvoices {
		return nil, errortypes.NewValidationError(
			"invoiceIds",
			errortypes.ErrInvalid,
			"A schedule can hold at most 200 invoices",
		)
	}
	return ids, nil
}

func planProblem(
	id pulid.ID,
	inv *invoice.Invoice,
	state *factoringreadiness.InvoiceState,
	factorID pulid.ID,
) string {
	switch {
	case inv == nil:
		return id.String() + " was not found"
	case inv.FactoringCompanyID == nil || *inv.FactoringCompanyID != factorID:
		return inv.Number + " is not assigned to this factor"
	case state == nil:
		return inv.Number + " has not been checked for factoring yet"
	case state.Verdict == factoringreadiness.VerdictSubmitted:
		return inv.Number + " is already on a schedule"
	case state.Verdict != factoringreadiness.VerdictReady || state.ReadySince == nil:
		return inv.Number + " is not ready to factor"
	default:
		return ""
	}
}

func planError(problems []string) error {
	shown := problems
	if len(shown) > maxPlanProblemsInError {
		shown = shown[:maxPlanProblemsInError]
	}
	message := strings.Join(shown, "; ")
	if extra := len(problems) - len(shown); extra > 0 {
		message += "; and " + strconv.Itoa(extra) + " more"
	}
	return errortypes.NewBusinessError("These invoices cannot be submitted: {0}", message)
}

func submissionKey(parts []string) string {
	slices.Sort(parts[1:])
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return submissionKeyPrefix + hex.EncodeToString(sum[:16])
}

func itemsFromPlan(
	plan *factoringsubmissionports.SubmissionPlan,
) []*factoringsubmission.SubmissionItem {
	items := make([]*factoringsubmission.SubmissionItem, 0, len(plan.Items))
	for _, item := range plan.Items {
		items = append(items, &factoringsubmission.SubmissionItem{
			InvoiceID:            item.InvoiceID,
			InvoiceNumber:        item.InvoiceNumber,
			CustomerID:           item.CustomerID,
			FaceMinor:            item.Funding.FaceMinor,
			ExpectedAdvanceMinor: item.Funding.AdvanceMinor,
			ExpectedFeeMinor:     item.Funding.FeeMinor,
			ExpectedReserveMinor: item.Funding.ReserveMinor,
			ExpectedDays:         item.Funding.Days,
			Status:               factoringsubmission.SubmissionItemStatusPending,
		})
	}
	return items
}
