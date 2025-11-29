package documenttemplate

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/internal/infrastructure/gotenberg"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/emoss08/trenova/pkg/utils/jsonutils"
	"github.com/emoss08/trenova/pkg/validator"
	"github.com/emoss08/trenova/pkg/validator/documenttemplatevalidator"
	"github.com/minio/minio-go/v7"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ServiceParams struct {
	fx.In

	Logger           *zap.Logger
	TemplateRepo     repositories.DocumentTemplateRepository
	GeneratedDocRepo repositories.GeneratedDocumentRepository
	AuditService     services.AuditService
	Validator        *documenttemplatevalidator.Validator
	Renderer         *Renderer
	Gotenberg        *gotenberg.Client
	Storage          *minio.Client
}

type Service struct {
	l                *zap.Logger
	templateRepo     repositories.DocumentTemplateRepository
	generatedDocRepo repositories.GeneratedDocumentRepository
	as               services.AuditService
	v                *documenttemplatevalidator.Validator
	renderer         *Renderer
	gotenberg        *gotenberg.Client
	storage          *minio.Client
}

//nolint:gocritic // dependency injection param
func NewService(p ServiceParams) *Service {
	return &Service{
		l:                p.Logger.Named("service.documenttemplate"),
		templateRepo:     p.TemplateRepo,
		generatedDocRepo: p.GeneratedDocRepo,
		as:               p.AuditService,
		v:                p.Validator,
		renderer:         p.Renderer,
		gotenberg:        p.Gotenberg,
		storage:          p.Storage,
	}
}

func (s *Service) ListTemplates(
	ctx context.Context,
	req *repositories.ListDocumentTemplateRequest,
) (*pagination.ListResult[*documenttemplate.DocumentTemplate], error) {
	return s.templateRepo.List(ctx, req)
}

func (s *Service) GetTemplate(
	ctx context.Context,
	req repositories.GetDocumentTemplateByIDRequest,
) (*documenttemplate.DocumentTemplate, error) {
	return s.templateRepo.GetByID(ctx, req)
}

func (s *Service) GetDefaultTemplate(
	ctx context.Context,
	req repositories.GetDefaultTemplateRequest,
) (*documenttemplate.DocumentTemplate, error) {
	return s.templateRepo.GetDefault(ctx, req)
}

func (s *Service) CreateTemplate(
	ctx context.Context,
	entity *documenttemplate.DocumentTemplate,
	userID pulid.ID,
) (*documenttemplate.DocumentTemplate, error) {
	log := s.l.With(
		zap.String("operation", "CreateTemplate"),
		zap.String("code", entity.Code),
		zap.String("buID", entity.BusinessUnitID.String()),
		zap.String("orgID", entity.OrganizationID.String()),
		zap.String("userID", userID.String()),
	)

	valCtx := &validator.ValidationContext{
		IsCreate: true,
		IsUpdate: false,
	}

	if err := s.v.Validate(ctx, valCtx, entity); err != nil {
		return nil, err
	}

	if entity.IsDefault {
		if err := s.templateRepo.ClearDefaultForType(
			ctx,
			entity.OrganizationID,
			entity.BusinessUnitID,
			entity.DocumentTypeID,
		); err != nil {
			log.Error("failed to clear default templates", zap.Error(err))
			return nil, err
		}
	}

	entity.CreatedByID = &userID

	createdEntity, err := s.templateRepo.Create(ctx, entity)
	if err != nil {
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceDocumentTemplate,
			ResourceID:     createdEntity.GetID(),
			Operation:      permission.OpCreate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(createdEntity),
			OrganizationID: createdEntity.OrganizationID,
			BusinessUnitID: createdEntity.BusinessUnitID,
		},
		audit.WithComment("Document template created"),
	)
	if err != nil {
		log.Error("failed to log document template creation", zap.Error(err))
	}

	return createdEntity, nil
}

func (s *Service) UpdateTemplate(
	ctx context.Context,
	entity *documenttemplate.DocumentTemplate,
	userID pulid.ID,
) (*documenttemplate.DocumentTemplate, error) {
	log := s.l.With(
		zap.String("operation", "UpdateTemplate"),
		zap.String("entityID", entity.ID.String()),
		zap.String("userID", userID.String()),
	)

	original, err := s.templateRepo.GetByID(ctx, repositories.GetDocumentTemplateByIDRequest{
		ID:    entity.ID,
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	})
	if err != nil {
		return nil, err
	}

	if !original.CanBeEdited() {
		return nil, ErrCannotEditArchivedTemplate
	}

	valCtx := &validator.ValidationContext{
		IsCreate: false,
		IsUpdate: true,
	}

	if err := s.v.Validate(ctx, valCtx, entity); err != nil {
		return nil, err
	}

	if entity.IsDefault && !original.IsDefault {
		if err = s.templateRepo.ClearDefaultForType(
			ctx,
			entity.OrganizationID,
			entity.BusinessUnitID,
			entity.DocumentTypeID,
		); err != nil {
			log.Error("failed to clear default templates", zap.Error(err))
			return nil, err
		}
	}

	entity.UpdatedByID = &userID

	updatedEntity, err := s.templateRepo.Update(ctx, entity)
	if err != nil {
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceDocumentTemplate,
			ResourceID:     updatedEntity.GetID(),
			Operation:      permission.OpUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedEntity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updatedEntity.OrganizationID,
			BusinessUnitID: updatedEntity.BusinessUnitID,
		},
		audit.WithComment("Document template updated"),
	)
	if err != nil {
		log.Error("failed to log document template update", zap.Error(err))
	}

	return updatedEntity, nil
}

func (s *Service) DeleteTemplate(
	ctx context.Context,
	req repositories.GetDocumentTemplateByIDRequest,
) error {
	log := s.l.With(
		zap.String("operation", "DeleteTemplate"),
		zap.String("entityID", req.ID.String()),
		zap.String("userID", req.UserID.String()),
	)

	entity, err := s.templateRepo.GetByID(ctx, req)
	if err != nil {
		return err
	}

	if !entity.CanBeDeleted() {
		return ErrCannotDeleteSystemTemplate
	}

	if err = s.templateRepo.Delete(ctx, entity); err != nil {
		return err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceDocumentTemplate,
			ResourceID:     entity.GetID(),
			Operation:      permission.OpDelete,
			UserID:         req.UserID,
			PreviousState:  jsonutils.MustToJSON(entity),
			OrganizationID: entity.OrganizationID,
			BusinessUnitID: entity.BusinessUnitID,
		},
		audit.WithComment("Document template deleted"),
	)
	if err != nil {
		log.Error("failed to log document template deletion", zap.Error(err))
	}

	return nil
}

type GenerateDocumentRequest struct {
	TemplateID     pulid.ID
	ReferenceType  string
	ReferenceID    pulid.ID
	Data           any
	OrgID          pulid.ID
	BuID           pulid.ID
	UserID         pulid.ID
	DocumentTypeID pulid.ID
}

func (s *Service) GenerateDocument(
	ctx context.Context,
	req *GenerateDocumentRequest,
) (*documenttemplate.GeneratedDocument, error) {
	log := s.l.With(
		zap.String("operation", "GenerateDocument"),
		zap.String("templateID", req.TemplateID.String()),
		zap.String("referenceType", req.ReferenceType),
		zap.String("referenceID", req.ReferenceID.String()),
	)

	tmpl, err := s.templateRepo.GetByID(ctx, repositories.GetDocumentTemplateByIDRequest{
		ID:    req.TemplateID,
		OrgID: req.OrgID,
		BuID:  req.BuID,
	})
	if err != nil {
		return nil, fmt.Errorf("get template: %w", err)
	}

	if tmpl.Status != documenttemplate.TemplateStatusActive {
		return nil, ErrTemplateNotActive
	}

	html, err := s.renderer.RenderHTML(tmpl, req.Data)
	if err != nil {
		return nil, fmt.Errorf("render html: %w", err)
	}

	headerHTML, err := s.renderer.RenderHeaderHTML(tmpl, req.Data)
	if err != nil {
		return nil, fmt.Errorf("render header: %w", err)
	}

	footerHTML, err := s.renderer.RenderFooterHTML(tmpl, req.Data)
	if err != nil {
		return nil, fmt.Errorf("render footer: %w", err)
	}

	pdfOpts := s.renderer.GetPDFOptions(tmpl)

	var pdfData []byte
	if len(headerHTML) > 0 || len(footerHTML) > 0 {
		pdfData, err = s.gotenberg.HTMLToPDFWithHeaderFooter(
			ctx,
			html,
			headerHTML,
			footerHTML,
			pdfOpts,
		)
	} else {
		pdfData, err = s.gotenberg.HTMLToPDF(ctx, html, pdfOpts)
	}
	if err != nil {
		return nil, fmt.Errorf("generate pdf: %w", err)
	}

	hash := sha256.Sum256(pdfData)
	checksum := hex.EncodeToString(hash[:])

	fileName := fmt.Sprintf(
		"%s_%s_%d.pdf",
		req.ReferenceType,
		req.ReferenceID.String(),
		utils.NowUnix(),
	)
	filePath := fmt.Sprintf("documents/%s/%s/%s", req.OrgID.String(), req.ReferenceType, fileName)

	genDoc := &documenttemplate.GeneratedDocument{
		BusinessUnitID: req.BuID,
		OrganizationID: req.OrgID,
		DocumentTypeID: req.DocumentTypeID,
		TemplateID:     req.TemplateID,
		ReferenceType:  req.ReferenceType,
		ReferenceID:    req.ReferenceID,
		FileName:       fileName,
		FilePath:       filePath,
		FileSize:       int64(len(pdfData)),
		MimeType:       "application/pdf",
		Checksum:       checksum,
		Status:         documenttemplate.GenerationStatusCompleted,
		GeneratedByID:  &req.UserID,
	}

	now := utils.NowUnix()
	genDoc.GeneratedAt = &now

	createdDoc, err := s.generatedDocRepo.Create(ctx, genDoc)
	if err != nil {
		return nil, fmt.Errorf("create generated document: %w", err)
	}

	log.Info("document generated successfully",
		zap.String("documentID", createdDoc.ID.String()),
		zap.Int64("fileSize", createdDoc.FileSize),
	)

	return createdDoc, nil
}

func (s *Service) ListGeneratedDocuments(
	ctx context.Context,
	req *repositories.ListGeneratedDocumentRequest,
) (*pagination.ListResult[*documenttemplate.GeneratedDocument], error) {
	return s.generatedDocRepo.List(ctx, req)
}

func (s *Service) GetGeneratedDocument(
	ctx context.Context,
	req repositories.GetGeneratedDocumentByIDRequest,
) (*documenttemplate.GeneratedDocument, error) {
	return s.generatedDocRepo.GetByID(ctx, req)
}

func (s *Service) GetGeneratedDocumentsByReference(
	ctx context.Context,
	req *repositories.GetByReferenceRequest,
) ([]*documenttemplate.GeneratedDocument, error) {
	return s.generatedDocRepo.GetByReference(ctx, req)
}

func (s *Service) DeleteGeneratedDocument(
	ctx context.Context,
	req repositories.GetGeneratedDocumentByIDRequest,
) error {
	log := s.l.With(
		zap.String("operation", "DeleteGeneratedDocument"),
		zap.String("entityID", req.ID.String()),
		zap.String("userID", req.UserID.String()),
	)

	entity, err := s.generatedDocRepo.GetByID(ctx, req)
	if err != nil {
		return err
	}

	if err = s.generatedDocRepo.Delete(ctx, entity); err != nil {
		return err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceGeneratedDocument,
			ResourceID:     entity.GetID(),
			Operation:      permission.OpDelete,
			UserID:         req.UserID,
			PreviousState:  jsonutils.MustToJSON(entity),
			OrganizationID: entity.OrganizationID,
			BusinessUnitID: entity.BusinessUnitID,
		},
		audit.WithComment("Generated document deleted"),
	)
	if err != nil {
		log.Error("failed to log generated document deletion", zap.Error(err))
	}

	return nil
}

func (s *Service) PreviewTemplate(
	ctx context.Context,
	tmpl *documenttemplate.DocumentTemplate,
	data any,
) ([]byte, error) {
	html, err := s.renderer.RenderHTML(tmpl, data)
	if err != nil {
		return nil, fmt.Errorf("render html: %w", err)
	}

	headerHTML, err := s.renderer.RenderHeaderHTML(tmpl, data)
	if err != nil {
		return nil, fmt.Errorf("render header: %w", err)
	}

	footerHTML, err := s.renderer.RenderFooterHTML(tmpl, data)
	if err != nil {
		return nil, fmt.Errorf("render footer: %w", err)
	}

	pdfOpts := s.renderer.GetPDFOptions(tmpl)

	var pdfData []byte
	if len(headerHTML) > 0 || len(footerHTML) > 0 {
		pdfData, err = s.gotenberg.HTMLToPDFWithHeaderFooter(
			ctx,
			html,
			headerHTML,
			footerHTML,
			pdfOpts,
		)
	} else {
		pdfData, err = s.gotenberg.HTMLToPDF(ctx, html, pdfOpts)
	}
	if err != nil {
		return nil, fmt.Errorf("generate pdf: %w", err)
	}

	return pdfData, nil
}
