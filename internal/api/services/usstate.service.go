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

	"github.com/emoss08/trenova/config"
	"github.com/emoss08/trenova/internal/server"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/uptrace/bun"
)

type USStateService struct {
	db     *bun.DB
	logger *config.ServerLogger
}

func NewUSStateService(s *server.Server) *USStateService {
	return &USStateService{
		db:     s.DB,
		logger: s.Logger,
	}
}

func (s USStateService) GetUSStates(ctx context.Context) ([]*models.UsState, int, error) {
	var states []*models.UsState
	count, err := s.db.NewSelect().
		Model(&states).
		Order("us.created_at DESC").
		ScanAndCount(ctx)
	if err != nil {
		return nil, 0, err
	}

	return states, count, nil
}
