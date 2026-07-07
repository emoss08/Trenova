package ediservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type SaveEDITestCaseRequest struct {
	TenantInfo               pagination.TenantInfo
	ID                       pulid.ID
	PartnerDocumentProfileID pulid.ID
	Name                     string
	Description              string
	Payload                  edi.DocumentPayload
	ExpectedWarnings         int
	ExpectedErrors           int
	Version                  int64
}

func (s *Service) ListTestCasesCursor(
	ctx context.Context,
	req *repositories.ListEDITestCasesRequest,
) (*pagination.CursorListResult[*edi.EDITestCase], error) {
	return s.testCaseRepo.ListTestCasesCursor(ctx, req)
}

func (s *Service) GetTestCase(
	ctx context.Context,
	req repositories.GetEDITestCaseByIDRequest,
) (*edi.EDITestCase, error) {
	return s.testCaseRepo.GetTestCaseByID(ctx, req)
}

func (s *Service) CreateTestCase(
	ctx context.Context,
	req *SaveEDITestCaseRequest,
) (*edi.EDITestCase, error) {
	if err := s.validateTestCase(ctx, req); err != nil {
		return nil, err
	}
	return s.testCaseRepo.CreateTestCase(ctx, &edi.EDITestCase{
		BusinessUnitID:           req.TenantInfo.BuID,
		OrganizationID:           req.TenantInfo.OrgID,
		PartnerDocumentProfileID: req.PartnerDocumentProfileID,
		Name:                     req.Name,
		Description:              req.Description,
		Payload:                  req.Payload,
		ExpectedWarnings:         req.ExpectedWarnings,
		ExpectedErrors:           req.ExpectedErrors,
	})
}

func (s *Service) UpdateTestCase(
	ctx context.Context,
	req *SaveEDITestCaseRequest,
) (*edi.EDITestCase, error) {
	if req == nil || req.ID.IsNil() {
		return nil, errortypes.NewValidationError(
			"id",
			errortypes.ErrRequired,
			"EDI test case ID is required",
		)
	}
	if err := s.validateTestCase(ctx, req); err != nil {
		return nil, err
	}
	entity, err := s.testCaseRepo.GetTestCaseByID(ctx, repositories.GetEDITestCaseByIDRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	entity.PartnerDocumentProfileID = req.PartnerDocumentProfileID
	entity.Name = req.Name
	entity.Description = req.Description
	entity.Payload = req.Payload
	entity.ExpectedWarnings = req.ExpectedWarnings
	entity.ExpectedErrors = req.ExpectedErrors
	entity.Version = req.Version
	return s.testCaseRepo.UpdateTestCase(ctx, entity)
}

func (s *Service) DeleteTestCase(
	ctx context.Context,
	req repositories.DeleteEDITestCaseRequest,
) error {
	return s.testCaseRepo.DeleteTestCase(ctx, req)
}

func (s *Service) validateTestCase(ctx context.Context, req *SaveEDITestCaseRequest) error {
	if req == nil {
		return errortypes.NewValidationError(
			"payload",
			errortypes.ErrRequired,
			"EDI test case request is required",
		)
	}
	multiErr := errortypes.NewMultiError()
	if req.Name == "" {
		multiErr.Add("name", errortypes.ErrRequired, "Name is required")
	}
	if req.PartnerDocumentProfileID.IsNil() {
		multiErr.Add(
			"partnerDocumentProfileId",
			errortypes.ErrRequired,
			"Document profile is required",
		)
	}
	req.Payload.Normalize()
	if !req.Payload.HasBranch() {
		multiErr.Add(
			"payload",
			errortypes.ErrRequired,
			"Payload must contain at least one transaction branch",
		)
	}
	if req.ExpectedWarnings < 0 {
		multiErr.Add(
			"expectedWarnings",
			errortypes.ErrInvalid,
			"Expected warnings cannot be negative",
		)
	}
	if req.ExpectedErrors < 0 {
		multiErr.Add("expectedErrors", errortypes.ErrInvalid, "Expected errors cannot be negative")
	}
	if multiErr.HasErrors() {
		return multiErr
	}
	if _, err := s.documentProfileRepo.GetPartnerDocumentProfileByID(
		ctx,
		repositories.GetEDIPartnerDocumentProfileByIDRequest{
			ID:         req.PartnerDocumentProfileID,
			TenantInfo: req.TenantInfo,
		},
	); err != nil {
		if errortypes.IsNotFoundError(err) {
			return errortypes.NewValidationError(
				"partnerDocumentProfileId",
				errortypes.ErrInvalid,
				"Document profile does not exist",
			)
		}
		return err
	}
	return nil
}
