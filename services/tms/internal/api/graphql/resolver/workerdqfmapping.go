package resolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/services/workerdqfservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func employmentVerificationFromInput(
	input *gqlmodel.RecordEmploymentVerificationInput,
	tenantInfo pagination.TenantInfo,
) (*worker.WorkerEmploymentVerification, error) {
	workerID, err := pulid.MustParse(input.WorkerID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"workerId",
			errortypes.ErrInvalid,
			"Worker is invalid",
		)
	}

	entity := &worker.WorkerEmploymentVerification{
		OrganizationID:    tenantInfo.OrgID,
		BusinessUnitID:    tenantInfo.BuID,
		WorkerID:          workerID,
		EmployerName:      input.EmployerName,
		EmployerDOTNumber: stringValue(input.EmployerDOTNumber),
		EmployerMCNumber:  stringValue(input.EmployerMcNumber),
		ContactName:       stringValue(input.ContactName),
		ContactPhone:      stringValue(input.ContactPhone),
		ContactEmail:      stringValue(input.ContactEmail),
		EmployedFrom:      int64Ptr(input.EmployedFrom),
		EmployedTo:        int64Ptr(input.EmployedTo),
		// A previous carrier is DOT-regulated unless the office says
		// otherwise, which is the case for all but a handful of records.
		WasDOTRegulated: true,
		RequestedAt:     int64Ptr(input.RequestedAt),
		Notes:           stringValue(input.Notes),
	}
	if input.WasDOTRegulated != nil {
		entity.WasDOTRegulated = *input.WasDOTRegulated
	}
	if input.Status != nil {
		entity.Status = *input.Status
	}
	if input.Method != nil {
		entity.Method = *input.Method
	}

	return entity, nil
}

func employmentVerificationUpdateRequest(
	input *gqlmodel.UpdateEmploymentVerificationInput,
	tenantInfo pagination.TenantInfo,
	userID pulid.ID,
) (*workerdqfservice.UpdateVerificationRequest, error) {
	verificationID, err := pulid.MustParse(input.VerificationID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"verificationId",
			errortypes.ErrInvalid,
			"Investigation is invalid",
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

	req := &workerdqfservice.UpdateVerificationRequest{
		TenantInfo:                    tenantInfo,
		VerificationID:                verificationID,
		EmployerName:                  input.EmployerName,
		EmployerDOTNumber:             input.EmployerDOTNumber,
		EmployerMCNumber:              input.EmployerMcNumber,
		ContactName:                   input.ContactName,
		ContactPhone:                  input.ContactPhone,
		ContactEmail:                  input.ContactEmail,
		EmployedFrom:                  int64Ptr(input.EmployedFrom),
		EmployedTo:                    int64Ptr(input.EmployedTo),
		WasDOTRegulated:               input.WasDOTRegulated,
		Status:                        input.Status,
		Method:                        input.Method,
		RequestedAt:                   int64Ptr(input.RequestedAt),
		ResponseReceivedAt:            int64Ptr(input.ResponseReceivedAt),
		DrugAlcoholResponseReceivedAt: int64Ptr(input.DrugAlcoholResponseReceivedAt),
		HadAccidents:                  input.HadAccidents,
		HadDrugAlcoholViolations:      input.HadDrugAlcoholViolations,
		Findings:                      input.Findings,
		Notes:                         input.Notes,
		DocumentID:                    documentID,
		UserID:                        userID,
	}
	if input.AccidentCount != nil {
		count := int32(*input.AccidentCount)
		req.AccidentCount = &count
	}

	return req, nil
}
