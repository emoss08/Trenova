package services

import (
	"context"

	"github.com/emoss08/trenova/internal/api/services/types"
	kfk "github.com/emoss08/trenova/internal/util/kafka"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/emoss08/trenova/internal/ent/tablechangealert"
	"github.com/emoss08/trenova/internal/util"
	"github.com/rs/zerolog"

	"github.com/google/uuid"
)

// TableChangeAlertService is the service for table change alert.
type TableChangeAlertService struct {
	Client *ent.Client
	Logger *zerolog.Logger
	Kafka  *kfk.Client
}

// NewTableChangeAlertService creates a new table change alert service.
func NewTableChangeAlertService(s *api.Server) *TableChangeAlertService {
	return &TableChangeAlertService{
		Client: s.Client,
		Logger: s.Logger,
		Kafka:  s.Kafka,
	}
}

// GetTableChangeAlerts gets the table change alert for an organization.
func (r *TableChangeAlertService) GetTableChangeAlerts(
	ctx context.Context, limit, offset int, orgID, buID uuid.UUID,
) ([]*ent.TableChangeAlert, int, error) {
	entityCount, countErr := r.Client.TableChangeAlert.Query().Where(
		tablechangealert.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	entities, err := r.Client.TableChangeAlert.Query().
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
func (r *TableChangeAlertService) CreateTableChangeAlert(
	ctx context.Context, entity *ent.TableChangeAlert,
) (*ent.TableChangeAlert, error) {
	updatedEntity := new(ent.TableChangeAlert)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		updatedEntity, err = r.createTableChangeAlertEntity(ctx, tx, entity)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}

func (r *TableChangeAlertService) createTableChangeAlertEntity(
	ctx context.Context, tx *ent.Tx, entity *ent.TableChangeAlert,
) (*ent.TableChangeAlert, error) {
	createdEntity, err := tx.TableChangeAlert.Create().
		SetOrganizationID(entity.OrganizationID).
		SetBusinessUnitID(entity.BusinessUnitID).
		SetStatus(entity.Status).
		SetName(entity.Name).
		SetDatabaseAction(entity.DatabaseAction).
		SetTopicName(entity.TopicName).
		SetDescription(entity.Description).
		SetCustomSubject(entity.CustomSubject).
		SetFunctionName(entity.FunctionName).
		SetTriggerName(entity.TriggerName).
		SetListenerName(entity.ListenerName).
		SetEmailRecipients(entity.EmailRecipients).
		SetEffectiveDate(entity.EffectiveDate).
		SetExpirationDate(entity.ExpirationDate).
		Save(ctx)
	if err != nil {
		return nil, err
	}

	return createdEntity, nil
}

// UpdateTableChangeAlert updates a table change alert.
func (r *TableChangeAlertService) UpdateTableChangeAlert(
	ctx context.Context, entity *ent.TableChangeAlert,
) (*ent.TableChangeAlert, error) {
	updatedEntity := new(ent.TableChangeAlert)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		updatedEntity, err = r.updateTableChangeAlertEntity(ctx, tx, entity)
		return err
	})
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}

func (r *TableChangeAlertService) updateTableChangeAlertEntity(
	ctx context.Context, tx *ent.Tx, entity *ent.TableChangeAlert,
) (*ent.TableChangeAlert, error) {
	current, err := tx.TableChangeAlert.Get(ctx, entity.ID)
	if err != nil {
		return nil, err
	}

	// Check if the version matches.
	if current.Version != entity.Version {
		return nil, util.NewValidationError("This record has been updated by another user. Please refresh and try again",
			"syncError",
			"name")
	}

	// Start building the update operation
	updateOp := tx.TableChangeAlert.UpdateOneID(entity.ID).
		SetOrganizationID(entity.OrganizationID).
		SetStatus(entity.Status).
		SetName(entity.Name).
		SetDatabaseAction(entity.DatabaseAction).
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
	updatedEntity, err := updateOp.Save(ctx)
	if err != nil {
		return nil, err
	}

	return updatedEntity, nil
}

func (r *TableChangeAlertService) GetTableNames(ctx context.Context) ([]types.TableName, int, error) {
	tableNames := make([]types.TableName, 0)
	var count int
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

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		tableNames, count, err = r.getTableNames(ctx, tx, excludedTableNames)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, 0, err
	}

	return tableNames, count, nil
}

func (r *TableChangeAlertService) getTableNames(
	ctx context.Context, tx *ent.Tx, excludedTableNames map[string]bool,
) ([]types.TableName, int, error) {
	query := "SELECT table_name FROM information_schema.tables WHERE table_schema = 'public' AND table_type = 'BASE TABLE'"
	rows, err := tx.QueryContext(ctx, query)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var tableNames []types.TableName
	var tableCount int
	for rows.Next() {
		var tableName string
		if scanErr := rows.Scan(&tableName); scanErr != nil {
			return nil, 0, scanErr
		}

		if _, excluded := excludedTableNames[tableName]; !excluded {
			tableNames = append(tableNames, types.TableName{Value: tableName, Label: tableName})
			tableCount++
		}
	}

	if rowErr := rows.Err(); rowErr != nil {
		return nil, 0, rowErr
	}

	return tableNames, tableCount, nil
}

func (r *TableChangeAlertService) GetTopicNames() ([]types.TopicName, int, error) {
	topics, err := r.Kafka.GetTopics()
	if err != nil {
		return nil, 0, err
	}

	topicNames := make([]types.TopicName, 0, len(topics))
	for _, topic := range topics {
		topicNames = append(topicNames, types.TopicName{
			Value: topic,
			Label: topic,
		})
	}

	return topicNames, len(topicNames), nil
}
