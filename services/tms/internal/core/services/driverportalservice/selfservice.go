package driverportalservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/selfserviceservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// PortalPolicy is a policy from the driver's side: what it is, whether they
// have signed the version in force, and what signing it asks of them.
type PortalPolicy struct {
	ID                pulid.ID `json:"id"`
	Code              string   `json:"code"`
	Title             string   `json:"title"`
	Summary           string   `json:"summary"`
	Body              string   `json:"body"`
	HasDocument       bool     `json:"hasDocument"`
	VersionLabel      string   `json:"versionLabel"`
	RequiresSignature bool     `json:"requiresSignature"`
	EffectiveFrom     int64    `json:"effectiveFrom"`
	AcknowledgedAt    *int64   `json:"acknowledgedAt"`
	SignatureName     string   `json:"signatureName"`
}

func (s *Service) requireSelfService() error {
	if s.selfService == nil {
		return errortypes.NewValidationError(
			"feature",
			errortypes.ErrInvalidOperation,
			"Self-service is not available",
		)
	}
	return nil
}

// MyPolicies is every policy in force that binds the driver, signed or not.
func (s *Service) MyPolicies(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*PortalPolicy, error) {
	if err := s.requireSelfService(); err != nil {
		return nil, err
	}
	wrk, err := s.ResolveWorker(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	standings, err := s.selfService.PoliciesForWorker(ctx, tenantInfo, wrk)
	if err != nil {
		return nil, err
	}

	out := make([]*PortalPolicy, 0, len(standings))
	for _, standing := range standings {
		view := &PortalPolicy{
			ID:                standing.Policy.ID,
			Code:              standing.Policy.Code,
			Title:             standing.Policy.Title,
			Summary:           standing.Policy.Summary,
			Body:              standing.Policy.Body,
			HasDocument:       !standing.Policy.DocumentID.IsNil(),
			VersionLabel:      standing.Policy.VersionLabel,
			RequiresSignature: standing.Policy.RequiresSignature,
			EffectiveFrom:     standing.Policy.EffectiveFrom,
		}
		if standing.Acknowledgement != nil {
			signedAt := standing.Acknowledgement.AcknowledgedAt
			view.AcknowledgedAt = &signedAt
			view.SignatureName = standing.Acknowledgement.SignatureName
		}
		out = append(out, view)
	}

	return out, nil
}

// MyPolicyDocumentURL is a short-lived link to read the attached document.
// The policy is resolved through the driver's own standing first, so a policy
// that does not bind them is not a document they can open.
func (s *Service) MyPolicyDocumentURL(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	policyID pulid.ID,
) (string, error) {
	if err := s.requireSelfService(); err != nil {
		return "", err
	}
	wrk, err := s.ResolveWorker(ctx, tenantInfo)
	if err != nil {
		return "", err
	}

	standings, err := s.selfService.PoliciesForWorker(ctx, tenantInfo, wrk)
	if err != nil {
		return "", err
	}
	for _, standing := range standings {
		if standing.Policy.ID != policyID {
			continue
		}
		if standing.Policy.DocumentID.IsNil() {
			return "", errortypes.NewValidationError(
				"policyId",
				errortypes.ErrInvalidOperation,
				"That policy has no attached document",
			)
		}
		return s.documentService.GetViewURL(ctx, repositories.GetDocumentByIDRequest{
			ID:         standing.Policy.DocumentID,
			TenantInfo: tenantInfo,
		})
	}

	return "", errortypes.NewNotFoundError("Policy not found")
}

// AcknowledgeMyPolicyRequest is the driver signing.
type AcknowledgeMyPolicyRequest struct {
	PolicyID      pulid.ID
	SignatureName string
	IP            string
	UserAgent     string
}

func (s *Service) AcknowledgeMyPolicy(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	req *AcknowledgeMyPolicyRequest,
) (*worker.WorkerPolicyAcknowledgement, error) {
	if err := s.requireSelfService(); err != nil {
		return nil, err
	}
	wrk, err := s.ResolveWorker(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	return s.selfService.Acknowledge(ctx, &selfserviceservice.AcknowledgeRequest{
		PolicyID:      req.PolicyID,
		Worker:        wrk,
		SignatureName: req.SignatureName,
		IP:            req.IP,
		UserAgent:     req.UserAgent,
		TenantInfo:    tenantInfo,
	})
}

// MyProfileChangeRequests is what the driver has asked to change, newest
// first, so the app can show a change waiting on the office.
func (s *Service) MyProfileChangeRequests(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*worker.WorkerProfileChangeRequest, error) {
	if err := s.requireSelfService(); err != nil {
		return nil, err
	}
	wrk, err := s.ResolveWorker(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	return s.selfService.ListChangeRequests(ctx, &repositories.ListProfileChangeRequestsRequest{
		TenantInfo: tenantInfo,
		WorkerID:   wrk.ID,
		Limit:      10,
	})
}

func (s *Service) WithdrawMyProfileChange(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*worker.WorkerProfileChangeRequest, error) {
	if err := s.requireSelfService(); err != nil {
		return nil, err
	}
	wrk, err := s.ResolveWorker(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	return s.selfService.WithdrawChange(ctx, &selfserviceservice.WithdrawChangeRequest{
		ID:            id,
		ActorWorkerID: wrk.ID,
		TenantInfo:    tenantInfo,
	})
}

// policiesOutstanding is how many signatures the driver still owes, which is
// what puts the policies card at the top of the profile rather than the
// bottom.
func (s *Service) policiesOutstanding(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	wrk *worker.Worker,
) (int, error) {
	if s.selfService == nil {
		return 0, nil
	}
	standings, err := s.selfService.PoliciesForWorker(ctx, tenantInfo, wrk)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, standing := range standings {
		if standing.Outstanding() {
			count++
		}
	}
	return count, nil
}
