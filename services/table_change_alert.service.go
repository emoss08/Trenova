package services

import (
	"context"

	"github.com/emoss08/trenova/ent/tablechangealert"
	"github.com/emoss08/trenova/tools/kafka"

	"github.com/emoss08/trenova/database"
	"github.com/emoss08/trenova/ent"
	"github.com/emoss08/trenova/ent/organization"
	"github.com/google/uuid"
)

type TableChangeAlertOps struct {
	ctx    context.Context
	client *ent.Client
}

// NewTableChangeAlertOps creates a new table change alert service.
func NewTableChangeAlertOps(ctx context.Context) *TableChangeAlertOps {
	return &TableChangeAlertOps{
		ctx:    ctx,
		client: database.GetClient(),
	}
}

// GetTableChangeAlerts gets the table change alert for an organization.
func (r *TableChangeAlertOps) GetTableChangeAlerts(limit, offset int, orgID, buID uuid.UUID) ([]*ent.TableChangeAlert, int, error) {
	tableChangeAlertCount, countErr := r.client.TableChangeAlert.Query().Where(
		tablechangealert.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
	).Count(r.ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	tableChangeAlerts, err := r.client.TableChangeAlert.Query().
		Limit(limit).
		Offset(offset).
		Where(
			tablechangealert.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
		).All(r.ctx)
	if err != nil {
		return nil, 0, err
	}

	return tableChangeAlerts, tableChangeAlertCount, nil
}

// CreateTableChangeAlert creates a new table change alert.
func (r *TableChangeAlertOps) CreateTableChangeAlert(newTableChangeAlert ent.TableChangeAlert) (*ent.TableChangeAlert, error) {
	tableChangeAlert, err := r.client.TableChangeAlert.Create().
		SetOrganizationID(newTableChangeAlert.OrganizationID).
		SetBusinessUnitID(newTableChangeAlert.BusinessUnitID).
		SetStatus(newTableChangeAlert.Status).
		SetName(newTableChangeAlert.Name).
		SetDatabaseAction(newTableChangeAlert.DatabaseAction).
		SetSource(newTableChangeAlert.Source).
		SetTableName(newTableChangeAlert.TableName).
		SetTopicName(newTableChangeAlert.TopicName).
		SetDescription(newTableChangeAlert.Description).
		SetCustomSubject(newTableChangeAlert.CustomSubject).
		SetFunctionName(newTableChangeAlert.FunctionName).
		SetTriggerName(newTableChangeAlert.TriggerName).
		SetListenerName(newTableChangeAlert.ListenerName).
		SetEmailRecipients(newTableChangeAlert.EmailRecipients).
		SetNillableEffectiveDate(newTableChangeAlert.EffectiveDate).
		SetNillableExpirationDate(newTableChangeAlert.ExpirationDate).
		Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return tableChangeAlert, nil
}

// UpdateTableChangeAlert updates a table change alert.
func (r *TableChangeAlertOps) UpdateTableChangeAlert(tableChangeAlert ent.TableChangeAlert) (*ent.TableChangeAlert, error) {
	// Start building the update operation
	updateOp := r.client.TableChangeAlert.UpdateOneID(tableChangeAlert.ID).
		SetStatus(tableChangeAlert.Status).
		SetName(tableChangeAlert.Name).
		SetDatabaseAction(tableChangeAlert.DatabaseAction).
		SetSource(tableChangeAlert.Source).
		SetTableName(tableChangeAlert.TableName).
		SetTopicName(tableChangeAlert.TopicName).
		SetDescription(tableChangeAlert.Description).
		SetCustomSubject(tableChangeAlert.CustomSubject).
		SetFunctionName(tableChangeAlert.FunctionName).
		SetTriggerName(tableChangeAlert.TriggerName).
		SetListenerName(tableChangeAlert.ListenerName).
		SetEmailRecipients(tableChangeAlert.EmailRecipients).
		SetNillableEffectiveDate(tableChangeAlert.EffectiveDate).
		SetNillableExpirationDate(tableChangeAlert.ExpirationDate)

	// Execute the update operation
	updateTableChangeAlert, err := updateOp.Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return updateTableChangeAlert, nil
}

type TableName struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

func (r *TableChangeAlertOps) GetTableNames() ([]TableName, int, error) {
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

	tx, err := r.client.Tx(r.ctx)
	if err != nil {
		return nil, 0, err
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit() // if Commit returns error update err with commit err
		}
	}()

	query := "SELECT table_name FROM information_schema.tables WHERE table_schema = 'public' AND table_type = 'BASE TABLE'"
	rows, err := tx.QueryContext(r.ctx, query)
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
