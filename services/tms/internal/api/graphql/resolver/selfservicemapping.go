package resolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/services/selfserviceservice"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

func workerPolicyFromInput(
	input gqlmodel.WorkerPolicyInput,
	tenant pagination.TenantInfo,
) (*worker.WorkerPolicy, error) {
	documentID, err := optionalID(input.DocumentID)
	if err != nil {
		return nil, invalidIDError("documentId", "Document is invalid")
	}

	status := domaintypes.StatusActive
	if input.Status != nil {
		status = *input.Status
	}

	return &worker.WorkerPolicy{
		OrganizationID:    tenant.OrgID,
		BusinessUnitID:    tenant.BuID,
		Status:            status,
		Code:              input.Code,
		Title:             input.Title,
		Summary:           stringValue(input.Summary),
		Body:              stringValue(input.Body),
		DocumentID:        documentID,
		VersionLabel:      input.VersionLabel,
		RequiresSignature: input.RequiresSignature,
		AppliesTo:         input.AppliesTo,
		EffectiveFrom:     int64Value(input.EffectiveFrom),
	}, nil
}

func policyComplianceToGQL(compliance *selfserviceservice.PolicyCompliance) *gqlmodel.PolicyCompliance {
	out := &gqlmodel.PolicyCompliance{
		Policy:      compliance.Policy,
		Signed:      compliance.Signed,
		Outstanding: compliance.Outstanding,
		Rows:        make([]*gqlmodel.PolicyComplianceRow, 0, len(compliance.Rows)),
	}

	for _, row := range compliance.Rows {
		item := &gqlmodel.PolicyComplianceRow{
			WorkerID:   row.WorkerID.String(),
			WorkerName: row.FirstName + " " + row.LastName,
			WorkerType: row.WorkerType,
		}
		if row.AcknowledgedAt > 0 {
			signedAt := int(row.AcknowledgedAt)
			item.AcknowledgedAt = &signedAt
		}
		item.SignatureName = emptyToNil(row.SignatureName)
		out.Rows = append(out.Rows, item)
	}

	return out
}
