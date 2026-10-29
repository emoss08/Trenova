package edisetup

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
)

func (s *Service) ResetControlNumber(
	ctx context.Context,
	req *edipartnerports.ResetEDIControlNumberRequest,
	actor *permissionports.RequestActor,
) (*edidocument.EDIControlNumberSequence, error) {
	if err := validateControlNumberRequest(req); err != nil {
		return nil, err
	}

	sequence, err := s.controlNumberRepo.ResetControlNumber(ctx, req)
	if err != nil {
		return nil, err
	}
	s.logAction(
		sequence,
		actor,
		permission.OpUpdate,
		nil,
		sequence,
		fmt.Sprintf(
			"EDI %s control number sequence reset to %d",
			sequence.Kind,
			req.NextValue,
		),
	)
	return sequence, nil
}

func (s *Service) StartControlNumber(
	ctx context.Context,
	req *edipartnerports.ResetEDIControlNumberRequest,
	actor *permissionports.RequestActor,
) (*edidocument.EDIControlNumberSequence, error) {
	if err := validateControlNumberRequest(req); err != nil {
		return nil, err
	}

	sequence, err := s.controlNumberRepo.StartControlNumber(ctx, req)
	if err != nil {
		return nil, err
	}
	s.logAction(
		sequence,
		actor,
		permission.OpUpdate,
		nil,
		sequence,
		fmt.Sprintf(
			"EDI %s control numbers start at %d",
			sequence.Kind,
			sequence.NextValue,
		),
	)
	return sequence, nil
}

func validateControlNumberRequest(req *edipartnerports.ResetEDIControlNumberRequest) error {
	multiErr := errortypes.NewMultiError()
	if req.PartnerID.IsNil() {
		multiErr.Add("partnerId", errortypes.ErrRequired, "EDI partner is required")
	}
	if req.DocumentTypeID.IsNil() {
		multiErr.Add("documentTypeId", errortypes.ErrRequired, "Document type is required")
	}
	switch req.Kind {
	case edidocument.ControlNumberKindInterchange,
		edidocument.ControlNumberKindGroup,
		edidocument.ControlNumberKindTransaction:
	default:
		multiErr.Add(
			"kind",
			errortypes.ErrInvalid,
			"Control number kind must be Interchange, Group, or Transaction",
		)
	}
	if req.NextValue < 1 {
		multiErr.Add("nextValue", errortypes.ErrInvalid, "Next value must be at least 1")
	}
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}
