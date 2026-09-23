package invoiceservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/temporaljobs/billingjobs"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

const (
	ediBlockerVoided            = "The invoice has been voided"
	ediBlockerNotPosted         = "The invoice has not been posted"
	ediBlockerProfileDisabled   = "EDI invoicing is not enabled on the customer's billing profile"
	ediBlockerNoPartner         = "The customer has no active EDI partner enabled for outbound documents"
	ediBlockerNoDocumentProfile = "The partner has no active outbound 210 document profile"
	ediBlockerNoCommunication   = "The partner has no active communication profile to deliver through"
	ediBlockerNotWired          = "EDI invoicing is not available on this server"
)

// ResolveEDISendPlans works out, for each invoice, whether an outbound 210 can
// go and where. Partners are fetched once per customer and profiles once per
// partner, so a page of invoices costs a handful of queries.
func (s *Service) ResolveEDISendPlans(
	ctx context.Context,
	req *servicesports.ResolveInvoiceEDISendPlansRequest,
) (map[pulid.ID]*servicesports.InvoiceEDISendPlan, error) {
	plans := make(map[pulid.ID]*servicesports.InvoiceEDISendPlan, len(req.Invoices))
	if len(req.Invoices) == 0 {
		return plans, nil
	}
	if s.ediPartnerRepo == nil {
		// EDI is not wired into this process: every plan is unconfigured, and no
		// customer lookup is needed to say so.
		for _, inv := range req.Invoices {
			if inv == nil {
				continue
			}
			plans[inv.ID] = &servicesports.InvoiceEDISendPlan{
				InvoiceID: inv.ID,
				Status:    inv.EDISendStatus,
				Blockers:  []string{ediBlockerNotWired},
			}
		}
		return plans, nil
	}

	customerIDs := make([]pulid.ID, 0, len(req.Invoices))
	seen := make(map[pulid.ID]struct{}, len(req.Invoices))
	for _, inv := range req.Invoices {
		if inv == nil || inv.CustomerID.IsNil() {
			continue
		}
		if _, dup := seen[inv.CustomerID]; dup {
			continue
		}
		seen[inv.CustomerID] = struct{}{}
		customerIDs = append(customerIDs, inv.CustomerID)
	}

	customers, err := s.customerRepo.GetByIDs(ctx, repositories.GetCustomersByIDsRequest{
		TenantInfo:            req.TenantInfo,
		CustomerIDs:           customerIDs,
		CustomerFilterOptions: repositories.CustomerFilterOptions{IncludeBillingProfile: true},
	})
	if err != nil {
		return nil, err
	}
	profiles := make(map[pulid.ID]*customer.CustomerBillingProfile, len(customers))
	ediCustomerIDs := make([]pulid.ID, 0, len(customers))
	for _, cus := range customers {
		if cus == nil || cus.BillingProfile == nil {
			continue
		}
		profiles[cus.ID] = cus.BillingProfile
		if cus.BillingProfile.EDIInvoiceEnabled {
			ediCustomerIDs = append(ediCustomerIDs, cus.ID)
		}
	}

	partners, err := s.outboundPartnersByCustomer(ctx, req.TenantInfo, ediCustomerIDs)
	if err != nil {
		return nil, err
	}
	targets := make(map[pulid.ID]*ediPartnerTarget, len(partners))
	for customerID, partner := range partners {
		target, resolveErr := s.resolvePartnerTarget(ctx, req.TenantInfo, partner)
		if resolveErr != nil {
			return nil, resolveErr
		}
		targets[customerID] = target
	}

	for _, inv := range req.Invoices {
		if inv == nil {
			continue
		}
		plans[inv.ID] = buildEDISendPlan(inv, profiles[inv.CustomerID], targets[inv.CustomerID])
	}

	return plans, nil
}

// ediPartnerTarget is a customer's outbound partner with the 210 profile and
// transport it would use, or the reason it cannot be used.
type ediPartnerTarget struct {
	Partner             *edi.EDIPartner
	DocumentProfile     *edi.EDIPartnerDocumentProfile
	CommunicationMethod edi.ConnectionMethod
	Blockers            []string
}

func (s *Service) outboundPartnersByCustomer(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	customerIDs []pulid.ID,
) (map[pulid.ID]*edi.EDIPartner, error) {
	result := make(map[pulid.ID]*edi.EDIPartner, len(customerIDs))
	if len(customerIDs) == 0 || s.ediPartnerRepo == nil {
		return result, nil
	}
	partners, err := s.ediPartnerRepo.ListOutboundPartnersByCustomerIDs(
		ctx,
		repositories.ListEDIPartnersByCustomerIDsRequest{
			CustomerIDs: customerIDs,
			TenantInfo:  tenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}
	for _, partner := range partners {
		if partner == nil || partner.CustomerID.IsNil() {
			continue
		}
		// The oldest active partner wins, matching the tender routing rule.
		if _, exists := result[partner.CustomerID]; exists {
			continue
		}
		result[partner.CustomerID] = partner
	}

	return result, nil
}

func (s *Service) resolvePartnerTarget(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	partner *edi.EDIPartner,
) (*ediPartnerTarget, error) {
	target := &ediPartnerTarget{Partner: partner}
	if s.ediDocumentProfileRepo != nil {
		profile, err := s.ediDocumentProfileRepo.GetActivePartnerDocumentProfile(
			ctx,
			repositories.GetActiveEDIPartnerDocumentProfileRequest{
				PartnerID:      partner.ID,
				TenantInfo:     tenantInfo,
				TransactionSet: edi.TransactionSet210,
				Direction:      edi.DocumentDirectionOutbound,
			},
		)
		switch {
		case err == nil:
			target.DocumentProfile = profile
		case errortypes.IsNotFoundError(err):
			target.Blockers = append(target.Blockers, ediBlockerNoDocumentProfile)
		default:
			return nil, err
		}
	} else {
		target.Blockers = append(target.Blockers, ediBlockerNoDocumentProfile)
	}

	if partner.Kind == edi.PartnerKindInternal {
		target.CommunicationMethod = edi.ConnectionMethodInternal
		return target, nil
	}
	if s.ediCommunicationProfileRepo == nil {
		target.Blockers = append(target.Blockers, ediBlockerNoCommunication)
		return target, nil
	}
	commProfile, err := s.ediCommunicationProfileRepo.GetActiveProfileByPartner(
		ctx,
		repositories.GetActiveEDICommunicationProfileByPartnerRequest{
			PartnerID:  partner.ID,
			TenantInfo: tenantInfo,
		},
	)
	switch {
	case err == nil:
		target.CommunicationMethod = commProfile.Method
	case errortypes.IsNotFoundError(err):
		target.Blockers = append(target.Blockers, ediBlockerNoCommunication)
	default:
		return nil, err
	}

	return target, nil
}

func buildEDISendPlan(
	inv *invoice.Invoice,
	profile *customer.CustomerBillingProfile,
	target *ediPartnerTarget,
) *servicesports.InvoiceEDISendPlan {
	plan := &servicesports.InvoiceEDISendPlan{
		InvoiceID:     inv.ID,
		Status:        inv.EDISendStatus,
		LastMessageID: inv.LastEDIMessageID,
		LastError:     inv.LastEDIError,
		SentAt:        inv.EDISentAt,
		Blockers:      make([]string, 0, 2),
	}
	if plan.Status == "" {
		plan.Status = invoice.EDISendStatusNotSent
	}
	if profile == nil || !profile.EDIInvoiceEnabled {
		plan.Blockers = append(plan.Blockers, ediBlockerProfileDisabled)
		return plan
	}
	plan.Enabled = true
	plan.AutoSend = profile.AutoSendInvoiceOnGeneration
	switch inv.Status {
	case invoice.StatusVoided:
		plan.Blockers = append(plan.Blockers, ediBlockerVoided)
	case invoice.StatusPosted:
	default:
		plan.Blockers = append(plan.Blockers, ediBlockerNotPosted)
	}
	if target == nil || target.Partner == nil {
		plan.Blockers = append(plan.Blockers, ediBlockerNoPartner)
		return plan
	}
	plan.PartnerID = target.Partner.ID
	plan.PartnerName = target.Partner.Name
	if target.DocumentProfile != nil {
		plan.DocumentProfileID = target.DocumentProfile.ID
	}
	plan.CommunicationMethod = string(target.CommunicationMethod)
	plan.Blockers = append(plan.Blockers, target.Blockers...)

	return plan
}

// SendEDI queues the outbound 210 for an invoice. Force resends an invoice
// that already went, which is what a partner asks for after losing a file.
func (s *Service) SendEDI(
	ctx context.Context,
	req *servicesports.SendInvoiceEDIRequest,
	actor *servicesports.RequestActor,
) (*servicesports.InvoiceEDISendResult, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request is required",
		)
	}
	if actor == nil {
		return nil, errortypes.NewValidationError(
			"actor",
			errortypes.ErrRequired,
			"Actor is required",
		)
	}
	entity, err := s.repo.GetByID(ctx, repositories.GetInvoiceByIDRequest{
		ID:         req.InvoiceID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if entity.Status != invoice.StatusPosted {
		return nil, errortypes.NewValidationError(
			"invoiceId",
			errortypes.ErrInvalidOperation,
			"Only a posted invoice can be sent by EDI",
		)
	}
	plans, err := s.ResolveEDISendPlans(ctx, &servicesports.ResolveInvoiceEDISendPlansRequest{
		TenantInfo: req.TenantInfo,
		Invoices:   []*invoice.Invoice{entity},
	})
	if err != nil {
		return nil, err
	}
	plan := plans[entity.ID]
	if plan == nil || !plan.Enabled || len(plan.Blockers) > 0 {
		blocker := ediBlockerProfileDisabled
		if plan != nil && len(plan.Blockers) > 0 {
			blocker = plan.Blockers[0]
		}
		return nil, errortypes.NewValidationError(
			"invoiceId",
			errortypes.ErrInvalidOperation,
			blocker,
		)
	}
	if !req.Force {
		switch entity.EDISendStatus {
		case invoice.EDISendStatusQueued, invoice.EDISendStatusSending:
			return nil, errortypes.NewValidationError(
				"invoiceId",
				errortypes.ErrInvalidOperation,
				"An EDI send for this invoice is already in progress",
			)
		case invoice.EDISendStatusSent:
			return nil, errortypes.NewValidationError(
				"invoiceId",
				errortypes.ErrInvalidOperation,
				"This invoice has already been sent by EDI; resend it with force",
			)
		}
	}

	return s.startEDISendWorkflow(ctx, entity, req.TenantInfo, actor, req.Force)
}

// enqueueEDIAfterPost is the auto-send hook. It never fails the post: a
// misconfigured partner leaves the invoice marked NotConfigured with the
// reason, and an enqueue failure leaves it Failed for a manual resend.
func (s *Service) enqueueEDIAfterPost(
	ctx context.Context,
	entity *invoice.Invoice,
	tenantInfo pagination.TenantInfo,
	actor *servicesports.RequestActor,
) {
	if entity == nil || s.ediPartnerRepo == nil {
		return
	}
	plans, err := s.ResolveEDISendPlans(ctx, &servicesports.ResolveInvoiceEDISendPlansRequest{
		TenantInfo: tenantInfo,
		Invoices:   []*invoice.Invoice{entity},
	})
	if err != nil {
		s.l.Warn("failed to resolve invoice EDI plan after post", zap.Error(err))
		return
	}
	plan := plans[entity.ID]
	if plan == nil || !plan.Enabled {
		return
	}
	if len(plan.Blockers) > 0 {
		s.recordEDIStatus(
			ctx,
			entity,
			tenantInfo,
			invoice.EDISendStatusNotConfigured,
			plan.Blockers[0],
		)
		return
	}
	if !plan.AutoSend {
		return
	}
	if _, err = s.startEDISendWorkflow(ctx, entity, tenantInfo, actor, false); err != nil {
		s.l.Warn("failed to queue invoice EDI send", zap.Error(err))
		s.recordEDIStatus(ctx, entity, tenantInfo, invoice.EDISendStatusFailed, err.Error())
	}
}

func (s *Service) startEDISendWorkflow(
	ctx context.Context,
	entity *invoice.Invoice,
	tenantInfo pagination.TenantInfo,
	actor *servicesports.RequestActor,
	force bool,
) (*servicesports.InvoiceEDISendResult, error) {
	if s.workflowStarter == nil || !s.workflowStarter.Enabled() {
		return nil, errortypes.NewBusinessError(
			"Invoice EDI delivery requires workflow processing to be enabled",
		).WithInternal(servicesports.ErrWorkflowStarterDisabled)
	}
	auditActor := actor.AuditActorOrSystem()
	run, err := s.workflowStarter.StartWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID: fmt.Sprintf(
				"invoice-edi-send-%s-%s",
				entity.ID.String(),
				pulid.MustNew("wf_").String(),
			),
			TaskQueue:     temporaltype.TaskQueueBilling.String(),
			StaticSummary: "Send invoice " + entity.Number + " by EDI",
		},
		billingjobs.SendInvoiceEDIWorkflowName,
		&billingjobs.SendInvoiceEDIPayload{
			BasePayload: temporaltype.BasePayload{
				OrganizationID: entity.OrganizationID,
				BusinessUnitID: entity.BusinessUnitID,
				UserID:         auditActor.UserID,
				Timestamp:      timeutils.NowUnix(),
			},
			InvoiceID:     entity.ID,
			Force:         force,
			PrincipalType: auditActor.PrincipalType,
			PrincipalID:   auditActor.PrincipalID,
			APIKeyID:      auditActor.APIKeyID,
		},
	)
	if err != nil {
		return nil, errortypes.NewDatabaseError("Failed to start invoice EDI send").
			WithInternal(err)
	}
	s.recordEDIStatus(ctx, entity, tenantInfo, invoice.EDISendStatusQueued, "")

	return &servicesports.InvoiceEDISendResult{
		InvoiceID:     entity.ID,
		Status:        invoice.EDISendStatusQueued,
		WorkflowID:    run.GetID(),
		WorkflowRunID: run.GetRunID(),
	}, nil
}

func (s *Service) recordEDIStatus(
	ctx context.Context,
	entity *invoice.Invoice,
	tenantInfo pagination.TenantInfo,
	status invoice.EDISendStatus,
	reason string,
) {
	if err := s.repo.UpdateEDISendStatus(ctx, repositories.UpdateInvoiceEDISendStatusRequest{
		TenantInfo: tenantInfo,
		InvoiceID:  entity.ID,
		MessageID:  entity.LastEDIMessageID,
		Status:     status,
		Error:      reason,
	}); err != nil {
		s.l.Warn("failed to record invoice EDI status", zap.Error(err))
		return
	}
	entity.EDISendStatus = status
	entity.LastEDIError = reason
}
