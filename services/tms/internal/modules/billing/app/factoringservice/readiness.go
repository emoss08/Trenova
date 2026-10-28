package factoringservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	shipmentResourceType = "shipment"
	invoiceResourceType  = "invoice"
	maxEvaluateBatch     = 200
	sweepBatchSize       = 200
	maxSweepInvoices     = 20000
	maxOverrideReason    = 500
	maxAnnouncements     = 50
)

type ReadinessParams struct {
	fx.In

	States        factoringports.InvoiceStateRepository
	Agreements    factoringports.AgreementRepository
	Noas          factoringports.NoaRepository
	Submissions   factoringports.SubmissionRepository
	Attention     factoringports.WatchtowerReader `optional:"true"`
	Invoices      invoiceports.InvoiceRepository
	Documents     documentports.DocumentRepository
	Shipments     shipmentports.ShipmentService `optional:"true"`
	Customers     customerports.CustomerRepository
	Organizations tenantports.OrganizationRepository
	Templates     documenttemplateports.DocumentTemplateResolver
	Inliner       documenttemplateports.AssetInliner `optional:"true"`
	Email         emailports.EmailService
	Events        agentports.AgentEventPublisher `optional:"true"`
	Audit         auditports.AuditService
	Realtime      realtimeports.RealtimeService `optional:"true"`
	Logger        *zap.Logger
}

type ReadinessService struct {
	states        factoringports.InvoiceStateRepository
	agreements    factoringports.AgreementRepository
	noas          factoringports.NoaRepository
	submissions   factoringports.SubmissionRepository
	attention     factoringports.WatchtowerReader
	invoices      invoiceports.InvoiceRepository
	documents     documentports.DocumentRepository
	shipments     shipmentports.ShipmentService
	customers     customerports.CustomerRepository
	organizations tenantports.OrganizationRepository
	templates     documenttemplateports.DocumentTemplateResolver
	inliner       documenttemplateports.AssetInliner
	email         emailports.EmailService
	events        agentports.AgentEventPublisher
	rec           recorder
	l             *zap.Logger
	now           func() int64
}

var _ factoringports.ReadinessService = (*ReadinessService)(nil)

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func NewReadinessService(p ReadinessParams) factoringports.ReadinessService {
	l := p.Logger.Named("service.factoring-readiness")
	return &ReadinessService{
		states:        p.States,
		agreements:    p.Agreements,
		noas:          p.Noas,
		submissions:   p.Submissions,
		attention:     p.Attention,
		invoices:      p.Invoices,
		documents:     p.Documents,
		shipments:     p.Shipments,
		customers:     p.Customers,
		organizations: p.Organizations,
		templates:     p.Templates,
		inliner:       p.Inliner,
		email:         p.Email,
		events:        p.Events,
		rec:           recorder{audit: p.Audit, realtime: p.Realtime, l: l},
		l:             l,
		now:           timeutils.NowUnix,
	}
}

func (s *ReadinessService) GetReadiness(
	ctx context.Context,
	req factoringports.GetReadinessRequest,
) (*factoring.InvoiceState, error) {
	inv, err := s.invoices.GetByID(ctx, invoiceports.GetInvoiceByIDRequest{
		ID:         req.InvoiceID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	state, _, err := s.evaluate(ctx, req.TenantInfo, inv)
	return state, err
}

func (s *ReadinessService) Evaluate(
	ctx context.Context,
	req *factoringports.EvaluateInvoicesRequest,
) (*factoringports.EvaluateResult, error) {
	result := &factoringports.EvaluateResult{}
	ids := uniqueIDs(req.InvoiceIDs)
	for start := 0; start < len(ids); start += maxEvaluateBatch {
		end := min(start+maxEvaluateBatch, len(ids))
		for _, id := range ids[start:end] {
			s.evaluateOne(ctx, req.TenantInfo, id, result)
		}
	}
	return result, nil
}

func (s *ReadinessService) evaluateOne(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	invoiceID pulid.ID,
	result *factoringports.EvaluateResult,
) {
	inv, err := s.invoices.GetByID(ctx, invoiceports.GetInvoiceByIDRequest{
		ID:         invoiceID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		if !errortypes.IsNotFoundError(err) {
			result.Failed++
			s.l.Warn("could not load an invoice to evaluate for factoring",
				zap.String("invoiceId", invoiceID.String()), zap.Error(err))
		}
		return
	}
	_, transition, err := s.evaluate(ctx, tenantInfo, inv)
	if err != nil {
		result.Failed++
		s.l.Warn("factoring readiness evaluation failed",
			zap.String("invoiceId", invoiceID.String()), zap.Error(err))
		return
	}
	result.Evaluated++
	if transition.BecameReady {
		result.BecameReady++
	}
	if transition.LeftReady {
		result.LeftReady++
	}
	if transition.announced {
		result.Announced++
	}
}

type evaluation struct {
	factoring.StateTransition
	announced bool
}

func (s *ReadinessService) evaluate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	inv *invoice.Invoice,
) (*factoring.InvoiceState, evaluation, error) {
	now := s.now()
	readiness, err := s.assess(ctx, tenantInfo, inv, now)
	if err != nil {
		return nil, evaluation{}, err
	}

	previous, err := s.states.Get(ctx, factoringports.GetInvoiceStateRequest{
		TenantInfo: tenantInfo,
		InvoiceID:  inv.ID,
	})
	if err != nil && !errortypes.IsNotFoundError(err) {
		return nil, evaluation{}, err
	}
	if err != nil {
		previous = nil
	}

	next := &factoring.InvoiceState{
		InvoiceID:          inv.ID,
		OrganizationID:     inv.OrganizationID,
		BusinessUnitID:     inv.BusinessUnitID,
		FactoringCompanyID: inv.FactoringCompanyID,
		CustomerID:         inv.CustomerID,
		OpenBalanceMinor:   inv.OpenBalanceMinor(),
	}
	result := evaluation{StateTransition: factoring.NextState(previous, next, &readiness, now)}
	if err = s.states.Upsert(ctx, next); err != nil {
		return nil, evaluation{}, err
	}

	if next.Verdict == factoring.VerdictReady && next.ReadyAnnouncedAt == nil {
		result.announced = s.announce(ctx, tenantInfo, next, inv, now)
	}
	if result.BecameReady || result.LeftReady || previous == nil ||
		previous.FailedChecks != next.FailedChecks || previous.Verdict != next.Verdict {
		s.rec.publish(ctx, permission.ResourceInvoice, next, nil, "factoring_readiness")
	}
	return next, result, nil
}

func (s *ReadinessService) assess(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	inv *invoice.Invoice,
	now int64,
) (factoring.Readiness, error) {
	inputs := &factoring.ReadinessInputs{At: now, Invoice: invoiceFacts(inv)}
	readiness := factoring.EvaluateReadiness(inputs)
	if readiness.Verdict == factoring.VerdictExcluded {
		return readiness, nil
	}
	if err := s.gather(ctx, tenantInfo, inv, inputs); err != nil {
		return factoring.Readiness{}, err
	}
	return factoring.EvaluateReadiness(inputs), nil
}

func (s *ReadinessService) announce(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	state *factoring.InvoiceState,
	inv *invoice.Invoice,
	now int64,
) bool {
	claimed, err := s.states.ClaimReadyAnnouncement(
		ctx,
		&factoringports.ClaimReadyAnnouncementRequest{
			TenantInfo: tenantInfo,
			InvoiceID:  state.InvoiceID,
			ReadySince: *state.ReadySince,
			At:         now,
		},
	)
	if err != nil {
		s.l.Warn("could not claim the factoring ready announcement",
			zap.String("invoiceId", state.InvoiceID.String()), zap.Error(err))
		return false
	}
	if !claimed {
		return false
	}
	announcedAt := now
	state.ReadyAnnouncedAt = &announcedAt

	related := make([]pulid.ID, 0, 2)
	if state.FactoringCompanyID != nil {
		related = append(related, *state.FactoringCompanyID)
	}
	related = append(related, state.CustomerID)
	agentports.PublishAgentEvent(ctx, s.events, agentports.AgentEvent{
		Kind:       agent.EventFactoringInvoiceReady,
		SubjectID:  state.InvoiceID,
		TenantInfo: tenantInfo,
		Related:    related,
		Detail:     "Invoice " + inv.Number + " is ready to factor",
	})
	return true
}

func (s *ReadinessService) gather(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	inv *invoice.Invoice,
	in *factoring.ReadinessInputs,
) error {
	factorID := *inv.FactoringCompanyID

	agreement, err := s.agreements.GetInForce(ctx, factoringports.GetAgreementInForceRequest{
		TenantInfo:         tenantInfo,
		FactoringCompanyID: factorID,
		At:                 in.At,
	})
	if err != nil && !errortypes.IsNotFoundError(err) {
		return err
	}
	if err == nil {
		in.Agreement = agreement
		in.AgreementDocuments = agreementNeeds(agreement)
	}

	noa, err := s.noas.GetOpen(ctx, &factoringports.GetOpenNoaRequest{
		TenantInfo:         tenantInfo,
		CustomerID:         inv.CustomerID,
		FactoringCompanyID: factorID,
	})
	if err != nil && !errortypes.IsNotFoundError(err) {
		return err
	}
	if err == nil {
		in.Noa = noa
	}

	in.OpenDispute = inv.DisputeStatus == invoice.DisputeStatusDisputed

	items, err := s.submissions.ListLiveItems(ctx, &factoringports.ListLiveItemsRequest{
		TenantInfo:         tenantInfo,
		InvoiceIDs:         []pulid.ID{inv.ID},
		IncludeChargedBack: true,
	})
	if err != nil {
		return err
	}
	if item := items[inv.ID]; item != nil {
		if item.IsLive() {
			in.SubmissionNumber = item.SubmissionNumber
			in.SubmissionItemStatus = item.ItemStatus
		} else {
			in.ChargedBack = item.ItemStatus == factoring.SubmissionItemStatusChargedBack
		}
	}

	legs := inv.LegShipmentIDs()
	accepted, err := s.acceptedTypes(ctx, tenantInfo, inv.ID, legs, in.At)
	if err != nil {
		return err
	}
	in.InvoiceAcceptedTypeIDs = accepted[inv.ID]
	in.Shipments = make([]factoring.ShipmentFacts, 0, len(legs))
	pros := legPros(inv)
	for _, shipmentID := range legs {
		facts := factoring.ShipmentFacts{
			ShipmentID:      shipmentID,
			ProNumber:       pros[shipmentID],
			AcceptedTypeIDs: accepted[shipmentID],
		}
		s.fillBillingReadiness(ctx, tenantInfo, &facts)
		in.Shipments = append(in.Shipments, facts)
	}
	return nil
}

func (s *ReadinessService) liveItem(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	invoiceID pulid.ID,
) (*factoringports.LiveItem, error) {
	live, err := s.submissions.ListLiveItems(ctx, &factoringports.ListLiveItemsRequest{
		TenantInfo: tenantInfo,
		InvoiceIDs: []pulid.ID{invoiceID},
	})
	if err != nil {
		return nil, err
	}
	return live[invoiceID], nil
}

func (s *ReadinessService) fillBillingReadiness(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	facts *factoring.ShipmentFacts,
) {
	if s.shipments == nil {
		return
	}
	readiness, err := s.shipments.GetBillingReadiness(ctx, facts.ShipmentID, tenantInfo)
	if err != nil {
		s.l.Warn("could not read a shipment's billing readiness for factoring",
			zap.String("shipmentId", facts.ShipmentID.String()), zap.Error(err))
		return
	}
	facts.Available = true
	for _, req := range readiness.MissingRequirements {
		typeID, parseErr := pulid.Parse(req.DocumentTypeID)
		if parseErr != nil {
			typeID = pulid.Nil
		}
		name := req.DocumentTypeName
		if name == "" {
			name = req.DocumentTypeCode
		}
		facts.MissingDocuments = append(facts.MissingDocuments, factoring.DocumentNeed{
			TypeID: typeID,
			Name:   name,
		})
	}
	for _, failure := range readiness.ValidationFailures {
		facts.ValidationFailures = append(facts.ValidationFailures, factoring.ShipmentValidation{
			Code:    failure.Code,
			Message: failure.Message,
		})
	}
}

func (s *ReadinessService) acceptedTypes(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	invoiceID pulid.ID,
	shipmentIDs []pulid.ID,
	now int64,
) (map[pulid.ID]map[pulid.ID]struct{}, error) {
	out := make(map[pulid.ID]map[pulid.ID]struct{}, len(shipmentIDs)+1)
	collect := func(resourceType string, ids []pulid.ID) error {
		if len(ids) == 0 {
			return nil
		}
		resourceIDs := make([]string, 0, len(ids))
		for _, id := range ids {
			resourceIDs = append(resourceIDs, id.String())
		}
		docs, err := s.documents.GetByResourceIDs(
			ctx,
			&documentports.GetDocumentsByResourceIDsRequest{
				TenantInfo:   tenantInfo,
				ResourceType: resourceType,
				ResourceIDs:  resourceIDs,
			},
		)
		if err != nil {
			return err
		}
		for _, doc := range docs {
			if doc == nil || doc.DocumentTypeID == nil ||
				doc.StandingAt(now) != document.StandingAccepted {
				continue
			}
			owner, parseErr := pulid.Parse(doc.ResourceID)
			if parseErr != nil {
				continue
			}
			if out[owner] == nil {
				out[owner] = make(map[pulid.ID]struct{}, 4)
			}
			out[owner][*doc.DocumentTypeID] = struct{}{}
		}
		return nil
	}
	if err := collect(shipmentResourceType, shipmentIDs); err != nil {
		return nil, err
	}
	if err := collect(invoiceResourceType, []pulid.ID{invoiceID}); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *ReadinessService) ListStates(
	ctx context.Context,
	req *factoringports.ListInvoiceStateConnectionRequest,
) (*pagination.CursorListResult[*factoring.InvoiceState], error) {
	return s.states.ListConnection(ctx, req)
}

func (s *ReadinessService) GetStates(
	ctx context.Context,
	req *factoringports.ListInvoiceStatesByInvoiceIDsRequest,
) (map[pulid.ID]*factoring.InvoiceState, error) {
	return s.states.ListByInvoiceIDs(ctx, req)
}

func (s *ReadinessService) Summary(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*factoringports.Summary, error) {
	now := s.now()
	return s.states.GetSummary(ctx, factoringports.GetSummaryRequest{
		TenantInfo:       tenantInfo,
		ReadyBefore:      now - factoring.ReadyUnsubmittedSeconds,
		ChargebackBefore: now + int64(factoring.ChargebackWarningDays)*timeutils.SecondsPerDay,
	})
}

func (s *ReadinessService) SetOverride(
	ctx context.Context,
	req *factoringports.SetOverrideRequest,
	actor *permissionports.RequestActor,
) (*factoring.InvoiceState, error) {
	if err := requireActor(actor, "Changing whether an invoice is factored"); err != nil {
		return nil, err
	}
	plan, err := s.PlanOverride(ctx, req)
	if err != nil {
		return nil, err
	}
	if !plan.Changed {
		state, _, evalErr := s.evaluate(ctx, req.TenantInfo, plan.Before)
		return state, evalErr
	}

	if err = s.states.SetInvoiceOverride(ctx, &factoringports.SetInvoiceOverrideRequest{
		TenantInfo:           req.TenantInfo,
		InvoiceID:            plan.After.ID,
		Override:             plan.After.FactoringOverride,
		Reason:               plan.After.FactoringOverrideReason,
		FactoringCompanyID:   plan.After.FactoringCompanyID,
		FactoringAgreementID: plan.After.FactoringAgreementID,
	}); err != nil {
		return nil, err
	}
	plan.After.Version++

	comment := "Invoice follows the customer's factoring again"
	if req.Override.Excludes() {
		comment = "Invoice kept out of factoring: " + strings.TrimSpace(req.Reason)
	}
	s.rec.log(actor, &auditEntry{
		resource: permission.ResourceInvoice,
		entity:   plan.After,
		previous: plan.Before,
		op:       permission.OpUpdate,
		comment:  comment,
	})
	state, _, err := s.evaluate(ctx, req.TenantInfo, plan.After)
	if err != nil {
		return nil, err
	}
	s.rec.publish(ctx, permission.ResourceInvoice, plan.After, actor, "updated")
	return state, nil
}

func (s *ReadinessService) PlanOverride(
	ctx context.Context,
	req *factoringports.SetOverrideRequest,
) (*factoringports.OverridePlan, error) {
	if err := requireRequest(req != nil); err != nil {
		return nil, err
	}
	if !req.Override.IsValid() {
		return nil, errortypes.NewValidationError(
			"override",
			errortypes.ErrInvalid,
			"Choose whether the invoice follows the customer's factoring or stays out of it",
		)
	}
	reason := strings.TrimSpace(req.Reason)
	if req.Override.Excludes() && reason == "" {
		return nil, errortypes.NewValidationError(
			"reason",
			errortypes.ErrRequired,
			"Say why this invoice stays out of factoring",
		)
	}
	if len(reason) > maxOverrideReason {
		return nil, errortypes.NewValidationError(
			"reason",
			errortypes.ErrInvalid,
			"Keep the reason under 500 characters",
		)
	}

	inv, err := s.invoices.GetByID(ctx, invoiceports.GetInvoiceByIDRequest{
		ID:         req.InvoiceID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if inv.Status == invoice.StatusVoided {
		return nil, errortypes.NewBusinessError("A voided invoice cannot be changed")
	}
	if inv.IsCreditMemo() {
		return nil, errortypes.NewBusinessError("A credit memo is never factored")
	}
	if inv.FactoringOverride == req.Override {
		return &factoringports.OverridePlan{Before: inv, After: inv}, nil
	}
	live, err := s.liveItem(ctx, req.TenantInfo, inv.ID)
	if err != nil {
		return nil, err
	}
	if live != nil {
		return nil, errortypes.NewBusinessError(
			"Invoice {0} is on factoring schedule {1}; cancel the schedule before changing whether it is factored",
			inv.Number,
			live.SubmissionNumber,
		)
	}

	factorID, agreementID, err := s.stampFor(ctx, req.TenantInfo, inv, req.Override)
	if err != nil {
		return nil, err
	}
	after := *inv
	after.FactoringOverride = req.Override
	after.FactoringOverrideReason = ""
	if req.Override.Excludes() {
		after.FactoringOverrideReason = reason
	}
	after.FactoringCompanyID = factorID
	after.FactoringAgreementID = agreementID
	return &factoringports.OverridePlan{Before: inv, After: &after, Changed: true}, nil
}

func (s *ReadinessService) stampFor(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	inv *invoice.Invoice,
	override invoice.FactoringOverride,
) (factorID, agreementID *pulid.ID, err error) {
	if override.Excludes() {
		return nil, nil, nil
	}
	if inv.Status != invoice.StatusPosted {
		return inv.FactoringCompanyID, inv.FactoringAgreementID, nil
	}

	found, err := s.customers.GetByIDs(ctx, customerports.GetCustomersByIDsRequest{
		TenantInfo:  tenantInfo,
		CustomerIDs: []pulid.ID{inv.CustomerID},
		CustomerFilterOptions: customerports.CustomerFilterOptions{
			IncludeBillingProfile: true,
		},
	})
	if err != nil {
		return nil, nil, err
	}
	for _, cus := range found {
		if cus != nil && cus.ID == inv.CustomerID && cus.BillingProfile != nil &&
			cus.BillingProfile.FactoringCompanyID != nil &&
			!cus.BillingProfile.FactoringCompanyID.IsNil() {
			id := *cus.BillingProfile.FactoringCompanyID
			factorID = &id
		}
	}
	if factorID == nil {
		return nil, nil, nil
	}

	at := s.now()
	if inv.PostedAt != nil {
		at = *inv.PostedAt
	}
	agreement, err := s.agreements.GetInForce(ctx, factoringports.GetAgreementInForceRequest{
		TenantInfo:         tenantInfo,
		FactoringCompanyID: *factorID,
		At:                 at,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return factorID, nil, nil
		}
		return nil, nil, err
	}
	stamped := agreement.ID
	return factorID, &stamped, nil
}

func (s *ReadinessService) ListSweepTenants(ctx context.Context) ([]pagination.TenantInfo, error) {
	return s.states.ListSweepTenants(ctx)
}

func (s *ReadinessService) SweepTenant(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*factoringports.SweepResult, error) {
	result := &factoringports.SweepResult{}
	after := pulid.Nil
	for seen := 0; seen < maxSweepInvoices; {
		ids, err := s.states.ListCandidateInvoiceIDs(
			ctx,
			factoringports.ListCandidateInvoicesRequest{
				TenantInfo:     tenantInfo,
				AfterInvoiceID: after,
				Limit:          sweepBatchSize,
			},
		)
		if err != nil {
			return nil, err
		}
		if len(ids) == 0 {
			break
		}
		evaluated, err := s.Evaluate(ctx, &factoringports.EvaluateInvoicesRequest{
			TenantInfo: tenantInfo,
			InvoiceIDs: ids,
		})
		if err != nil {
			return nil, err
		}
		result.Add(evaluated)
		seen += len(ids)
		after = ids[len(ids)-1]
		if len(ids) < sweepBatchSize {
			break
		}
	}

	warned, err := s.warnExpiringNoas(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	result.NoasWarned = warned

	if result.ChargebacksWarned, err = s.warnChargebacksDue(ctx, tenantInfo); err != nil {
		return nil, err
	}
	return result, nil
}

func (s *ReadinessService) warnChargebacksDue(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (int, error) {
	if s.attention == nil {
		return 0, nil
	}
	now := s.now()
	due, err := s.attention.ListChargebacksDue(ctx, &factoringports.ListChargebacksDueRequest{
		TenantInfo:   tenantInfo,
		Before:       now + int64(factoring.ChargebackWarningDays)*timeutils.SecondsPerDay,
		UnwarnedOnly: true,
		Limit:        maxAnnouncements,
	})
	if err != nil {
		return 0, err
	}
	warned := 0
	for _, row := range due {
		claimed, claimErr := s.submissions.ClaimChargebackWarning(
			ctx,
			factoringports.ClaimChargebackWarningRequest{
				TenantInfo: tenantInfo,
				ItemID:     row.ItemID,
				At:         now,
			},
		)
		if claimErr != nil {
			s.l.Warn("could not claim a chargeback warning",
				zap.String("itemId", row.ItemID.String()), zap.Error(claimErr))
			continue
		}
		if !claimed {
			continue
		}
		agentports.PublishAgentEvent(ctx, s.events, agentports.AgentEvent{
			Kind:       agent.EventFactoringChargebackDue,
			SubjectID:  row.InvoiceID,
			TenantInfo: tenantInfo,
			Related:    []pulid.ID{row.SubmissionID, row.FactoringCompanyID, row.CustomerID},
			Detail: "Invoice " + row.InvoiceNumber + " can be charged back from " +
				timeutils.FormatUnixDate(row.DeadlineAt),
		})
		warned++
	}
	return warned, nil
}

func (s *ReadinessService) warnExpiringNoas(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (int, error) {
	now := s.now()
	expiring, err := s.noas.ListExpiring(ctx, factoringports.ListExpiringNoasRequest{
		TenantInfo: tenantInfo,
		At:         now,
		Before:     now + int64(factoring.NoaExpiryWarningDays)*timeutils.SecondsPerDay,
		Limit:      maxAnnouncements,
	})
	if err != nil {
		return 0, err
	}
	warned := 0
	for _, noa := range expiring {
		claimed, claimErr := s.noas.ClaimExpiryWarning(
			ctx,
			factoringports.ClaimNoaExpiryWarningRequest{
				TenantInfo: tenantInfo,
				ID:         noa.ID,
				At:         now,
			},
		)
		if claimErr != nil {
			s.l.Warn("could not claim a notice expiry warning",
				zap.String("noaId", noa.ID.String()), zap.Error(claimErr))
			continue
		}
		if !claimed {
			continue
		}
		agentports.PublishAgentEvent(ctx, s.events, agentports.AgentEvent{
			Kind:       agent.EventFactoringNoaExpiring,
			SubjectID:  noa.ID,
			TenantInfo: tenantInfo,
			Related:    []pulid.ID{noa.CustomerID, noa.FactoringCompanyID},
			Detail: "The notice of assignment expires " +
				timeutils.FormatUnixDate(*noa.ExpiresAt),
		})
		warned++
	}
	return warned, nil
}

func invoiceFacts(inv *invoice.Invoice) factoring.InvoiceFacts {
	facts := factoring.InvoiceFacts{
		Posted:           inv.Status == invoice.StatusPosted,
		Voided:           inv.Status == invoice.StatusVoided,
		CreditMemo:       inv.BillType == billingqueue.BillTypeCreditMemo,
		Excluded:         inv.FactoringOverride.Excludes(),
		OpenBalanceMinor: inv.OpenBalanceMinor(),
	}
	if inv.FactoringCompanyID != nil {
		facts.FactoringCompanyID = *inv.FactoringCompanyID
	}
	return facts
}

func agreementNeeds(a *factoring.Agreement) []factoring.DocumentNeed {
	needs := make([]factoring.DocumentNeed, 0, len(a.DocumentTypes))
	for _, dt := range a.DocumentTypes {
		if dt == nil {
			continue
		}
		name := dt.Name
		if name == "" {
			name = dt.Code
		}
		needs = append(needs, factoring.DocumentNeed{TypeID: dt.ID, Name: name})
	}
	return needs
}

func legPros(inv *invoice.Invoice) map[pulid.ID]string {
	pros := make(map[pulid.ID]string, len(inv.Lines)+1)
	for _, line := range inv.Lines {
		if line == nil || line.ShipmentID.IsNil() || line.ShipmentProNumber == "" {
			continue
		}
		if _, ok := pros[line.ShipmentID]; !ok {
			pros[line.ShipmentID] = line.ShipmentProNumber
		}
	}
	if !inv.ShipmentID.IsNil() && inv.ShipmentProNumber != "" {
		if _, ok := pros[inv.ShipmentID]; !ok {
			pros[inv.ShipmentID] = inv.ShipmentProNumber
		}
	}
	return pros
}

func uniqueIDs(ids []pulid.ID) []pulid.ID {
	seen := make(map[pulid.ID]struct{}, len(ids))
	out := make([]pulid.ID, 0, len(ids))
	for _, id := range ids {
		if id.IsNil() {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
