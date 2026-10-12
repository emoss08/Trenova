package inboundreplyservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type TemplateInput struct {
	Name string
	Body string
}

type CreateTemplateRequest struct {
	TenantInfo pagination.TenantInfo
	UserID     pulid.ID
	Input      TemplateInput
}

type UpdateTemplateRequest struct {
	TenantInfo pagination.TenantInfo
	ID         pulid.ID
	Version    int64
	Input      TemplateInput
}

type RenderTemplateRequest struct {
	TenantInfo pagination.TenantInfo
	TemplateID pulid.ID
	MessageID  pulid.ID
}

func (s *Service) ListTemplates(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*triage.ReplyTemplate, error) {
	return s.templates.List(ctx, tenantInfo)
}

func (s *Service) CreateTemplate(
	ctx context.Context,
	req *CreateTemplateRequest,
) (*triage.ReplyTemplate, error) {
	entity := &triage.ReplyTemplate{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		Name:           strings.TrimSpace(req.Input.Name),
		Body:           strings.TrimSpace(req.Input.Body),
		CreatedByID:    req.UserID,
	}
	if err := s.validateTemplate(ctx, req.TenantInfo, entity); err != nil {
		return nil, err
	}

	count, err := s.templates.Count(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	if count >= triage.MaxTemplatesPerTenant {
		return nil, errortypes.NewBusinessError(
			"An organization keeps at most {0} reply templates. Delete one you no longer use first.",
			triage.MaxTemplatesPerTenant,
		)
	}

	created, err := s.templates.Create(ctx, entity)
	if err != nil {
		return nil, templateWriteError(err, entity.Name)
	}

	return created, nil
}

func (s *Service) UpdateTemplate(
	ctx context.Context,
	req *UpdateTemplateRequest,
) (*triage.ReplyTemplate, error) {
	entity, err := s.templates.GetByID(ctx, inboundmessageports.GetTriageReplyTemplateRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	entity.Name = strings.TrimSpace(req.Input.Name)
	entity.Body = strings.TrimSpace(req.Input.Body)
	entity.Version = req.Version
	if err = s.validateTemplate(ctx, req.TenantInfo, entity); err != nil {
		return nil, err
	}

	updated, err := s.templates.Update(ctx, entity)
	if err != nil {
		return nil, templateWriteError(err, entity.Name)
	}

	return updated, nil
}

func (s *Service) DeleteTemplate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) error {
	return s.templates.Delete(ctx, inboundmessageports.GetTriageReplyTemplateRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	})
}

func (s *Service) RenderTemplate(ctx context.Context, req *RenderTemplateRequest) (string, error) {
	template, err := s.templates.GetByID(ctx, inboundmessageports.GetTriageReplyTemplateRequest{
		ID:         req.TemplateID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return "", err
	}

	message, err := s.message(ctx, req.TenantInfo, req.MessageID, false)
	if err != nil {
		return "", err
	}

	return RenderTemplate(template.Body, &s.messageContext(ctx, message).reply), nil
}

func (s *Service) validateTemplate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	entity *triage.ReplyTemplate,
) error {
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}

	taken, err := s.templates.NameTaken(ctx, tenantInfo, entity.Name, entity.ID)
	if err != nil {
		return err
	}
	if taken {
		return duplicateTemplateName(entity.Name)
	}

	return nil
}

func templateWriteError(err error, name string) error {
	if dberror.IsUniqueConstraintViolation(err) {
		return duplicateTemplateName(name)
	}

	return err
}

func duplicateTemplateName(name string) error {
	return errortypes.NewValidationError(
		"name",
		errortypes.ErrDuplicate,
		"A template called {0} already exists",
		name,
	)
}
