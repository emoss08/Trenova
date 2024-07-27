// COPYRIGHT(c) 2024 Trenova
//
// This file is part of Trenova.
//
// The Trenova software is licensed under the Business Source License 1.1. You are granted the right
// to copy, modify, and redistribute the software, but only for non-production use or with a total
// of less than three server instances. Starting from the Change Date (November 16, 2026), the
// software will be made available under version 2 or later of the GNU General Public License.
// If you use the software in violation of this license, your rights under the license will be
// terminated automatically. The software is provided "as is," and the Licensor disclaims all
// warranties and conditions. If you use this license's text or the "Business Source License" name
// and trademark, you must comply with the Licensor's covenants, which include specifying the
// Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
// Grant, and not modifying the license in any other way.

package services

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/pkg/models/property"

	"github.com/rs/zerolog"

	"github.com/emoss08/trenova/config"
	"github.com/emoss08/trenova/internal/server"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type AuditLogService struct {
	db     *bun.DB
	logger *config.ServerLogger
}

func NewAuditLogService(s *server.Server) *AuditLogService {
	return &AuditLogService{
		db:     s.DB,
		logger: s.Logger,
	}
}

type AuditLogQueryFilter struct {
	OrganizationID uuid.UUID
	BusinessUnitID uuid.UUID
	UserID         uuid.UUID
	Limit          int
	Offset         int
	TableName      string
	EntityID       string
	Action         property.AuditLogAction
	Status         property.LogStatus
}

func (s AuditLogService) filterQuery(q *bun.SelectQuery, f *AuditLogQueryFilter) *bun.SelectQuery {
	q = q.Where("al.organization_id = ?", f.OrganizationID).
		Where("al.business_unit_id = ?", f.BusinessUnitID)

	if f.TableName != "" {
		q = q.Where("al.table_name = ?", f.TableName)
	}

	if f.UserID != uuid.Nil {
		q = q.Where("al.user_id = ?", f.UserID)
	}

	if f.EntityID != "" {
		q = q.Where("al.entity_id = ?", f.EntityID)
	}

	if f.Action != "" {
		q = q.Where("al.action = ?", f.Action)
	}

	if f.Status != "" {
		q = q.Where("al.status = ?", f.Status)
	}

	return q.Limit(f.Limit).Offset(f.Offset)
}

func (s AuditLogService) GetAll(ctx context.Context, filter *AuditLogQueryFilter) ([]*models.AuditLog, int, error) {
	var entities []*models.AuditLog

	q := s.db.NewSelect().Model(&entities).
		Relation("User").Order("al.timestamp DESC")

	q = s.filterQuery(q, filter)

	count, err := q.ScanAndCount(ctx)
	if err != nil {
		s.logger.Error().Dict("information", zerolog.
			Dict().
			Str("orgID", filter.OrganizationID.String()).
			Str("buID", filter.BusinessUnitID.String()),
		).
			Err(err).
			Msg("Failed to fetch Audit logs")

		return nil, 0, fmt.Errorf("failed to fetch audit log: %w", err)
	}

	return entities, count, nil
}
