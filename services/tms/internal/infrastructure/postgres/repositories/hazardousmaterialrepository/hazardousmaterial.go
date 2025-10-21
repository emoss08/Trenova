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
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils/querybuilder"
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

func NewRepository(p Params) repositories.HazardousMaterialRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.hazardousmaterial-repository"),
	}
}

func (r *repository) filterQuery(
	q *bun.SelectQuery,
	req *repositories.ListHazardousMaterialRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFilters(
		q,
		"hm",
		req.Filter,
		(*hazardousmaterial.HazardousMaterial)(nil),
	)

	return q.Limit(req.Filter.Limit).Offset(req.Filter.Offset)
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListHazardousMaterialRequest,
) (*pagination.ListResult[*hazardousmaterial.HazardousMaterial], error) {
	log := r.l.With(
		zap.String("operation", "List"),
		zap.String("orgId", req.Filter.TenantOpts.OrgID.String()),
		zap.String("buId", req.Filter.TenantOpts.BuID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entities := make([]*hazardousmaterial.HazardousMaterial, 0, req.Filter.Limit)

	total, err := db.NewSelect().Model(&entities).Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
		return r.filterQuery(sq, req)
	}).ScanAndCount(ctx)
	if err != nil {
		log.Error("failed to scan hazardous materials", zap.Error(err))
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
		zap.String("hmID", req.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(hazardousmaterial.HazardousMaterial)
	err = db.NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("hm.id = ?", req.ID).
				Where("hm.organization_id = ?", req.OrgID).
				Where("hm.business_unit_id = ?", req.BuID)
		}).Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "Hazardous Material")
	}

	return entity, nil
}

func (r *repository) Create(
	ctx context.Context,
	hm *hazardousmaterial.HazardousMaterial,
) (*hazardousmaterial.HazardousMaterial, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("hmID", hm.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	if hm.Code == "" {
		generatedCode, genErr := r.generateCode(ctx, hm)
		if genErr != nil {
			log.Error("failed to generate code", zap.Error(genErr))
			return nil, fmt.Errorf("failed to generate code: %w", genErr)
		}

		hm.Code = generatedCode
		log.Info("generated code for hazardous material", zap.String("code", generatedCode))
	}

	if _, err = db.NewInsert().Model(hm).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to insert hazardous material", zap.Error(err))
		return nil, err
	}

	return hm, nil
}

func (r *repository) Update(
	ctx context.Context,
	hm *hazardousmaterial.HazardousMaterial,
) (*hazardousmaterial.HazardousMaterial, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("hmID", hm.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	ov := hm.Version
	hm.Version++

	results, rErr := db.NewUpdate().
		Model(hm).
		WherePK().
		OmitZero().
		Where("hm.version = ?", ov).
		Returning("*").
		Exec(ctx)
	if rErr != nil {
		log.Error("failed to update hazardous material", zap.Error(rErr))
		return nil, rErr
	}

	roErr := dberror.CheckRowsAffected(results, "Hazardous Material", hm.ID.String())
	if roErr != nil {
		return nil, roErr
	}

	return hm, nil
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
	db, err := r.db.DB(ctx)
	if err != nil {
		return false, err
	}

	exists, err := db.NewSelect().
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

	// Extract class number using strings.Builder
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

	// Convert packing group to number
	packingGroupNum := ""
	switch hm.PackingGroup {
	case hazardousmaterial.PackingGroupI:
		packingGroupNum = "1"
	case hazardousmaterial.PackingGroupII:
		packingGroupNum = "2"
	case hazardousmaterial.PackingGroupIII:
		packingGroupNum = "3"
	}

	// Build base code using strings.Builder
	nameAbbrev := abbreviateName(hm.Name, 4)
	var baseCodeBuilder strings.Builder
	baseCodeBuilder.Grow(10) // Pre-allocate for max length
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

	// Check for uniqueness and add suffix if needed
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

		// Build code with suffix
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
