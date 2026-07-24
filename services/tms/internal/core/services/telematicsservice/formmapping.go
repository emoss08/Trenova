package telematicsservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/telematics"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type FormMappingItemInput struct {
	SourceFieldLabel     string
	TargetKind           telematics.FormMappingTargetKind
	TargetField          string
	TargetCustomFieldKey string
}

type SaveFormMappingRequest struct {
	TenantInfo   pagination.TenantInfo
	ID           pulid.ID
	Provider     string
	TemplateID   string
	TemplateName string
	Name         string
	Description  string
	Enabled      bool
	Version      int64
	Items        []FormMappingItemInput
}

func (s *Service) ListFormMappings(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*telematics.FormMapping, error) {
	return s.repo.ListFormMappings(ctx, &repositories.ListFormMappingsRequest{
		TenantInfo: tenantInfo,
	})
}

func (s *Service) GetFormMapping(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*telematics.FormMapping, error) {
	return s.repo.GetFormMapping(ctx, tenantInfo, id)
}

func (s *Service) SaveFormMapping(
	ctx context.Context,
	req *SaveFormMappingRequest,
) (*telematics.FormMapping, error) {
	if err := validateFormMappingRequest(req); err != nil {
		return nil, err
	}

	provider := req.Provider
	if provider == "" {
		provider = "Samsara"
	}

	mapping := &telematics.FormMapping{
		ID:             req.ID,
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		Provider:       provider,
		TemplateID:     req.TemplateID,
		TemplateName:   req.TemplateName,
		Name:           req.Name,
		Description:    req.Description,
		Enabled:        req.Enabled,
		Version:        req.Version,
	}

	items := make([]*telematics.FormMappingItem, 0, len(req.Items))
	for i := range req.Items {
		item := &req.Items[i]
		items = append(items, &telematics.FormMappingItem{
			SourceFieldLabel:     item.SourceFieldLabel,
			TargetKind:           item.TargetKind,
			TargetField:          item.TargetField,
			TargetCustomFieldKey: item.TargetCustomFieldKey,
		})
	}

	return s.repo.SaveFormMapping(ctx, mapping, items)
}

func (s *Service) DeleteFormMapping(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) error {
	return s.repo.DeleteFormMapping(ctx, tenantInfo, id)
}

func validateFormMappingRequest(req *SaveFormMappingRequest) error {
	multiErr := errortypes.NewMultiError()
	if req.Name == "" {
		multiErr.Add("name", errortypes.ErrRequired, "Name is required")
	}
	if req.TemplateID == "" {
		multiErr.Add("templateId", errortypes.ErrRequired, "Template is required")
	}
	for i := range req.Items {
		item := &req.Items[i]
		prefix := multiErr.WithIndex("items", i)
		if item.SourceFieldLabel == "" {
			prefix.Add("sourceFieldLabel", errortypes.ErrRequired, "Source field is required")
		}
		if !item.TargetKind.IsValid() {
			prefix.Add("targetKind", errortypes.ErrInvalid, "Invalid target kind")
			continue
		}
		validateFormMappingTarget(item, prefix)
	}
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func validateFormMappingTarget(item *FormMappingItemInput, prefix *errortypes.MultiError) {
	switch item.TargetKind {
	case telematics.FormMappingTargetShipmentField:
		if _, ok := telematics.ShipmentFieldTargets[item.TargetField]; !ok {
			prefix.Add("targetField", errortypes.ErrInvalid, "Unsupported shipment field target")
		}
	case telematics.FormMappingTargetStopField:
		if _, ok := telematics.StopFieldTargets[item.TargetField]; !ok {
			prefix.Add("targetField", errortypes.ErrInvalid, "Unsupported stop field target")
		}
	case telematics.FormMappingTargetShipmentCustomField:
		if item.TargetCustomFieldKey == "" {
			prefix.Add(
				"targetCustomFieldKey",
				errortypes.ErrRequired,
				"Custom field key is required",
			)
		}
	}
}
