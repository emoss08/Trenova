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
	"strings"

	"github.com/emoss08/trenova/internal/server"
	"github.com/emoss08/trenova/internal/types"
	"github.com/emoss08/trenova/pkg/kfk"
	"github.com/emoss08/trenova/pkg/models"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/uptrace/bun"
)

type TableChangeAlertService struct {
	db     *bun.DB
	logger *zerolog.Logger
	kafka  *kfk.KafkaClient
}

func NewTableChangeAlertService(s *server.Server) *TableChangeAlertService {
	return &TableChangeAlertService{
		db:     s.DB,
		logger: s.Logger,
		kafka:  s.Kafka,
	}
}

func (s TableChangeAlertService) GetTableChangeAlerts(ctx context.Context, limit, offset int, orgID, buID uuid.UUID) ([]*models.TableChangeAlert, int, error) {
	var tableChangeAlerts []*models.TableChangeAlert
	count, err := s.db.NewSelect().
		Model(&tableChangeAlerts).
		Where("tca.organization_id = ?", orgID).
		Where("tca.business_unit_id = ?", buID).
		Order("tca.created_at DESC").
		Limit(limit).
		Offset(offset).
		ScanAndCount(ctx)
	if err != nil {
		return nil, 0, err
	}

	return tableChangeAlerts, count, nil
}

func (s TableChangeAlertService) CreateTableChangeAlert(ctx context.Context, tca *models.TableChangeAlert) (*models.TableChangeAlert, error) {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewInsert().
			Model(tca).
			Returning("*").
			Exec(ctx); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return tca, nil
}

func (s TableChangeAlertService) UpdateTableChangeAlert(ctx context.Context, tca *models.TableChangeAlert) (*models.TableChangeAlert, error) {
	err := s.db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if err := tca.OptimisticUpdate(ctx, tx); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return tca, nil
}

func (s TableChangeAlertService) GetTopicNames() ([]types.TopicName, int, error) {
	topics, err := s.kafka.GetTopics()
	if err != nil {
		return nil, 0, err
	}

	excludedTopics := []string{
		"__",
		"schema",
		"docker",
		"organization",
		"business_units",
		"google_apis",
		"permissions",
		"user_roles",
		"bun",
		"pro_number_counters",
		"master_key_generations",
		"audit_logs",
		"users",
		"us_states",
	}

	topicNames := make([]types.TopicName, 0, len(topics))
	for _, topic := range topics {
		exclude := false
		for _, excludedTopic := range excludedTopics {
			if strings.Contains(topic, excludedTopic) {
				exclude = true
				break
			}
		}
		if !exclude {
			topicNames = append(topicNames, types.TopicName{
				Value: topic,
				Label: topic,
			})
		}
	}

	return topicNames, len(topicNames), nil
}
