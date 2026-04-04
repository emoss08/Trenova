package hazardousmaterialrepository

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/emoss08/trenova/internal/core/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.HazardousMaterialRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.hazardousmaterial-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListHazardousMaterialsRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		"hm",
		req.Filter,
		(*hazardousmaterial.HazardousMaterial)(nil),
	)

	return q.Limit(req.Filter.Pagination.SafeLimit()).Offset(req.Filter.Pagination.SafeOffset())
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListHazardousMaterialsRequest,
) (*pagination.ListResult[*hazardousmaterial.HazardousMaterial], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.Any("request", req),
	)

	entities := make([]*hazardousmaterial.HazardousMaterial, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DB().
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return r.filterQuery(sq, req)
		}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan and count hazardous materials", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*hazardousmaterial.HazardousMaterial]{
		Items: entities,
		Total: total,
	}, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetHazardousMaterialByIDRequest,
) (*hazardousmaterial.HazardousMaterial, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("id", req.ID.String()),
	)

	entity := new(hazardousmaterial.HazardousMaterial)
	err := r.db.DB().
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("hm.id = ?", req.ID).
				Where("hm.organization_id = ?", req.TenantInfo.OrgID).
				Where("hm.business_unit_id = ?", req.TenantInfo.BuID)
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get hazardous material", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "HazardousMaterial")
	}

	return entity, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *hazardousmaterial.HazardousMaterial,
) (*hazardousmaterial.HazardousMaterial, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("name", entity.Name),
	)

	if entity.Code == "" {
		generatedCode, genErr := r.generateCode(ctx, entity)
		if genErr != nil {
			log.Error("failed to generate code", zap.Error(genErr))
			return nil, fmt.Errorf("failed to generate code: %w", genErr)
		}

		entity.Code = generatedCode
	}

	if _, err := r.db.DB().NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to create hazardous material", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *hazardousmaterial.HazardousMaterial,
) (*hazardousmaterial.HazardousMaterial, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("id", entity.ID.String()),
	)

	ov := entity.Version
	entity.Version++

	results, err := r.db.DB().
		NewUpdate().
		Model(entity).
		WherePK().
		Where("version = ?", ov).
		OmitZero().
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to update hazardous material", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckRowsAffected(results, "HazardousMaterial", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) BulkUpdateStatus(
	ctx context.Context,
	req *repositories.BulkUpdateHazardousMaterialStatusRequest,
) ([]*hazardousmaterial.HazardousMaterial, error) {
	log := r.l.With(
		zap.String("operation", "BulkUpdateStatus"),
		zap.Any("request", req),
	)

	entities := make([]*hazardousmaterial.HazardousMaterial, 0, len(req.HazardousMaterialIDs))
	results, err := r.db.DB().
		NewUpdate().
		Model(&entities).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return uq.Where("hm.organization_id = ?", req.TenantInfo.OrgID).
				Where("hm.business_unit_id = ?", req.TenantInfo.BuID).
				Where("hm.id IN (?)", bun.In(req.HazardousMaterialIDs))
		}).
		Set("status = ?", req.Status).
		Returning("*").
		Exec(ctx)
	if err != nil {
		log.Error("failed to bulk update hazardous material status", zap.Error(err))
		return nil, err
	}

	if err = dberror.CheckBulkRowsAffected(results, "HazardousMaterial", req.HazardousMaterialIDs); err != nil {
		return nil, err
	}

	return entities, nil
}

func (r *repository) GetByIDs(
	ctx context.Context,
	req repositories.GetHazardousMaterialsByIDsRequest,
) ([]*hazardousmaterial.HazardousMaterial, error) {
	log := r.l.With(
		zap.String("operation", "GetByIDs"),
		zap.Any("request", req),
	)

	entities := make([]*hazardousmaterial.HazardousMaterial, 0, len(req.HazardousMaterialIDs))
	err := r.db.DB().
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("hm.organization_id = ?", req.TenantInfo.OrgID).
				Where("hm.business_unit_id = ?", req.TenantInfo.BuID).
				Where("hm.id IN (?)", bun.In(req.HazardousMaterialIDs))
		}).
		Scan(ctx)
	if err != nil {
		log.Error("failed to get hazardous materials", zap.Error(err))
		return nil, dberror.HandleNotFoundError(err, "HazardousMaterial")
	}

	return entities, nil
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *repositories.HazardousMaterialSelectOptionsRequest,
) (*pagination.ListResult[*hazardousmaterial.HazardousMaterial], error) {
	return dbhelper.SelectOptions[*hazardousmaterial.HazardousMaterial](
		ctx,
		r.db.DB(),
		req.SelectQueryRequest,
		&dbhelper.SelectOptionsConfig{
			Columns: []string{
				"id",
				"name",
				"class",
				"packing_group",
			},
			OrgColumn: "hm.organization_id",
			BuColumn:  "hm.business_unit_id",
			QueryModifier: func(q *bun.SelectQuery) *bun.SelectQuery {
				return q.Where("hm.status = ?", domaintypes.StatusActive)
			},
			EntityName: "HazardousMaterial",
			SearchColumns: []string{
				"hm.name",
				"hm.description",
			},
		},
	)
}

func abbreviateName(name string, maxLength int) string {
	if name == "" {
		return ""
	}

	name = strings.TrimSpace(name)
	name = regexp.MustCompile(`[^\w\s-]`).ReplaceAllString(name, "")
	name = strings.ToUpper(name)

	if len(name) <= maxLength {
		return name
	}

	words := strings.Fields(name)
	if len(words) > 1 {
		var acronym strings.Builder
		acronym.Grow(len(words))
		for _, word := range words {
			if word != "" {
				acronym.WriteByte(word[0])
			}
		}
		result := acronym.String()
		if len(result) <= maxLength && len(result) >= 2 {
			return result
		}
	}

	var consonants strings.Builder
	consonants.Grow(len(name))
	for _, r := range name {
		if unicode.IsLetter(r) && !strings.ContainsRune("AEIOU", r) {
			consonants.WriteRune(r)
		}
	}
	result := consonants.String()
	if len(result) >= maxLength {
		return result[:maxLength]
	}

	return name[:maxLength]
}

func (r *repository) codeExists(
	ctx context.Context,
	code string,
	orgID, buID pulid.ID,
) (bool, error) {
	exists, err := r.db.DB().NewSelect().
		Model((*hazardousmaterial.HazardousMaterial)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("code = ?", code).
				Where("organization_id = ?", orgID).
				Where("business_unit_id = ?", buID)
		}).
		Exists(ctx)

	return exists, err
}

func (r *repository) generateCode(
	ctx context.Context,
	hm *hazardousmaterial.HazardousMaterial,
) (string, error) {
	log := r.l.With(
		zap.String("operation", "generateCode"),
		zap.String("name", hm.Name),
	)

	classStr := string(hm.Class)
	var classNumber strings.Builder
	if strings.Contains(classStr, "And") {
		parts := strings.Split(strings.TrimPrefix(classStr, "HazardClass"), "And")
		classNumber.Grow(len(parts) * 2)
		for _, part := range parts {
			classNumber.WriteString(part)
		}
	} else {
		classNumber.WriteString(strings.TrimPrefix(classStr, "HazardClass"))
	}

	packingGroupNum := ""
	switch hm.PackingGroup {
	case hazardousmaterial.PackingGroupI:
		packingGroupNum = "1"
	case hazardousmaterial.PackingGroupII:
		packingGroupNum = "2"
	case hazardousmaterial.PackingGroupIII:
		packingGroupNum = "3"
	}

	nameAbbrev := abbreviateName(hm.Name, 4)
	var baseCodeBuilder strings.Builder
	baseCodeBuilder.Grow(10)
	baseCodeBuilder.WriteString(nameAbbrev)
	baseCodeBuilder.WriteString(classNumber.String())
	baseCodeBuilder.WriteString(packingGroupNum)

	if hm.UNNumber != "" {
		baseCodeBuilder.WriteString(hm.UNNumber)
	}

	baseCode := baseCodeBuilder.String()
	if len(baseCode) > 10 {
		baseCode = baseCode[:10]
	}

	code := baseCode
	const maxAttempts = 5
	for attempt := range maxAttempts {
		exists, err := r.codeExists(ctx, code, hm.OrganizationID, hm.BusinessUnitID)
		if err != nil {
			log.Error("failed to check code existence", zap.Error(err))
			return "", fmt.Errorf("failed to check code uniqueness: %w", err)
		}

		if !exists {
			return code, nil
		}

		suffix := strconv.Itoa(attempt + 1)
		maxBaseLen := 10 - len(suffix)

		var codeBuilder strings.Builder
		codeBuilder.Grow(10)
		if len(baseCode) > maxBaseLen {
			codeBuilder.WriteString(baseCode[:maxBaseLen])
		} else {
			codeBuilder.WriteString(baseCode)
		}
		codeBuilder.WriteString(suffix)
		code = codeBuilder.String()
	}

	log.Error("failed to generate unique code after max attempts",
		zap.String("baseCode", baseCode),
		zap.Int("maxAttempts", maxAttempts),
	)
	return "", fmt.Errorf("failed to generate unique code after %d attempts", maxAttempts)
}
