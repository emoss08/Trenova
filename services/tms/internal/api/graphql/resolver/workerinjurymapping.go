package resolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/services/workerinjuryservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
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
	safetyEventID, err := optionalID(input.SafetyEventID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"safetyEventId",
			errortypes.ErrInvalid,
			"Safety event is invalid",
		)
	}
	documentID, err := optionalID(input.DocumentID)
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
		ReportedAt:       int64Ptr(input.ReportedAt),
		ReturnedToWorkAt: int64Ptr(input.ReturnedToWorkAt),
		Location:         stringValue(input.Location),
		BodyPart:         stringValue(input.BodyPart),
		HarmfulAgent:     stringValue(input.HarmfulAgent),
		DaysAway:         int32(intValue(input.DaysAway)),
		DaysRestricted:   int32(intValue(input.DaysRestricted)),
		PrivacyCase:      boolValue(input.PrivacyCase),
		ClaimNumber:      stringValue(input.ClaimNumber),
		ClaimCarrier:     stringValue(input.ClaimCarrier),
		ClaimFiledAt:     int64Ptr(input.ClaimFiledAt),
		SafetyEventID:    safetyEventID,
		DocumentID:       documentID,
		Notes:            stringValue(input.Notes),
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
	safetyEventID, err := optionalID(input.SafetyEventID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"safetyEventId",
			errortypes.ErrInvalid,
			"Safety event is invalid",
		)
	}
	documentID, err := optionalID(input.DocumentID)
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
		OccurredAt:       int64Ptr(input.OccurredAt),
		ReportedAt:       int64Ptr(input.ReportedAt),
		ReturnedToWorkAt: int64Ptr(input.ReturnedToWorkAt),
		Location:         input.Location,
		Description:      input.Description,
		BodyPart:         input.BodyPart,
		HarmfulAgent:     input.HarmfulAgent,
		PrivacyCase:      input.PrivacyCase,
		ClaimStatus:      input.ClaimStatus,
		ClaimNumber:      input.ClaimNumber,
		ClaimCarrier:     input.ClaimCarrier,
		ClaimFiledAt:     int64Ptr(input.ClaimFiledAt),
		ClaimClosedAt:    int64Ptr(input.ClaimClosedAt),
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
		PostedFrom:          int64Ptr(input.PostedFrom),
		PostedThrough:       int64Ptr(input.PostedThrough),
		SubmittedAt:         int64Ptr(input.SubmittedAt),
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
