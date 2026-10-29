package edisetup

import (
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
)

func mapEDIConnectionConstraint(err error) error {
	if !dberror.IsUniqueConstraintViolation(err) {
		return err
	}

	multiErr := errortypes.NewMultiError()
	switch dberror.ExtractConstraintName(err) {
	case "idx_edi_connections_internal_open":
		multiErr.Add(
			"targetOrganizationId",
			errortypes.ErrDuplicate,
			"An open internal EDI connection already exists for these organizations",
		)
	default:
		return err
	}

	return multiErr
}

func mapEDICommunicationProfileConstraint(err error) error {
	if !dberror.IsUniqueConstraintViolation(err) {
		return err
	}

	multiErr := errortypes.NewMultiError()
	switch dberror.ExtractConstraintName(err) {
	case "idx_edi_communication_profiles_name_org":
		multiErr.Add(
			"name",
			errortypes.ErrDuplicate,
			"EDI communication profile with this name already exists",
		)
	case "uq_edi_communication_profiles_active_as2_identifiers":
		multiErr.Add(
			"config.localAS2Id",
			errortypes.ErrDuplicate,
			"Another active AS2 profile already uses this local and partner AS2 ID pair",
		)
	default:
		return err
	}

	return multiErr
}
