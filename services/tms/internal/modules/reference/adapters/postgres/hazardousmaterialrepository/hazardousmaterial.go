package hazardousmaterialrepository

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/reference/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/modules/reference/domain/hazardousmaterial/hazardousmaterialdb"
	"github.com/emoss08/trenova/internal/modules/reference/ports/hazardousmaterialports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var hazardousMaterialErrors = dberror.Mapping{Entity: "HazardousMaterial"}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) hazardousmaterialports.HazardousMaterialRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.hazardousmaterial-repository"),
	}
}

func (r *repository) List(
	ctx context.Context,
	req *hazardousmaterialports.ListHazardousMaterialsRequest,
) (*pagination.ListResult[*hazardousmaterial.HazardousMaterial], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*hazardousmaterial.HazardousMaterial], error) {
			hm := hazardousmaterialdb.HazardousMaterials
			result, err := listquery.List(ctx, r.db.Keel(ctx), hm.Query(),
				&listquery.Request[hazardousmaterial.HazardousMaterial]{
					Filter: req.Filter,
					Then:   []keel.Order{hm.CreatedAt.Desc()},
				})
			if err != nil {
				r.l.Error("failed to list hazardous materials", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *hazardousmaterialports.ListHazardousMaterialConnectionRequest,
) (*pagination.CursorListResult[*hazardousmaterial.HazardousMaterial], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*hazardousmaterial.HazardousMaterial], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				hazardousmaterialdb.HazardousMaterials.Query(),
				&listquery.Request[hazardousmaterial.HazardousMaterial]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.HazardousMaterialColumns,
				},
			)
			if err != nil {
				r.l.Error("failed to list hazardous materials", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) Create(
	ctx context.Context,
	entity *hazardousmaterial.HazardousMaterial,
) (*hazardousmaterial.HazardousMaterial, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*hazardousmaterial.HazardousMaterial, error) {
			if entity.Code == "" {
				code, err := r.generateCode(ctx, entity)
				if err != nil {
					r.l.Error("failed to generate code", zap.Error(err))
					return nil, fmt.Errorf("failed to generate code: %w", err)
				}
				entity.Code = code
			}

			if err := hazardousmaterialdb.HazardousMaterials.Create(
				ctx,
				r.db.Keel(ctx),
				entity,
			); err != nil {
				r.l.Error(
					"failed to create hazardous material",
					zap.String("name", entity.Name),
					zap.Error(err),
				)
				return nil, hazardousMaterialErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) Update(
	ctx context.Context,
	entity *hazardousmaterial.HazardousMaterial,
) (*hazardousmaterial.HazardousMaterial, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*hazardousmaterial.HazardousMaterial, error) {
			if err := hazardousmaterialdb.HazardousMaterials.Save(
				ctx,
				r.db.Keel(ctx),
				entity,
			); err != nil {
				r.l.Error(
					"failed to update hazardous material",
					zap.String("id", entity.ID.String()),
					zap.Error(err),
				)
				return nil, hazardousMaterialErrors.Map(err, entity.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) GetByID(
	ctx context.Context,
	req hazardousmaterialports.GetHazardousMaterialByIDRequest,
) (*hazardousmaterial.HazardousMaterial, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*hazardousmaterial.HazardousMaterial, error) {
			entity, err := hazardousmaterialdb.HazardousMaterials.Get(ctx, r.db.Keel(ctx), req.ID)
			if err != nil {
				return nil, hazardousMaterialErrors.Map(err, req.ID.String())
			}

			return entity, nil
		},
	)
}

func (r *repository) GetByIDs(
	ctx context.Context,
	req hazardousmaterialports.GetHazardousMaterialsByIDsRequest,
) ([]*hazardousmaterial.HazardousMaterial, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*hazardousmaterial.HazardousMaterial, error) {
			if len(req.HazardousMaterialIDs) == 0 {
				return []*hazardousmaterial.HazardousMaterial{}, nil
			}

			hm := hazardousmaterialdb.HazardousMaterials
			entities, err := hm.Where(hm.ID.In(req.HazardousMaterialIDs...)).
				Limit(len(req.HazardousMaterialIDs)).
				List(ctx, r.db.Keel(ctx))
			if err != nil {
				r.l.Error("failed to get hazardous materials", zap.Error(err))
				return nil, err
			}

			return entities, nil
		},
	)
}

func (r *repository) BulkUpdateStatus(
	ctx context.Context,
	req *hazardousmaterialports.BulkUpdateHazardousMaterialStatusRequest,
) ([]*hazardousmaterial.HazardousMaterial, error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) ([]*hazardousmaterial.HazardousMaterial, error) {
			hm := hazardousmaterialdb.HazardousMaterials
			entities, err := hm.Where(hm.ID.In(req.HazardousMaterialIDs...)).
				UpdateReturning(ctx, r.db.Keel(ctx), hm.Status.To(req.Status))
			if err != nil {
				r.l.Error("failed to bulk update hazardous material status", zap.Error(err))
				return nil, err
			}

			err = dberror.CheckBulkAffected(
				int64(len(entities)),
				"HazardousMaterial",
				req.HazardousMaterialIDs,
			)
			if err != nil {
				return nil, err
			}

			return entities, nil
		},
	)
}

func (r *repository) SelectOptions(
	ctx context.Context,
	req *hazardousmaterialports.HazardousMaterialSelectOptionsRequest,
) (*pagination.ListResult[*hazardousmaterial.HazardousMaterial], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*hazardousmaterial.HazardousMaterial], error) {
			hm := hazardousmaterialdb.HazardousMaterials
			q := hm.Pick(hm.ID, hm.Name, hm.Class, hm.PackingGroup).
				Where(hm.Status.Eq(domaintypes.StatusActive))

			return listquery.SelectOptions(
				ctx,
				r.db.Keel(ctx),
				q,
				req.SelectQueryRequest,
				func(s string) keel.Pred {
					return keel.Or(hm.Name.ContainsFold(s), hm.Description.ContainsFold(s))
				},
			)
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

func (r *repository) codeExists(ctx context.Context, code string) (bool, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (bool, error) {
		hm := hazardousmaterialdb.HazardousMaterials
		return hm.Where(hm.Code.Lower().Eq(strings.ToLower(code))).Exists(ctx, r.db.Keel(ctx))
	})
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
		exists, err := r.codeExists(ctx, code)
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
