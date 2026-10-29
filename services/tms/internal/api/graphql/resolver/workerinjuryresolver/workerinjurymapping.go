package workerinjuryresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerinjuryservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerinjury"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
)

func injuryFromInput(
	input *gqlmodel.RecordWorkerInjuryInput,
	tenantInfo pagination.TenantInfo,
) (*workerinjury.WorkerInjury, error) {
	workerID, err := pulid.MustParse(input.WorkerID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"workerId",
			errortypes.ErrInvalid,
			"Worker is invalid",
		)
	}
	safetyEventID, err := gqlid.OptionalID(input.SafetyEventID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"safetyEventId",
			errortypes.ErrInvalid,
			"Safety event is invalid",
		)
	}
	documentID, err := gqlid.OptionalID(input.DocumentID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"documentId",
			errortypes.ErrInvalid,
			"Document is invalid",
		)
	}

	entity := &workerinjury.WorkerInjury{
		OrganizationID:   tenantInfo.OrgID,
		BusinessUnitID:   tenantInfo.BuID,
		WorkerID:         workerID,
		OccurredAt:       int64(input.OccurredAt),
		Description:      input.Description,
		ReportedAt:       gqlscalar.Int64Ptr(input.ReportedAt),
		ReturnedToWorkAt: gqlscalar.Int64Ptr(input.ReturnedToWorkAt),
		Location:         gqlscalar.StringValue(input.Location),
		BodyPart:         gqlscalar.StringValue(input.BodyPart),
		HarmfulAgent:     gqlscalar.StringValue(input.HarmfulAgent),
		DaysAway:         intutils.SafeToInt32(gqlscalar.IntValue(input.DaysAway)),
		DaysRestricted:   intutils.SafeToInt32(gqlscalar.IntValue(input.DaysRestricted)),
		PrivacyCase:      gqlscalar.BoolValue(input.PrivacyCase),
		ClaimNumber:      gqlscalar.StringValue(input.ClaimNumber),
		ClaimCarrier:     gqlscalar.StringValue(input.ClaimCarrier),
		ClaimFiledAt:     gqlscalar.Int64Ptr(input.ClaimFiledAt),
		SafetyEventID:    safetyEventID,
		DocumentID:       documentID,
		Notes:            gqlscalar.StringValue(input.Notes),
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
	safetyEventID, err := gqlid.OptionalID(input.SafetyEventID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"safetyEventId",
			errortypes.ErrInvalid,
			"Safety event is invalid",
		)
	}
	documentID, err := gqlid.OptionalID(input.DocumentID)
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
		OccurredAt:       gqlscalar.Int64Ptr(input.OccurredAt),
		ReportedAt:       gqlscalar.Int64Ptr(input.ReportedAt),
		ReturnedToWorkAt: gqlscalar.Int64Ptr(input.ReturnedToWorkAt),
		Location:         input.Location,
		Description:      input.Description,
		BodyPart:         input.BodyPart,
		HarmfulAgent:     input.HarmfulAgent,
		PrivacyCase:      input.PrivacyCase,
		ClaimStatus:      input.ClaimStatus,
		ClaimNumber:      input.ClaimNumber,
		ClaimCarrier:     input.ClaimCarrier,
		ClaimFiledAt:     gqlscalar.Int64Ptr(input.ClaimFiledAt),
		ClaimClosedAt:    gqlscalar.Int64Ptr(input.ClaimClosedAt),
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
		PostedFrom:          gqlscalar.Int64Ptr(input.PostedFrom),
		PostedThrough:       gqlscalar.Int64Ptr(input.PostedThrough),
		SubmittedAt:         gqlscalar.Int64Ptr(input.SubmittedAt),
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
