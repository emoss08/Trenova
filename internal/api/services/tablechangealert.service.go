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
	kafka  *kfk.Client
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
