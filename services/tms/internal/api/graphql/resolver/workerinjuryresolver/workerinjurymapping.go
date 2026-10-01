package workerinjuryresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/services/workerinjuryservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
)

func injuryFromInput(
	input *gqlmodel.RecordWorkerInjuryInput,
	tenantInfo pagination.TenantInfo,
) (*worker.WorkerInjury, error) {
	workerID, err := pulid.MustParse(input.WorkerID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"workerId",
			errortypes.ErrInvalid,
			"Worker is invalid",
		)
	}
	safetyEventID, err := base.OptionalID(input.SafetyEventID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"safetyEventId",
			errortypes.ErrInvalid,
			"Safety event is invalid",
		)
	}
	documentID, err := base.OptionalID(input.DocumentID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"documentId",
			errortypes.ErrInvalid,
			"Document is invalid",
		)
	}

	entity := &worker.WorkerInjury{
		OrganizationID:   tenantInfo.OrgID,
		BusinessUnitID:   tenantInfo.BuID,
		WorkerID:         workerID,
		OccurredAt:       int64(input.OccurredAt),
		Description:      input.Description,
		ReportedAt:       base.Int64Ptr(input.ReportedAt),
		ReturnedToWorkAt: base.Int64Ptr(input.ReturnedToWorkAt),
		Location:         base.StringValue(input.Location),
		BodyPart:         base.StringValue(input.BodyPart),
		HarmfulAgent:     base.StringValue(input.HarmfulAgent),
		DaysAway:         intutils.SafeToInt32(base.IntValue(input.DaysAway)),
		DaysRestricted:   intutils.SafeToInt32(base.IntValue(input.DaysRestricted)),
		PrivacyCase:      base.BoolValue(input.PrivacyCase),
		ClaimNumber:      base.StringValue(input.ClaimNumber),
		ClaimCarrier:     base.StringValue(input.ClaimCarrier),
		ClaimFiledAt:     base.Int64Ptr(input.ClaimFiledAt),
		SafetyEventID:    safetyEventID,
		DocumentID:       documentID,
		Notes:            base.StringValue(input.Notes),
	}
	if input.Classification != nil {
		entity.Classification = *input.Classification
	}
	if input.IllnessType != nil {
		entity.IllnessType = *input.IllnessType
	}
	if input.Treatment != nil {
		entity.Treatment = *input.Treatment
	}
	if input.ClaimStatus != nil {
		entity.ClaimStatus = *input.ClaimStatus
	}

	return entity, nil
}

func injuryUpdateRequest(
	input *gqlmodel.UpdateWorkerInjuryInput,
	tenantInfo pagination.TenantInfo,
	userID pulid.ID,
) (*workerinjuryservice.UpdateInjuryRequest, error) {
	injuryID, err := pulid.MustParse(input.InjuryID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"injuryId",
			errortypes.ErrInvalid,
			"Case is invalid",
		)
	}
	safetyEventID, err := base.OptionalID(input.SafetyEventID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"safetyEventId",
			errortypes.ErrInvalid,
			"Safety event is invalid",
		)
	}
	documentID, err := base.OptionalID(input.DocumentID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"documentId",
			errortypes.ErrInvalid,
			"Document is invalid",
		)
	}

	req := &workerinjuryservice.UpdateInjuryRequest{
		TenantInfo:       tenantInfo,
		InjuryID:         injuryID,
		Classification:   input.Classification,
		IllnessType:      input.IllnessType,
		Treatment:        input.Treatment,
		Status:           input.Status,
		OccurredAt:       base.Int64Ptr(input.OccurredAt),
		ReportedAt:       base.Int64Ptr(input.ReportedAt),
		ReturnedToWorkAt: base.Int64Ptr(input.ReturnedToWorkAt),
		Location:         input.Location,
		Description:      input.Description,
		BodyPart:         input.BodyPart,
		HarmfulAgent:     input.HarmfulAgent,
		PrivacyCase:      input.PrivacyCase,
		ClaimStatus:      input.ClaimStatus,
		ClaimNumber:      input.ClaimNumber,
		ClaimCarrier:     input.ClaimCarrier,
		ClaimFiledAt:     base.Int64Ptr(input.ClaimFiledAt),
		ClaimClosedAt:    base.Int64Ptr(input.ClaimClosedAt),
		SafetyEventID:    safetyEventID,
		DocumentID:       documentID,
		Notes:            input.Notes,
		UserID:           userID,
	}
	if input.DaysAway != nil {
		days := int32(*input.DaysAway)
		req.DaysAway = &days
	}
	if input.DaysRestricted != nil {
		days := int32(*input.DaysRestricted)
		req.DaysRestricted = &days
	}

	return req, nil
}

func oshaSummaryRequest(
	input *gqlmodel.SaveOSHASummaryInput,
	tenantInfo pagination.TenantInfo,
	userID pulid.ID,
) *workerinjuryservice.SaveSummaryRequest {
	req := &workerinjuryservice.SaveSummaryRequest{
		TenantInfo:          tenantInfo,
		Year:                int16(input.Year),
		NAICSCode:           input.NaicsCode,
		ExecutiveName:       input.ExecutiveName,
		ExecutiveTitle:      input.ExecutiveTitle,
		ExecutivePhone:      input.ExecutivePhone,
		PostedFrom:          base.Int64Ptr(input.PostedFrom),
		PostedThrough:       base.Int64Ptr(input.PostedThrough),
		SubmittedAt:         base.Int64Ptr(input.SubmittedAt),
		SubmissionReference: input.SubmissionReference,
		Notes:               input.Notes,
		UserID:              userID,
	}
	if input.AverageEmployees != nil {
		count := int32(*input.AverageEmployees)
		req.AverageEmployees = &count
	}
	if input.TotalHoursWorked != nil {
		hours := int64(*input.TotalHoursWorked)
		req.TotalHoursWorked = &hours
	}

	return req
}
