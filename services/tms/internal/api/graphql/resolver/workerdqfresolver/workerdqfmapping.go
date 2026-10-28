package workerdqfresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerdqfservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/employment"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func employmentVerificationFromInput(
	input *gqlmodel.RecordEmploymentVerificationInput,
	tenantInfo pagination.TenantInfo,
) (*employment.WorkerEmploymentVerification, error) {
	workerID, err := pulid.MustParse(input.WorkerID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"workerId",
			errortypes.ErrInvalid,
			"Worker is invalid",
		)
	}

	entity := &employment.WorkerEmploymentVerification{
		OrganizationID:    tenantInfo.OrgID,
		BusinessUnitID:    tenantInfo.BuID,
		WorkerID:          workerID,
		EmployerName:      input.EmployerName,
		EmployerDOTNumber: gqlscalar.StringValue(input.EmployerDOTNumber),
		EmployerMCNumber:  gqlscalar.StringValue(input.EmployerMcNumber),
		ContactName:       gqlscalar.StringValue(input.ContactName),
		ContactPhone:      gqlscalar.StringValue(input.ContactPhone),
		ContactEmail:      gqlscalar.StringValue(input.ContactEmail),
		EmployedFrom:      gqlscalar.Int64Ptr(input.EmployedFrom),
		EmployedTo:        gqlscalar.Int64Ptr(input.EmployedTo),
		// A previous carrier is DOT-regulated unless the office says
		// otherwise, which is the case for all but a handful of records.
		WasDOTRegulated: true,
		RequestedAt:     gqlscalar.Int64Ptr(input.RequestedAt),
		Notes:           gqlscalar.StringValue(input.Notes),
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
	documentID, err := gqlid.OptionalID(input.DocumentID)
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
		EmployedFrom:                  gqlscalar.Int64Ptr(input.EmployedFrom),
		EmployedTo:                    gqlscalar.Int64Ptr(input.EmployedTo),
		WasDOTRegulated:               input.WasDOTRegulated,
		Status:                        input.Status,
		Method:                        input.Method,
		RequestedAt:                   gqlscalar.Int64Ptr(input.RequestedAt),
		ResponseReceivedAt:            gqlscalar.Int64Ptr(input.ResponseReceivedAt),
		DrugAlcoholResponseReceivedAt: gqlscalar.Int64Ptr(input.DrugAlcoholResponseReceivedAt),
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
