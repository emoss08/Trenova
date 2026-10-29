package documentsselectoptions

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/selectoption"
	"github.com/emoss08/trenova/internal/modules/documents/app/capturedevices"
	"github.com/emoss08/trenova/internal/modules/documents/app/captureprofiles"
	"github.com/emoss08/trenova/internal/modules/documents/domain/capture"
	"github.com/emoss08/trenova/internal/modules/documents/domain/captureprofile"
	"github.com/emoss08/trenova/internal/modules/reference/app/documenttypeservice"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/internal/modules/reference/ports/documenttypeports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/emailservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
)

type Sources struct {
	CaptureProfileService *captureprofiles.Service
	CaptureDeviceService  *capturedevices.Service
	DocumentTypeService   *documenttypeservice.Service
	EmailService          *emailservice.Service
}

func (r *Sources) DocumentType(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectoption.Item, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.DocumentTypeService.Get(
				ctx,
				documenttypeports.GetDocumentTypeByIDRequest{
					ID:         id,
					TenantInfo: req.TenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, documentTypeSelectOptionItem(entity))
		}

		return selectoption.Connection(items, len(items), 0)
	}

	result, err := r.DocumentTypeService.SelectOptions(
		ctx,
		req.SelectQuery,
	)
	if err != nil {
		return nil, err
	}

	return selectoption.ListConnection(
		result,
		req.SelectQuery.Pagination,
		documentTypeSelectOptionItem,
	)
}

func (r *Sources) EmailProfile(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectoption.Item, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.EmailService.GetProfile(
				ctx,
				emailports.GetEmailEntityRequest{
					ID:         id,
					TenantInfo: req.TenantInfo,
				},
			)
			if err != nil {
				return nil, err
			}
			items = append(items, emailProfileSelectOptionItem(entity))
		}

		return selectoption.Connection(items, len(items), 0)
	}

	result, err := r.EmailService.SelectProfileOptions(
		ctx,
		&emailports.EmailProfileSelectOptionsRequest{
			SelectQueryRequest: req.SelectQuery,
		},
	)
	if err != nil {
		return nil, err
	}

	return selectoption.ListConnection(
		result,
		req.SelectQuery.Pagination,
		emailProfileSelectOptionItem,
	)
}

func documentTypeSelectOptionItem(entity *documenttype.DocumentType) selectoption.Item {
	return selectoption.ItemFor(
		documentTypeSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func documentTypeSelectOption(entity *documenttype.DocumentType) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Code,
		Description: stringutils.Ptr(entity.Name),
		Meta: map[string]any{
			selectoption.MetaKeyCode: entity.Code,
			"color":                  entity.Color,
			"name":                   entity.Name,
		},
	}
}

func emailProfileSelectOptionItem(entity *email.Profile) selectoption.Item {
	return selectoption.ItemFor(
		emailProfileSelectOption(entity),
		entity.CreatedAt,
		entity.ID,
	)
}

func emailProfileSelectOption(entity *email.Profile) *gqlmodel.SelectOption {
	return &gqlmodel.SelectOption{
		ID:          entity.ID.String(),
		Label:       entity.Name,
		Description: stringutils.Ptr(entity.SenderEmail),
		Meta: map[string]any{
			"senderEmail": entity.SenderEmail,
			"provider":    string(entity.Provider),
		},
	}
}

func (r *Sources) CaptureDevice(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	query := req.SelectQuery.Query
	if len(req.IDs) > 0 {
		query = ""
	}

	result, err := r.CaptureDeviceService.ListDevices(ctx, &capturedevices.ListDevicesRequest{
		TenantInfo: req.TenantInfo,
		Filter: &pagination.QueryOptions{
			TenantInfo: req.TenantInfo,
			Pagination: pagination.Info{Limit: maxCaptureDevices},
			Query:      query,
		},
		Mine:   true,
		Status: capture.DeviceActive,
	})
	if err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	items := make([]selectoption.Item, 0, len(result.Items))
	for _, device := range withIDs(result.Items, req.IDs, func(d *capture.CaptureDevice) pulid.ID {
		return d.ID
	}) {
		items = append(items, captureDeviceSelectOptionItem(device, now))
	}

	return pageSelectOptionItems(items, req)
}

func (r *Sources) CaptureProfile(
	ctx context.Context,
	req *selectoption.Request,
) (*gqlmodel.SelectOptionConnection, error) {
	profiles, err := r.CaptureProfileService.AvailableProfiles(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	matched := withIDs(profiles, req.IDs, func(p *captureprofile.CaptureProfile) pulid.ID {
		return p.ID
	})
	query := strings.ToLower(strings.TrimSpace(req.SelectQuery.Query))

	items := make([]selectoption.Item, 0, len(matched))
	for _, profile := range matched {
		if len(req.IDs) == 0 && query != "" &&
			!strings.Contains(strings.ToLower(profile.Name), query) {
			continue
		}
		items = append(items, captureProfileSelectOptionItem(profile))
	}

	return pageSelectOptionItems(items, req)
}
