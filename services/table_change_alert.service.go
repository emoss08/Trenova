package services

import (
	"context"

	"github.com/emoss08/trenova/ent/tablechangealert"

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
		SetTopic(newTableChangeAlert.Topic).
		SetDescription(newTableChangeAlert.Description).
		SetCustomSubject(newTableChangeAlert.CustomSubject).
		SetFunctionName(newTableChangeAlert.FunctionName).
		SetTriggerName(newTableChangeAlert.TriggerName).
		SetListenerName(newTableChangeAlert.ListenerName).
		SetEmailRecipients(newTableChangeAlert.EmailRecipients).
		SetEffectiveDate(newTableChangeAlert.EffectiveDate).
		SetExpirationDate(newTableChangeAlert.ExpirationDate).
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
		SetTopic(tableChangeAlert.Topic).
		SetDescription(tableChangeAlert.Description).
		SetCustomSubject(tableChangeAlert.CustomSubject).
		SetFunctionName(tableChangeAlert.FunctionName).
		SetTriggerName(tableChangeAlert.TriggerName).
		SetListenerName(tableChangeAlert.ListenerName).
		SetEmailRecipients(tableChangeAlert.EmailRecipients).
		SetEffectiveDate(tableChangeAlert.EffectiveDate).
		SetExpirationDate(tableChangeAlert.ExpirationDate)

	// Execute the update operation
	updateTableChangeAlert, err := updateOp.Save(r.ctx)
	if err != nil {
		return nil, err
	}

	return updateTableChangeAlert, nil
}
