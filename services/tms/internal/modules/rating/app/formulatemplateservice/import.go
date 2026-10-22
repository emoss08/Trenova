package formulatemplateservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const (
	importMaxTemplates = 50

	ExportVersion = "2.0"

	ImportConflictReject = "reject"
	ImportConflictRename = "rename"
)

type ImportTemplatePayload struct {
	Name                string         `json:"name"`
	Description         string         `json:"description"`
	Source              string         `json:"source"`
	Metadata            map[string]any `json:"metadata"`
	SourceTemplateID    *pulid.ID      `json:"sourceTemplateId"`
	SourceVersionNumber *int64         `json:"sourceVersionNumber"`
}

type ImportTemplatesRequest struct {
	ExportVersion string                   `json:"exportVersion"`
	Template      *ImportTemplatePayload   `json:"template"`
	Templates     []*ImportTemplatePayload `json:"templates"`
	OnConflict    string                   `json:"onConflict"`
	TenantInfo    pagination.TenantInfo    `json:"-"`
}

type ImportTemplatesResponse struct {
	Created []*formulatemplate.FormulaTemplate `json:"created"`
	Renamed map[string]string                  `json:"renamed,omitempty"`
}

func (s *Service) Import(
	ctx context.Context,
	req *ImportTemplatesRequest,
) (*ImportTemplatesResponse, error) {
	log := s.l.With(zap.String("operation", "Import"))

	payloads, err := normalizeImportRequest(req)
	if err != nil {
		return nil, err
	}

	renamed, err := s.resolveImportNames(ctx, req, payloads)
	if err != nil {
		return nil, err
	}

	entities, err := s.buildImportEntities(ctx, req, payloads)
	if err != nil {
		return nil, err
	}

	var created []*formulatemplate.FormulaTemplate
	err = s.db.WithTx(ctx, kernelports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		created = make([]*formulatemplate.FormulaTemplate, 0, len(entities))
		for _, entity := range entities {
			createdEntity, txErr := s.repo.Create(txCtx, entity)
			if txErr != nil {
				log.Error("failed to create imported template", zap.Error(txErr))
				return txErr
			}

			if txErr = s.createVersionSnapshot(
				txCtx, createdEntity, 1, req.TenantInfo.UserID, "Imported", nil,
			); txErr != nil {
				log.Error("failed to create version snapshot", zap.Error(txErr))
				return txErr
			}

			created = append(created, createdEntity)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	for _, entity := range created {
		s.logAuditAction(
			ctx,
			log,
			entity,
			permission.OpCreate,
			req.TenantInfo.UserID,
			nil,
			"Formula template imported",
		)
	}

	return &ImportTemplatesResponse{Created: created, Renamed: renamed}, nil
}

func normalizeImportRequest(req *ImportTemplatesRequest) ([]*ImportTemplatePayload, error) {
	multiErr := errortypes.NewMultiError()

	if req.ExportVersion != ExportVersion {
		multiErr.Add(
			"exportVersion",
			errortypes.ErrInvalid,
			"Unsupported export version: {0}", req.ExportVersion,
		)
	}

	if req.OnConflict == "" {
		req.OnConflict = ImportConflictReject
	}
	if req.OnConflict != ImportConflictReject && req.OnConflict != ImportConflictRename {
		multiErr.Add(
			"onConflict",
			errortypes.ErrInvalid,
			"Conflict policy must be reject or rename",
		)
	}

	payloads := req.Templates
	if req.Template != nil {
		payloads = append([]*ImportTemplatePayload{req.Template}, payloads...)
	}

	compact := make([]*ImportTemplatePayload, 0, len(payloads))
	for _, payload := range payloads {
		if payload != nil {
			compact = append(compact, payload)
		}
	}

	switch {
	case len(compact) == 0:
		multiErr.Add(
			"templates",
			errortypes.ErrRequired,
			"At least one template is required",
		)
	case len(compact) > importMaxTemplates:
		multiErr.Add(
			"templates",
			errortypes.ErrInvalid,
			"Cannot import more than {0} templates at once", importMaxTemplates,
		)
	}

	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return compact, nil
}

func (s *Service) resolveImportNames(
	ctx context.Context,
	req *ImportTemplatesRequest,
	payloads []*ImportTemplatePayload,
) (map[string]string, error) {
	multiErr := errortypes.NewMultiError()

	names := make([]string, 0, len(payloads))
	for _, payload := range payloads {
		if payload.Name != "" {
			names = append(names, payload.Name)
		}
	}

	existing, err := s.repo.FindByNames(ctx, formulatemplateports.GetFormulaTemplatesByNamesRequest{
		TenantInfo: req.TenantInfo,
		Names:      names,
	})
	if err != nil {
		return nil, err
	}

	taken := make(map[string]struct{}, len(existing)+len(payloads))
	for _, template := range existing {
		taken[template.Name] = struct{}{}
	}

	renamed := make(map[string]string)
	for index, payload := range payloads {
		if payload.Name == "" {
			continue
		}

		if _, conflict := taken[payload.Name]; !conflict {
			taken[payload.Name] = struct{}{}
			continue
		}

		if req.OnConflict == ImportConflictReject {
			multiErr.WithIndex("templates", index).Add(
				"name",
				errortypes.ErrInvalid,
				"A template with this name already exists",
			)
			continue
		}

		original := payload.Name
		candidate := original + " (Imported)"
		for suffix := 2; ; suffix++ {
			if _, stillTaken := taken[candidate]; !stillTaken {
				break
			}
			candidate = fmt.Sprintf("%s (Imported %d)", original, suffix)
		}

		payload.Name = candidate
		taken[candidate] = struct{}{}
		renamed[original] = candidate
	}

	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return renamed, nil
}

func (s *Service) buildImportEntities(
	ctx context.Context,
	req *ImportTemplatesRequest,
	payloads []*ImportTemplatePayload,
) ([]*formulatemplate.FormulaTemplate, error) {
	multiErr := errortypes.NewMultiError()

	sourceIDs := make([]pulid.ID, 0, len(payloads))
	for _, payload := range payloads {
		if payload.SourceTemplateID != nil && !payload.SourceTemplateID.IsNil() {
			sourceIDs = append(sourceIDs, *payload.SourceTemplateID)
		}
	}

	resolvableSources := make(map[pulid.ID]struct{}, len(sourceIDs))
	if len(sourceIDs) > 0 {
		sources, err := s.repo.GetByIDs(ctx, formulatemplateports.GetFormulaTemplatesByIDsRequest{
			TenantInfo:  req.TenantInfo,
			TemplateIDs: sourceIDs,
		})
		if err != nil {
			return nil, err
		}
		for _, source := range sources {
			resolvableSources[source.ID] = struct{}{}
		}
	}

	entities := make([]*formulatemplate.FormulaTemplate, 0, len(payloads))
	for index, payload := range payloads {
		entity := formulatemplate.Seed{
			OrganizationID:       req.TenantInfo.OrgID,
			BusinessUnitID:       req.TenantInfo.BuID,
			Name:                 payload.Name,
			Description:          payload.Description,
			Source:               payload.Source,
			Metadata:             payload.Metadata,
		}.Build()

		if payload.SourceTemplateID != nil && !payload.SourceTemplateID.IsNil() {
			if _, ok := resolvableSources[*payload.SourceTemplateID]; ok {
				sourceID := *payload.SourceTemplateID
				entity.SourceTemplateID = &sourceID
				entity.SourceVersionNumber = payload.SourceVersionNumber
			} else {
				if entity.Metadata == nil {
					entity.Metadata = map[string]any{}
				}
				entity.Metadata["importedSource"] = payload.SourceTemplateID.String()
			}
		}

		indexed := multiErr.WithIndex("templates", index)
		if vErr := s.prepareContent(ctx, entity); vErr != nil {
			if !errortypes.IsError(vErr) {
				return nil, vErr
			}
			addIndexedValidationError(indexed, vErr)
		}

		entities = append(entities, entity)
	}

	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return entities, nil
}

func addIndexedValidationError(indexed *errortypes.MultiError, err error) {
	var multiErr *errortypes.MultiError
	if errors.As(err, &multiErr) {
		for _, fieldErr := range multiErr.Errors {
			indexed.AddError(fieldErr)
		}
		return
	}
	var fieldErr *errortypes.Error
	if errors.As(err, &fieldErr) {
		indexed.AddError(fieldErr)
		return
	}
	indexed.Add(fieldSource, errortypes.ErrInvalid, err.Error())
}
