package captureservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/capture"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/emoss08/trenova/shared/tokenutils"
)

// maxCoverSheetsPerRequest bounds one print run: a packet's worth of sheets
// for a shipment is a handful, and a thousand would be somebody's loop.
const maxCoverSheetsPerRequest = 50

// CoverSheetSpec is one sheet to issue. A spec with no record issues a plain
// separator that divides a stack and routes nothing.
type CoverSheetSpec struct {
	TargetType     string    `json:"targetType"`
	TargetID       *pulid.ID `json:"targetId"`
	DocumentTypeID *pulid.ID `json:"documentTypeId"`
}

type CreateCoverSheetsInput struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	Sheets     []CoverSheetSpec      `json:"sheets"`
}

// IssuedCoverSheet is a sheet, the payload its QR code must carry and the code
// drawn from it. The payload is returned once, to be printed; only its hash is
// kept.
type IssuedCoverSheet struct {
	Sheet   *capture.CaptureCoverSheet `json:"sheet"`
	Payload string                     `json:"payload"`
	QRCode  *services.CaptureQRCode    `json:"qrCode"`
}

// CreateCoverSheets issues sheets for printing. Each routing sheet is checked
// as a filing would be, so a sheet cannot be printed for a record its issuer
// could not file onto.
func (s *Service) CreateCoverSheets(
	ctx context.Context,
	in *CreateCoverSheetsInput,
) ([]IssuedCoverSheet, error) {
	if _, err := s.require(
		ctx,
		in.TenantInfo,
		permission.ResourceCaptureBatch,
		permission.OpCreate,
	); err != nil {
		return nil, err
	}
	if _, err := s.requireEnabled(ctx, in.TenantInfo); err != nil {
		return nil, err
	}
	if len(in.Sheets) == 0 || len(in.Sheets) > maxCoverSheetsPerRequest {
		return nil, errortypes.NewValidationError("sheets", errortypes.ErrInvalid,
			"Print between 1 and {0} cover sheets at a time", maxCoverSheetsPerRequest)
	}

	now := timeutils.NowUnix()
	issued := make([]IssuedCoverSheet, 0, len(in.Sheets))
	sheets := make([]*capture.CaptureCoverSheet, 0, len(in.Sheets))
	for _, spec := range in.Sheets {
		target := capture.Target{
			ResourceType:   spec.TargetType,
			ResourceID:     spec.TargetID,
			DocumentTypeID: spec.DocumentTypeID,
		}
		if target.HasRecord() || target.DocumentTypeID != nil {
			if err := s.checkTarget(
				ctx,
				in.TenantInfo,
				target,
				"targetType",
				"targetId",
			); err != nil {
				return nil, err
			}
		}

		token, hash, err := tokenutils.New()
		if err != nil {
			return nil, err
		}

		sheet := &capture.CaptureCoverSheet{
			ID:             pulid.MustNew("ccs_"),
			OrganizationID: in.TenantInfo.OrgID,
			BusinessUnitID: in.TenantInfo.BuID,
			TokenHash:      hash,
			TargetType:     spec.TargetType,
			TargetID:       spec.TargetID,
			DocumentTypeID: spec.DocumentTypeID,
			IssuedByID:     in.TenantInfo.UserID,
			ExpiresAt:      now + capture.CoverSheetLifetimeSeconds,
		}

		multiErr := errortypes.NewMultiError()
		sheet.Validate(multiErr)
		if multiErr.HasErrors() {
			return nil, multiErr
		}

		payload := capture.CoverSheetPayload(token)
		code, err := s.qrCodes.Encode(payload)
		if err != nil {
			return nil, err
		}

		sheets = append(sheets, sheet)
		issued = append(issued, IssuedCoverSheet{Sheet: sheet, Payload: payload, QRCode: code})
	}

	if err := s.coverSheets.CreateMany(ctx, sheets); err != nil {
		return nil, err
	}

	return issued, nil
}
