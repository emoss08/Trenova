package formulatemplateservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/ratetypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const (
	importMaxTemplates = 50
	importMaxTestCases = 100

	maxImportTestCaseNameLength = 100

	ImportConflictReject = "reject"
	ImportConflictRename = "rename"
)

var supportedImportVersions = map[string]struct{}{
	"1.0": {},
	"1.1": {},
	"1.2": {},
	"1.3": {},
}

type ImportTemplatePayload struct {
	Name                 string                              `json:"name"`
	Description          string                              `json:"description"`
	Type                 formulatemplate.TemplateType        `json:"type"`
	Expression           string                              `json:"expression"`
	SchemaID             string                              `json:"schemaId"`
	VariableDefinitions  []*formulatypes.VariableDefinition  `json:"variableDefinitions"`
	BreakdownDefinitions []*formulatypes.BreakdownDefinition `json:"breakdownDefinitions"`
	MinCharge            decimal.NullDecimal                 `json:"minCharge"`
	MaxCharge            decimal.NullDecimal                 `json:"maxCharge"`
	RoundingMode         ratetypes.RoundingMode              `json:"roundingMode"`
	RoundingPrecision    *int32                              `json:"roundingPrecision"`
	Metadata             map[string]any                      `json:"metadata"`
	SourceTemplateID     *pulid.ID                           `json:"sourceTemplateId"`
	SourceVersionNumber  *int64                              `json:"sourceVersionNumber"`
	TestCases            []*TestCaseInput                    `json:"testCases"`
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
	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		created = make([]*formulatemplate.FormulaTemplate, 0, len(entities))
		for index, entity := range entities {
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

			if txErr = s.createImportedTestCases(
				txCtx, createdEntity, req.TenantInfo, payloads[index].TestCases,
			); txErr != nil {
				log.Error("failed to create imported test scenarios", zap.Error(txErr))
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

	if _, ok := supportedImportVersions[req.ExportVersion]; !ok {
		multiErr.Add(
			"exportVersion",
			errortypes.ErrInvalid,
			"Unsupported export version: "+req.ExportVersion,
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
			fmt.Sprintf("Cannot import more than %d templates at once", importMaxTemplates),
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

	existing, err := s.repo.FindByNames(ctx, repositories.GetFormulaTemplatesByNamesRequest{
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
		sources, err := s.repo.GetByIDs(ctx, repositories.GetFormulaTemplatesByIDsRequest{
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
			Type:                 payload.Type,
			Expression:           payload.Expression,
			SchemaID:             payload.SchemaID,
			VariableDefinitions:  payload.VariableDefinitions,
			BreakdownDefinitions: payload.BreakdownDefinitions,
			MinCharge:            payload.MinCharge,
			MaxCharge:            payload.MaxCharge,
			RoundingMode:         payload.RoundingMode,
			RoundingPrecision:    importedPrecision(payload),
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
		entity.Validate(indexed)

		if vErr := s.validateExpression(ctx, entity); vErr != nil {
			addIndexedValidationError(indexed, vErr)
		}

		validateImportTestCases(indexed, payload.TestCases)

		entities = append(entities, entity)
	}

	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return entities, nil
}

// validateImportTestCases mirrors the domain rules for a scenario before any
// template exists to hang one on, so a broken export is refused with indexed
// field errors instead of failing mid-transaction.
func validateImportTestCases(indexed *errortypes.MultiError, testCases []*TestCaseInput) {
	if len(testCases) > importMaxTestCases {
		indexed.Add(
			"testCases",
			errortypes.ErrInvalid,
			fmt.Sprintf(
				"Cannot import more than %d test scenarios per template",
				importMaxTestCases,
			),
		)
		return
	}

	for index, testCase := range testCases {
		if testCase == nil {
			continue
		}

		caseErrs := indexed.WithIndex("testCases", index)
		switch {
		case testCase.Name == "":
			caseErrs.Add("name", errortypes.ErrRequired, "Name is required")
		case len(testCase.Name) > maxImportTestCaseNameLength:
			caseErrs.Add(
				"name",
				errortypes.ErrInvalid,
				"Name cannot be longer than 100 characters",
			)
		}

		if testCase.ExpectedAmount.IsNegative() {
			caseErrs.Add(
				"expectedAmount",
				errortypes.ErrInvalid,
				"Expected amount cannot be negative",
			)
		}

		if testCase.Tolerance.IsNegative() {
			caseErrs.Add("tolerance", errortypes.ErrInvalid, "Tolerance cannot be negative")
		}
	}
}

func (s *Service) createImportedTestCases(
	ctx context.Context,
	template *formulatemplate.FormulaTemplate,
	tenantInfo pagination.TenantInfo,
	testCases []*TestCaseInput,
) error {
	for _, testCase := range testCases {
		if testCase == nil {
			continue
		}

		entity := buildTestCaseEntity(template.ID, tenantInfo, testCase)
		if _, err := s.testCaseRepo.Create(ctx, entity); err != nil {
			return err
		}
	}

	return nil
}

func addIndexedValidationError(indexed *errortypes.MultiError, err error) {
	switch typed := err.(type) {
	case *errortypes.MultiError:
		for _, fieldErr := range typed.Errors {
			indexed.Add(fieldErr.Field, fieldErr.Code, fieldErr.Message)
		}
	case *errortypes.Error:
		indexed.Add(typed.Field, typed.Code, typed.Message)
	default:
		indexed.Add("expression", errortypes.ErrInvalid, err.Error())
	}
}

// importedPrecision reads the precision an export carried. Exports before 1.3
// carried no policy at all, and an export that names a mode without a
// precision meant the default; NormalizeRounding settles the fully absent
// case when the entity is validated.
func importedPrecision(payload *ImportTemplatePayload) int32 {
	if payload.RoundingPrecision != nil {
		return *payload.RoundingPrecision
	}
	if payload.RoundingMode != "" {
		return formulatypes.DefaultRoundingPrecision
	}
	return 0
}
