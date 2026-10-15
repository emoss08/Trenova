package agentsubjectservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func (s *Service) factoringNoa(
	ctx context.Context,
	tenant pagination.TenantInfo,
	noaID pulid.ID,
) (*agentdefinition.RuntimeSubject, error) {
	subject := &agentdefinition.RuntimeSubject{
		Type:  agent.SubjectFactoringNoa,
		ID:    noaID.String(),
		Label: "Notice of assignment",
	}
	if s.noas == nil {
		return subject, nil
	}

	found, err := s.noas.GetByID(ctx, factoringports.GetNoaRequest{
		ID:         noaID,
		TenantInfo: tenant,
	})
	if err != nil {
		return nil, fmt.Errorf("load notice of assignment: %w", err)
	}

	subject.Notes = marshalNotes(map[string]any{
		noteStatus:           found.Status,
		noteCustomerID:       found.CustomerID,
		"factoringCompanyId": found.FactoringCompanyID,
		"sentAt":             found.SentAt,
		"sentTo":             found.SentTo,
		"acknowledgedAt":     found.AcknowledgedAt,
		"expiresAt":          found.ExpiresAt,
		"releasedAt":         found.ReleasedAt,
	})
	return subject, nil
}

func (s *Service) factoringSubmission(
	ctx context.Context,
	tenant pagination.TenantInfo,
	submissionID pulid.ID,
) (*agentdefinition.RuntimeSubject, error) {
	subject := &agentdefinition.RuntimeSubject{
		Type:  agent.SubjectFactoringSubmission,
		ID:    submissionID.String(),
		Label: "Factoring schedule",
	}
	if s.submissions == nil {
		return subject, nil
	}

	found, err := s.submissions.GetByID(ctx, factoringports.GetSubmissionRequest{
		ID:         submissionID,
		TenantInfo: tenant,
	})
	if err != nil {
		return nil, fmt.Errorf("load factoring schedule: %w", err)
	}

	subject.Label = "Factoring schedule " + found.Number
	subject.Notes = marshalNotes(map[string]any{
		noteStatus:           found.Status,
		noteNumber:           found.Number,
		"factoringCompanyId": found.FactoringCompanyID,
		"invoiceCount":       found.InvoiceCount,
		"faceMinor":          found.FaceMinor,
		"transportMethod":    found.TransportMethod,
		"transportTarget":    found.TransportTarget,
		"sentAt":             found.SentAt,
		"failedAt":           found.FailedAt,
		"lastError":          found.LastError,
		"version":            found.Version,
	})
	return subject, nil
}

func (s *Service) factoringRemittance(
	ctx context.Context,
	tenant pagination.TenantInfo,
	remittanceID pulid.ID,
) (*agentdefinition.RuntimeSubject, error) {
	subject := &agentdefinition.RuntimeSubject{
		Type:  agent.SubjectFactoringRemittance,
		ID:    remittanceID.String(),
		Label: "Factor remittance",
	}
	if s.remittances == nil {
		return subject, nil
	}

	found, err := s.remittances.Get(ctx, factoringports.GetRemittanceRequest{
		ID:         remittanceID,
		TenantInfo: tenant,
	})
	if err != nil {
		return nil, fmt.Errorf("load factor remittance: %w", err)
	}

	if found.FileName != "" {
		subject.Label = "Factor remittance " + found.FileName
	}
	subject.Notes = marshalNotes(map[string]any{
		noteStatus:           found.Status,
		"factoringCompanyId": found.FactoringCompanyID,
		"fileName":           found.FileName,
		"source":             found.Source,
		"defaultKind":        found.DefaultKind,
		"lineCount":          found.LineCount,
		"matchedCount":       found.MatchedCount,
		"exceptionCount":     found.ExceptionCount,
		"appliedCount":       found.AppliedCount,
		"totalCashMinor":     found.TotalCashMinor,
		"error":              found.Error,
		"version":            found.Version,
	})
	return subject, nil
}
