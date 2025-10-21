package m2msync

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Config struct {
	Table            string
	SourceField      string
	TargetField      string
	AdditionalFields map[string]any
}

type SyncerParams struct {
	fx.In

	Logger *zap.Logger
}

type Syncer struct {
	logger *zap.Logger
}

func NewSyncer(p SyncerParams) *Syncer {
	return &Syncer{
		logger: p.Logger.Named("m2m-sync"),
	}
}

func (s *Syncer) SyncIDs(
	ctx context.Context,
	tx bun.IDB,
	config Config,
	sourceID pulid.ID,
	targetIDs []pulid.ID,
) error {
	log := s.logger.With(
		zap.String("table", config.Table),
		zap.String("sourceID", sourceID.String()),
		zap.Int("targetCount", len(targetIDs)),
	)

	whereConditions := []string{config.SourceField + " = ?"}
	whereValues := []any{sourceID}

	for field, value := range config.AdditionalFields {
		whereConditions = append(whereConditions, field+" = ?")
		whereValues = append(whereValues, value)
	}

	deleteQuery := tx.NewDelete().
		TableExpr(config.Table)

	for i, condition := range whereConditions {
		deleteQuery = deleteQuery.Where(condition, whereValues[i])
	}

	if len(targetIDs) > 0 {
		deleteQuery = deleteQuery.Where(config.TargetField+" NOT IN (?)", bun.In(targetIDs))
	}

	if _, err := deleteQuery.Exec(ctx); err != nil {
		log.Error("failed to delete removed relationships", zap.Error(err))
		return err
	}

	if len(targetIDs) > 0 {
		columns := []string{config.SourceField, config.TargetField}
		for field := range config.AdditionalFields {
			columns = append(columns, field)
		}

		var allValues []any
		valueStrings := make([]string, 0, len(targetIDs))

		for i, targetID := range targetIDs {
			placeholders := make([]string, len(columns))
			for j := range placeholders {
				placeholders[j] = "?"
			}
			valueStrings = append(valueStrings, "("+strings.Join(placeholders, ", ")+")")

			allValues = append(allValues, sourceID, targetID)
			for _, col := range columns[2:] { //! @NOTE: Skip first two columns already added
				allValues = append(allValues, config.AdditionalFields[col])
			}
			_ = i //! @NOTE: Avoid unused variable warning
		}

		query := fmt.Sprintf(
			"INSERT INTO %s (%s) VALUES %s ON CONFLICT DO NOTHING",
			config.Table,
			strings.Join(columns, ", "),
			strings.Join(valueStrings, ", "),
		)

		if _, err := tx.ExecContext(ctx, query, allValues...); err != nil {
			log.Error("failed to insert relationships", zap.Error(err))
			return err
		}
	}

	log.Debug("successfully synced relationships")
	return nil
}

func (s *Syncer) SyncEntities(
	ctx context.Context,
	tx bun.IDB,
	config Config,
	sourceID pulid.ID,
	entities any,
) error {
	targetIDs, err := s.extractIDs(entities)
	if err != nil {
		return err
	}

	return s.SyncIDs(ctx, tx, config, sourceID, targetIDs)
}

func (s *Syncer) extractIDs(entities any) ([]pulid.ID, error) {
	v := reflect.ValueOf(entities)
	if v.Kind() != reflect.Slice {
		return nil, ErrInvalidInput
	}

	ids := make([]pulid.ID, 0, v.Len())
	for i := 0; i < v.Len(); i++ {
		elem := v.Index(i)
		if elem.Kind() == reflect.Pointer {
			elem = elem.Elem()
		}

		idField := elem.FieldByName("ID")
		if !idField.IsValid() {
			return nil, ErrNoIDField
		}

		if id, ok := idField.Interface().(pulid.ID); ok {
			ids = append(ids, id)
		} else {
			return nil, ErrInvalidIDType
		}
	}

	return ids, nil
}
