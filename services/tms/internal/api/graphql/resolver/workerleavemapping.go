package resolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/services/workerleaveservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func leaveCaseFromInput(
	input *gqlmodel.OpenLeaveCaseInput,
	tenantInfo pagination.TenantInfo,
) (*worker.WorkerLeaveCase, error) {
	workerID, err := pulid.MustParse(input.WorkerID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"workerId",
			errortypes.ErrInvalid,
			"Worker is invalid",
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

	entity := &worker.WorkerLeaveCase{
		OrganizationID:    tenantInfo.OrgID,
		BusinessUnitID:    tenantInfo.BuID,
		WorkerID:          workerID,
		Reason:            stringValue(input.Reason),
		MilitaryCaregiver: boolValue(input.MilitaryCaregiver),
		RequestedAt:       int64Value(input.RequestedAt),
		StartsAt:          int64(input.StartsAt),
		EndsAt:            int64Ptr(input.EndsAt),
		DocumentID:        documentID,
		Notes:             stringValue(input.Notes),
	}
	if input.LeaveType != nil {
		entity.LeaveType = *input.LeaveType
	}
	if input.Frequency != nil {
		entity.Frequency = *input.Frequency
	}
	if input.EligibilityHoursWorked != nil {
		hours := int32(*input.EligibilityHoursWorked)
		entity.EligibilityHoursWorked = &hours
	}

	return entity, nil
}

func leaveCaseUpdateRequest(
	input *gqlmodel.UpdateLeaveCaseInput,
	tenantInfo pagination.TenantInfo,
	userID pulid.ID,
) (*workerleaveservice.UpdateCaseRequest, error) {
	caseID, err := pulid.MustParse(input.CaseID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"caseId",
			errortypes.ErrInvalid,
			"Case is invalid",
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

	req := &workerleaveservice.UpdateCaseRequest{
		TenantInfo:        tenantInfo,
		CaseID:            caseID,
		LeaveType:         input.LeaveType,
		Frequency:         input.Frequency,
		Reason:            input.Reason,
		MilitaryCaregiver: input.MilitaryCaregiver,
		StartsAt:          int64Ptr(input.StartsAt),
		EndsAt:            int64Ptr(input.EndsAt),
		DocumentID:        documentID,
		Notes:             input.Notes,
		UserID:            userID,
	}
	if input.EligibilityHoursWorked != nil {
		hours := int32(*input.EligibilityHoursWorked)
		req.EligibilityHoursWorked = &hours
	}

	return req, nil
}

func leaveCertificationRequest(
	input *gqlmodel.RecordLeaveCertificationInput,
	tenantInfo pagination.TenantInfo,
	userID pulid.ID,
) (*workerleaveservice.RecordCertificationRequest, error) {
	caseID, err := pulid.MustParse(input.CaseID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"caseId",
			errortypes.ErrInvalid,
			"Case is invalid",
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

	return &workerleaveservice.RecordCertificationRequest{
		TenantInfo:           tenantInfo,
		CaseID:               caseID,
		Status:               input.Status,
		ReceivedAt:           int64Ptr(input.ReceivedAt),
		RecertificationDueAt: int64Ptr(input.RecertificationDueAt),
		DocumentID:           documentID,
		UserID:               userID,
	}, nil
}

func leaveDayRequest(
	input *gqlmodel.RecordLeaveDayInput,
	tenantInfo pagination.TenantInfo,
	userID pulid.ID,
) (*workerleaveservice.RecordDayRequest, error) {
	caseID, err := pulid.MustParse(input.CaseID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"caseId",
			errortypes.ErrInvalid,
			"Case is invalid",
		)
	}
	ptoID, err := optionalID(input.PTOID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"ptoId",
			errortypes.ErrInvalid,
			"Time off record is invalid",
		)
	}
	hours, err := parseDecimalField("hours", input.Hours, true)
	if err != nil {
		return nil, err
	}

	return &workerleaveservice.RecordDayRequest{
		TenantInfo: tenantInfo,
		CaseID:     caseID,
		UsedOn:     int64(input.UsedOn),
		Hours:      hours,
		PTOID:      ptoID,
		Notes:      stringValue(input.Notes),
		UserID:     userID,
	}, nil
}

func leaveDayUpdateRequest(
	input *gqlmodel.UpdateLeaveDayInput,
	tenantInfo pagination.TenantInfo,
	userID pulid.ID,
) (*workerleaveservice.UpdateDayRequest, error) {
	entryID, err := pulid.MustParse(input.EntryID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"entryId",
			errortypes.ErrInvalid,
			"Day is invalid",
		)
	}
	ptoID, err := optionalID(input.PTOID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"ptoId",
			errortypes.ErrInvalid,
			"Time off record is invalid",
		)
	}
	hours, err := parseNullDecimalField("hours", input.Hours)
	if err != nil {
		return nil, err
	}

	return &workerleaveservice.UpdateDayRequest{
		TenantInfo: tenantInfo,
		EntryID:    entryID,
		Hours:      nullDecimalValue(hours),
		Counts:     input.CountsAgainstEntitlement,
		PTOID:      ptoID,
		Notes:      input.Notes,
		UserID:     userID,
	}, nil
}

func leaveControlRequest(
	input *gqlmodel.UpdateLeaveControlInput,
	tenantInfo pagination.TenantInfo,
	userID pulid.ID,
) (*workerleaveservice.UpdateControlRequest, error) {
	entitlement, err := parseNullDecimalField("entitlementWeeks", input.EntitlementWeeks)
	if err != nil {
		return nil, err
	}
	caregiver, err := parseNullDecimalField(
		"militaryCaregiverWeeks",
		input.MilitaryCaregiverWeeks,
	)
	if err != nil {
		return nil, err
	}
	workweek, err := parseNullDecimalField("workweekHours", input.WorkweekHours)
	if err != nil {
		return nil, err
	}

	req := &workerleaveservice.UpdateControlRequest{
		TenantInfo:             tenantInfo,
		MeasurementMethod:      input.MeasurementMethod,
		EntitlementWeeks:       nullDecimalValue(entitlement),
		MilitaryCaregiverWeeks: nullDecimalValue(caregiver),
		WorkweekHours:          nullDecimalValue(workweek),
		UserID:                 userID,
	}
	if input.EligibilityMonths != nil {
		months := int32(*input.EligibilityMonths)
		req.EligibilityMonths = &months
	}
	if input.EligibilityHours != nil {
		hours := int32(*input.EligibilityHours)
		req.EligibilityHours = &hours
	}
	if input.CertificationDueDays != nil {
		days := int32(*input.CertificationDueDays)
		req.CertificationDueDays = &days
	}

	return req, nil
}
