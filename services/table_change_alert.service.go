package services

import (
	"context"

	"github.com/emoss08/trenova/ent/tablechangealert"
	"github.com/emoss08/trenova/tools"
	"github.com/emoss08/trenova/tools/kafka"
	"github.com/emoss08/trenova/tools/logger"
	"github.com/rotisserie/eris"
	"github.com/sirupsen/logrus"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/organization"
	"github.com/google/uuid"
)

type TableChangeAlertOps struct {
	client *ent.Client
	logger *logrus.Logger
}

// NewTableChangeAlertOps creates a new table change alert service.
func NewTableChangeAlertOps() *TableChangeAlertOps {
	return &TableChangeAlertOps{
		client: database.GetClient(),
		logger: logger.GetLogger(),
	}
}

// GetTableChangeAlerts gets the table change alert for an organization.
func (r *TableChangeAlertOps) GetTableChangeAlerts(
	ctx context.Context, limit, offset int, orgID, buID uuid.UUID,
) ([]*ent.TableChangeAlert, int, error) {
	entityCount, countErr := r.client.TableChangeAlert.Query().Where(
		tablechangealert.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	entities, err := r.client.TableChangeAlert.Query().
		Limit(limit).
		Offset(offset).
		Where(
			tablechangealert.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return entities, entityCount, nil
}

// CreateTableChangeAlert creates a new table change alert.
func (r *TableChangeAlertOps) CreateTableChangeAlert(
	ctx context.Context, newEntity ent.TableChangeAlert,
) (*ent.TableChangeAlert, error) {
	// Begin a new transaction
	tx, err := r.client.Tx(ctx)
	if err != nil {
		wrappedErr := eris.Wrap(err, "failed to start transaction")
		r.logger.WithField("error", wrappedErr).Error("failed to start transaction")
		return nil, wrappedErr
	}

	// Ensure the transaction is either committed or rolled back
	defer func() {
		if v := recover(); v != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				wrappedErr := eris.Wrap(rollbackErr, "failed to rollback transaction")
				r.logger.WithField("error", wrappedErr).Error("failed to rollback transaction")
			}
			panic(v)
		}
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				wrappedErr := eris.Wrap(err, "failed to rollback transaction")
				r.logger.WithField("error", wrappedErr).Error("failed to rollback transaction")
			}
		} else {
			if commitErr := tx.Commit(); commitErr != nil {
				wrappedErr := eris.Wrap(err, "failed to commit transaction")
				r.logger.WithField("error", wrappedErr).Error("failed to commit transaction")
			}
		}
	}()

	createdEntity, err := tx.TableChangeAlert.Create().
		SetOrganizationID(newEntity.OrganizationID).
		SetBusinessUnitID(newEntity.BusinessUnitID).
		SetStatus(newEntity.Status).
		SetName(newEntity.Name).
		SetDatabaseAction(newEntity.DatabaseAction).
		SetSource(newEntity.Source).
		SetTableName(newEntity.TableName).
		SetTopicName(newEntity.TopicName).
		SetDescription(newEntity.Description).
		SetCustomSubject(newEntity.CustomSubject).
		SetFunctionName(newEntity.FunctionName).
		SetTriggerName(newEntity.TriggerName).
		SetListenerName(newEntity.ListenerName).
		SetEmailRecipients(newEntity.EmailRecipients).
		SetEffectiveDate(newEntity.EffectiveDate).
		SetExpirationDate(newEntity.ExpirationDate).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return createdEntity, nil
}

// UpdateTableChangeAlert updates a table change alert.
func (r *TableChangeAlertOps) UpdateTableChangeAlert(
	ctx context.Context, entity ent.TableChangeAlert,
) (*ent.TableChangeAlert, error) {
	// Begin a new transaction
	tx, err := r.client.Tx(ctx)
	if err != nil {
		wrappedErr := eris.Wrap(err, "failed to start transaction")
		r.logger.WithField("error", wrappedErr).Error("failed to start transaction")
		return nil, wrappedErr
	}

	// Ensure the transaction is either committed or rolled back
	defer func() {
		if v := recover(); v != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				wrappedErr := eris.Wrap(rollbackErr, "failed to rollback transaction")
				r.logger.WithField("error", wrappedErr).Error("failed to rollback transaction")
			}
			panic(v)
		}
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				wrappedErr := eris.Wrap(err, "failed to rollback transaction")
				r.logger.WithField("error", wrappedErr).Error("failed to rollback transaction")
			}
		} else {
			if commitErr := tx.Commit(); commitErr != nil {
				wrappedErr := eris.Wrap(err, "failed to commit transaction")
				r.logger.WithField("error", wrappedErr).Error("failed to commit transaction")
			}
		}
	}()

	current, err := tx.TableChangeAlert.Get(ctx, entity.ID) // Get the current entity.
	if err != nil {
		wrappedErr := eris.Wrap(err, "failed to retrieve requested entity")
		r.logger.WithField("error", wrappedErr).Error("failed to retrieve requested entity")
		return nil, wrappedErr
	}

	// Check if the version matches.
	if current.Version != entity.Version {
		return nil, tools.NewValidationError("This record has been updated by another user. Please refresh and try again",
			"syncError",
			"name")
	}

	// Start building the update operation
	updateOp := tx.TableChangeAlert.UpdateOneID(entity.ID).
		SetStatus(entity.Status).
		SetName(entity.Name).
		SetDatabaseAction(entity.DatabaseAction).
		SetSource(entity.Source).
		SetTableName(entity.TableName).
		SetTopicName(entity.TopicName).
		SetDescription(entity.Description).
		SetCustomSubject(entity.CustomSubject).
		SetFunctionName(entity.FunctionName).
		SetTriggerName(entity.TriggerName).
		SetListenerName(entity.ListenerName).
		SetEmailRecipients(entity.EmailRecipients).
		SetEffectiveDate(entity.EffectiveDate).
		SetExpirationDate(entity.ExpirationDate).
		SetVersion(entity.Version + 1) // Increment the version

	// Execute the update operation
	updateTableChangeAlert, err := updateOp.Save(ctx)
	if err != nil {
		return nil, err
	}

	return updateTableChangeAlert, nil
}

type TableName struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

func (r *TableChangeAlertOps) GetTableNames(ctx context.Context) ([]TableName, int, error) {
	excludedTableNames := map[string]bool{
		"table_change_alerts":       true,
		"shipment_controls":         true,
		"billing_controls":          true,
		"sessions":                  true,
		"organizations":             true,
		"business_units":            true,
		"feasibility_tool_controls": true,
		"users":                     true,
		"general_ledger_accounts":   true,
		"user_favorites":            true,
		"us_states":                 true,
		"invoice_controls":          true,
		"email_controls":            true,
		"route_controls":            true,
		"accounting_controls":       true,
		"email_profiles":            true,
	}

	tx, err := r.client.Tx(ctx)
	if err != nil {
		return nil, 0, err
	}
	// Ensure the transaction is either committed or rolled back
	defer func() {
		if v := recover(); v != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				wrappedErr := eris.Wrap(rollbackErr, "failed to rollback transaction")
				r.logger.WithField("error", wrappedErr).Error("failed to rollback transaction")
			}
			panic(v)
		}
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				wrappedErr := eris.Wrap(err, "failed to rollback transaction")
				r.logger.WithField("error", wrappedErr).Error("failed to rollback transaction")
			}
		} else {
			if commitErr := tx.Commit(); commitErr != nil {
				wrappedErr := eris.Wrap(err, "failed to commit transaction")
				r.logger.WithField("error", wrappedErr).Error("failed to commit transaction")
			}
		}
	}()

	query := "SELECT table_name FROM information_schema.tables WHERE table_schema = 'public' AND table_type = 'BASE TABLE'"
	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tableNames []TableName
	var tableCount int
	for rows.Next() {
		var tableName string
		if scanErr := rows.Scan(&tableName); scanErr != nil {
			return nil, 0, scanErr
		}

		if _, excluded := excludedTableNames[tableName]; !excluded {
			tableNames = append(tableNames, TableName{Value: tableName, Label: tableName})
			tableCount++
		}
	}

	if rowErr := rows.Err(); rowErr != nil {
		return nil, 0, rowErr
	}

	return tableNames, tableCount, nil
}

type TopicName struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

func (r *TableChangeAlertOps) GetTopicNames() ([]TopicName, int, error) {
	topics, err := kafka.GetKafkaTopics()
	if err != nil {
		return nil, 0, err
	}

	topicNames := make([]TopicName, 0, len(topics))
	for _, topic := range topics {
		topicNames = append(topicNames, TopicName{
			Value: topic,
			Label: topic,
		})
	}

	return topicNames, len(topicNames), nil
}
