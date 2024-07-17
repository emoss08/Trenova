// Copyright (c) 2024 Trenova Technologies, LLC
//
// Licensed under the Business Source License 1.1 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://trenova.app/pricing/
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//
// Key Terms:
// - Non-production use only
// - Change Date: 2026-11-16
// - Change License: GNU General Public License v2 or later
//
// For full license text, see the LICENSE file in the root directory.

package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

// DebeziumPayload represents the structure of a Debezium message payload.
// It contains information about the state of a database row before and after a change,
// the source of the change, the type of operation, and the timestamp of the change.
type DebeziumPayload struct {
	Before      map[string]interface{} `json:"before"`
	After       map[string]interface{} `json:"after"`
	Source      Source                 `json:"source"`
	Op          string                 `json:"op"`
	TsMs        int64                  `json:"ts_ms"`
	Transaction interface{}            `json:"transaction"`
}

// Source represents the source metadata for a Debezium message.
// It includes information about the database, table, and transaction involved in the change.
type Source struct {
	Version   string `json:"version"`
	Connector string `json:"connector"`
	Name      string `json:"name"`
	TsMs      int64  `json:"ts_ms"`
	Snapshot  string `json:"snapshot"`
	DB        string `json:"db"`
	Sequence  string `json:"sequence"`
	Schema    string `json:"schema"`
	Table     string `json:"table"`
	TxId      int64  `json:"txId"`
	Lsn       int64  `json:"lsn"`
	Xmin      int64  `json:"xmin"`
}

// SubscriptionDatabaseAction represents the type of database action for a subscription.
type SubscriptionDatabaseAction string

const (
	Insert = SubscriptionDatabaseAction("Insert")
	Update = SubscriptionDatabaseAction("Update")
	Delete = SubscriptionDatabaseAction("Delete")
	All    = SubscriptionDatabaseAction("All")
)

// SubscriptionStatus represents the status of a subscription.
type SubscriptionStatus string

const (
	Active   = SubscriptionStatus("A")
	Inactive = SubscriptionStatus("I")
)

// SubscriptionDeliveryMethod represents the delivery method for a subscription.
type SubscriptionDeliveryMethod string

const (
	Email = SubscriptionDeliveryMethod("Email")
	Local = SubscriptionDeliveryMethod("Local")
	API   = SubscriptionDeliveryMethod("Api")
	SMS   = SubscriptionDeliveryMethod("Sms")
)

// Subscription represents a subscription to database changes for a specific organization.
// It contains details about the subscription's status, associated organization, topic, and delivery method.
type Subscription struct {
	ID              string                     `json:"id"`
	Status          SubscriptionStatus         `json:"status"`
	BusinessUnitID  string                     `json:"businessUnitID"`
	OrganizationID  string                     `json:"organizationID"`
	TopicName       string                     `json:"topicName"`
	DatabaseAction  SubscriptionDatabaseAction `json:"databaseAction"`
	DeliveryMethod  SubscriptionDeliveryMethod `json:"deliveryMethod"`
	CustomSubject   string                     `json:"customSubject"`
	EmailRecipients string                     `json:"emailRecipients"`
	EffectiveDate   *pgtype.Date               `json:"effectiveDate"`
	ExpirationDate  *pgtype.Date               `json:"expirationDate"`
}

// SubscriptionService provides methods to manage subscriptions for database change alerts.
// It interacts with the database, cache, and logger to retrieve, cache, and validate subscriptions.
type SubscriptionService struct {
	db     *sql.DB
	logger *zerolog.Logger
	cache  *redis.Client
}

// NewSubscriptionService creates a new SubscriptionService with the provided database connection,
// logger, and cache client.
//
// Parameters:
//
//	db - database connection
//	logger - logger instance for logging messages
//	cache - Redis client for caching subscription data
//
// Returns:
//
//	*SubscriptionService - a new SubscriptionService instance
func NewSubscriptionService(db *sql.DB, logger *zerolog.Logger, cache *redis.Client) *SubscriptionService {
	return &SubscriptionService{db: db, logger: logger, cache: cache}
}

// GetActiveSubscriptions retrieves active subscriptions from the cache or database.
// If the cache is empty or invalid, it fetches subscriptions from the database and caches them.
//
// Parameters:
//
//	ctx - context for managing the lifecycle of the operation
//
// Returns:
//
//	[]Subscription - a slice of active subscriptions
//	error - an error if retrieval fails
func (s *SubscriptionService) GetActiveSubscriptions(ctx context.Context) ([]Subscription, error) {
	cacheKey := "active_subscriptions"
	cachedTopics, err := s.cache.Get(ctx, cacheKey).Result()
	if err != nil && err != redis.Nil {
		s.logger.Error().Err(err).Msg("failed to get cache")
		return nil, fmt.Errorf("failed to get cache: %w", err)
	}

	if cachedTopics != "" {
		var subscriptions []Subscription
		if err := sonic.Unmarshal([]byte(cachedTopics), &subscriptions); err != nil {
			s.logger.Error().Err(err).Msg("failed to unmarshal topics")
			return nil, fmt.Errorf("failed to unmarshal topics: %w", err)
		}
		return subscriptions, nil
	}

	subscriptions, err := s.fetchSubscriptionsFromDB(ctx)
	if err != nil {
		return nil, err
	}

	s.cacheSubscriptions(ctx, subscriptions)

	return subscriptions, nil
}

// fetchSubscriptionsFromDB retrieves active subscriptions directly from the database.
//
// Parameters:
//
//	ctx - context for managing the lifecycle of the operation
//
// Returns:
//
//	[]Subscription - a slice of active subscriptions fetched from the database
//	error - an error if retrieval fails
func (s *SubscriptionService) fetchSubscriptionsFromDB(ctx context.Context) ([]Subscription, error) {
	query := `SELECT
	id,
    status,
    topic_name,
    business_unit_id,
    custom_subject,
    organization_id,
    database_action,
	delivery_method,
    effective_date,
    expiration_date,
    email_recipients
FROM
    table_change_alerts`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to execute query")
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	var subscriptions []Subscription

	for rows.Next() {
		var subscription Subscription
		if err := rows.Scan(
			&subscription.ID,
			&subscription.Status,
			&subscription.TopicName,
			&subscription.BusinessUnitID,
			&subscription.CustomSubject,
			&subscription.OrganizationID,
			&subscription.DatabaseAction,
			&subscription.DeliveryMethod,
			&subscription.EffectiveDate,
			&subscription.ExpirationDate,
			&subscription.EmailRecipients,
		); err != nil {
			s.logger.Error().Err(err).Msg("failed to scan row")
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		subscriptions = append(subscriptions, subscription)
	}

	if err = rows.Err(); err != nil {
		s.logger.Error().Err(err).Msg("error iterating rows")
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return subscriptions, nil
}

// cacheSubscriptions caches the active subscriptions in Redis.
//
// Parameters:
//
//	ctx - context for managing the lifecycle of the operation
//	subscriptions - a slice of subscriptions to be cached
func (s *SubscriptionService) cacheSubscriptions(ctx context.Context, subscriptions []Subscription) {
	cacheKey := "active_subscriptions"
	cacheData, err := sonic.Marshal(subscriptions)
	if err != nil {
		s.logger.Error().Err(err).Msg("failed to marshal topics")
		return
	}

	if err := s.cache.Set(ctx, cacheKey, cacheData, 0).Err(); err != nil {
		s.logger.Error().Err(err).Msg("failed to set cache")
	}
}

// InvalidateCache invalidates the cache of active subscriptions.
//
// Parameters:
//
//	ctx - context for managing the lifecycle of the operation
func (s *SubscriptionService) InvalidateCache(ctx context.Context) {
	cacheKey := "active_subscriptions"
	if err := s.cache.Del(ctx, cacheKey).Err(); err != nil {
		s.logger.Error().Err(err).Msg("failed to invalidate cache")
	}
}

// MapActionToDebeziumType maps a SubscriptionDatabaseAction to the corresponding Debezium operation type.
//
// Parameters:
//
//	action - the subscription database action
//
// Returns:
//
//	string - the corresponding Debezium operation type ("c" for create, "u" for update, "d" for delete, "all" for all operations)
func (s *SubscriptionService) MapActionToDebeziumType(action SubscriptionDatabaseAction) string {
	switch action {
	case Insert:
		return "c"
	case Update:
		return "u"
	case Delete:
		return "d"
	case All:
		return "all"
	default:
		return ""
	}
}

// ParseDebeziumPayload unmarshals a JSON-encoded Debezium payload into a DebeziumPayload struct.
//
// Parameters:
//
//	payload - the JSON-encoded Debezium payload
//
// Returns:
//
//	*DebeziumPayload - the unmarshaled DebeziumPayload struct
//	error - an error if unmarshalling fails
func ParseDebeziumPayload(payload []byte) (*DebeziumPayload, error) {
	var dPayload DebeziumPayload
	if err := sonic.Unmarshal(payload, &dPayload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	return &dPayload, nil
}

// MatchesSubscription checks if a Debezium payload matches the criteria of a subscription.
//
// Parameters:
//
//	subscription - the subscription to check against
//	payload - the Debezium payload to check
//
// Returns:
//
//	bool - true if the payload matches the subscription criteria, false otherwise
func (s *SubscriptionService) MatchesSubscription(subscription Subscription, payload DebeziumPayload) bool {
	if subscription.Status != Active {
		return false
	}

	now := pgtype.Date{Time: time.Now()}
	if subscription.EffectiveDate != nil && now.Time.Before(subscription.EffectiveDate.Time) {
		return false
	}
	if subscription.ExpirationDate != nil && now.Time.After(subscription.ExpirationDate.Time) {
		return false
	}

	action := s.MapActionToDebeziumType(subscription.DatabaseAction)
	if action != "" && action != "all" && action != payload.Op {
		return false
	}

	return true
}
