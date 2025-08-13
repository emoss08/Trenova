package database

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/shared/edi/internal/core/domain"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type MigrationParams struct {
	fx.In
	DB     *DB
	Logger *zap.Logger
}

func RunMigrations(lc fx.Lifecycle, params MigrationParams) error {
	models := []any{
		(*domain.EDIDocument)(nil),
		(*domain.EDITransaction)(nil),
		(*domain.EDIShipment)(nil),
		(*domain.EDIStop)(nil),
		(*domain.EDIAcknowledgment)(nil),
		(*domain.EDIPartnerProfile)(nil),
	}

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			params.Logger.Info("running database migrations")
			
			for _, model := range models {
				if _, err := params.DB.NewCreateTable().
					Model(model).
					IfNotExists().
					Exec(ctx); err != nil {
					return fmt.Errorf("failed to create table for %T: %w", model, err)
				}
			}

			// Add missing columns if they don't exist
			if err := addMissingColumns(ctx, params.DB); err != nil {
				params.Logger.Warn("failed to add missing columns (may already exist)", zap.Error(err))
			}

			if err := createIndexes(ctx, params.DB); err != nil {
				return fmt.Errorf("failed to create indexes: %w", err)
			}

			params.Logger.Info("database migrations completed successfully")
			return nil
		},
	})

	return nil
}

func addMissingColumns(ctx context.Context, db *DB) error {
	// Add missing columns to edi_partner_profiles if they don't exist
	columns := []struct {
		name     string
		dataType string
	}{
		{"description", "TEXT"},
		{"profile_data", "JSONB NOT NULL DEFAULT '{}'::jsonb"},
	}
	
	for _, col := range columns {
		_, err := db.ExecContext(ctx, fmt.Sprintf(`
			ALTER TABLE edi_partner_profiles 
			ADD COLUMN IF NOT EXISTS %s %s
		`, col.name, col.dataType))
		if err != nil {
			return fmt.Errorf("failed to add %s column: %w", col.name, err)
		}
	}
	
	return nil
}

func createIndexes(ctx context.Context, db *DB) error {
	indexes := []struct {
		table   string
		columns []string
		unique  bool
	}{
		{"edi_documents", []string{"partner_id", "control_number"}, true},
		{"edi_documents", []string{"status"}, false},
		{"edi_documents", []string{"created_at"}, false},
		{"edi_transactions", []string{"document_id"}, false},
		{"edi_transactions", []string{"reference_id"}, false},
		{"edi_transactions", []string{"status"}, false},
		{"edi_shipments", []string{"shipment_id"}, false},
		{"edi_shipments", []string{"carrier_scac"}, false},
		{"edi_shipments", []string{"pickup_date"}, false},
		{"edi_stops", []string{"shipment_id", "stop_number"}, false},
		{"edi_acknowledgments", []string{"document_id"}, false},
	}

	for _, idx := range indexes {
		indexName := fmt.Sprintf("idx_%s_%s", idx.table, idx.columns[0])
		
		query := db.NewCreateIndex().
			Table(idx.table).
			Index(indexName).
			Column(idx.columns...).
			IfNotExists()

		if idx.unique {
			query = query.Unique()
		}

		if _, err := query.Exec(ctx); err != nil {
			return fmt.Errorf("failed to create index %s: %w", indexName, err)
		}
	}

	return nil
}