package captureresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/core/domain/capture"
	"github.com/emoss08/trenova/internal/core/services/captureservice"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

const defaultCaptureBatchPage = 25

func captureBatchConnection(
	result *pagination.CursorListResult[*capture.CaptureBatch],
) (*gqlmodel.CaptureBatchConnection, error) {
	page, err := base.EntityCursorConnection(
		result,
		func(node *capture.CaptureBatch, cursor string) *gqlmodel.CaptureBatchEdge {
			return &gqlmodel.CaptureBatchEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.CaptureBatchEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.CaptureBatchConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

// captureBatchSort is the queue's order as the fields the cursor sorts by. The
// id breaks ties, which the cursor adds on its own.
func captureBatchSort(sort *gqlmodel.CaptureBatchSort) []domaintypes.SortField {
	if sort == nil {
		return nil
	}

	switch *sort {
	case gqlmodel.CaptureBatchSortOldest:
		return []domaintypes.SortField{{Field: "createdAt", Direction: dbtype.SortDirectionAsc}}
	case gqlmodel.CaptureBatchSortExpiringSoonest:
		return []domaintypes.SortField{{Field: "retainUntil", Direction: dbtype.SortDirectionAsc}}
	case gqlmodel.CaptureBatchSortMostPages:
		return []domaintypes.SortField{
			{Field: "receivedPageCount", Direction: dbtype.SortDirectionDesc},
		}
	case gqlmodel.CaptureBatchSortNewest:
		return []domaintypes.SortField{{Field: "createdAt", Direction: dbtype.SortDirectionDesc}}
	default:
		return nil
	}
}

// optionalIDRef parses an optional id into the pointer form the capture
// domain keeps. Absent and blank are both no id.
func optionalIDRef(value *string) (*pulid.ID, error) {
	id, err := base.OptionalID(value)
	if err != nil || id.IsNil() {
		return nil, err
	}

	return &id, nil
}

func captureProfileSettings(input *gqlmodel.CaptureProfileInput) captureservice.ProfileSettings {
	return captureservice.ProfileSettings{
		Name:                input.Name,
		Description:         stringutils.FromPtr(input.Description),
		Status:              input.Status,
		IsDefault:           input.IsDefault,
		DPI:                 input.DPI,
		PixelType:           input.PixelType,
		Duplex:              input.Duplex,
		UseFeeder:           input.UseFeeder,
		DiscardBlankPages:   input.DiscardBlankPages,
		JPEGQuality:         input.JPEGQuality,
		ShowDriverUI:        input.ShowDriverUI,
		SeparatorStrategies: input.SeparatorStrategies,
		FixedPageCount:      input.FixedPageCount,
	}
}

func captureItemLayouts(
	input *gqlmodel.EditCaptureItemsInput,
) ([]captureservice.ItemLayout, error) {
	layouts := make([]captureservice.ItemLayout, 0, len(input.Items))
	for _, item := range input.Items {
		if item == nil {
			return nil, errortypes.NewValidationError("items", errortypes.ErrInvalid,
				"Every document needs its pages")
		}
		pageIDs, err := base.ParseIDs(item.PageIds)
		if err != nil {
			return nil, err
		}
		layouts = append(layouts, captureservice.ItemLayout{PageIDs: pageIDs})
	}

	return layouts, nil
}

func capturePageRotations(input []*gqlmodel.CapturePageRotationInput) (map[pulid.ID]int, error) {
	if len(input) == 0 {
		return nil, nil //nolint:nilnil // no rotations leaves every page as it is
	}

	rotations := make(map[pulid.ID]int, len(input))
	for _, rotation := range input {
		if rotation == nil {
			continue
		}
		pageID, err := pulid.MustParse(rotation.PageID)
		if err != nil {
			return nil, err
		}
		rotations[pageID] = rotation.Rotation
	}

	return rotations, nil
}

func captureCoverSheetSpecs(
	input []*gqlmodel.CaptureCoverSheetInput,
) ([]captureservice.CoverSheetSpec, error) {
	specs := make([]captureservice.CoverSheetSpec, 0, len(input))
	for _, sheet := range input {
		if sheet == nil {
			continue
		}
		spec := captureservice.CoverSheetSpec{TargetType: stringutils.FromPtr(sheet.TargetType)}

		var err error
		if spec.TargetID, err = optionalIDRef(sheet.TargetID); err != nil {
			return nil, err
		}
		if spec.DocumentTypeID, err = optionalIDRef(sheet.DocumentTypeID); err != nil {
			return nil, err
		}
		specs = append(specs, spec)
	}

	return specs, nil
}

func issuedCoverSheet(issued *captureservice.IssuedCoverSheet) *gqlmodel.CaptureCoverSheet {
	sheet := issued.Sheet

	return &gqlmodel.CaptureCoverSheet{
		ID:             sheet.ID.String(),
		TargetType:     sheet.TargetType,
		TargetID:       base.IDPtrFromPtr(sheet.TargetID),
		DocumentTypeID: base.IDPtrFromPtr(sheet.DocumentTypeID),
		Payload:        issued.Payload,
		QRCode:         issued.QRCode,
		ExpiresAt:      int(sheet.ExpiresAt),
		CreatedAt:      int(sheet.CreatedAt),
	}
}

// captureFiling parses where one document is to be filed. The single and the
// bulk mutation both come through here, so they cannot disagree on an id.
func captureFiling(
	itemID, targetType, targetID string,
	documentTypeID *string,
	version int,
) (captureservice.FileItemInput, error) {
	var in captureservice.FileItemInput

	var err error
	if in.ItemID, err = pulid.MustParse(itemID); err != nil {
		return in, err
	}
	if in.TargetID, err = pulid.MustParse(targetID); err != nil {
		return in, err
	}
	if in.DocumentTypeID, err = optionalIDRef(documentTypeID); err != nil {
		return in, err
	}
	in.TargetType = targetType
	in.Version = int64(version)

	return in, nil
}
