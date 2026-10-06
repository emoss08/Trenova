package tenantbootstrap

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/costingcontrol"
	"github.com/emoss08/trenova/internal/core/domain/dataentrycontrol"
	"github.com/emoss08/trenova/internal/core/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

func CreateControls(ctx context.Context, db bun.IDB, scope Scope) error {
	now := scope.now()

	accountingControl := &tenant.AccountingControl{
		ID:                              pulid.MustNew("ac_"),
		OrganizationID:                  scope.OrganizationID,
		BusinessUnitID:                  scope.BusinessUnitID,
		RequireManualJEApproval:         true,
		RequirePeriodCloseApproval:      true,
		NotifyOnReconciliationException: true,
		CreatedAt:                       now,
		UpdatedAt:                       now,
	}
	if _, err := db.NewInsert().Model(accountingControl).Exec(ctx); err != nil {
		return fmt.Errorf("create accounting control: %w", err)
	}
	if err := scope.record(ctx, "accounting_controls", accountingControl.ID); err != nil {
		return err
	}

	billingControl := &tenant.BillingControl{
		ID:                           pulid.MustNew("bc_"),
		OrganizationID:               scope.OrganizationID,
		BusinessUnitID:               scope.BusinessUnitID,
		BillingQueueTransferSchedule: tenant.TransferScheduleContinuous,
		ShowDueDateOnInvoice:         true,
		ShowBalanceDueOnInvoice:      true,
		NotifyOnBillingExceptions:    true,
		RequireRateOverrideReason:    true,
		CreatedAt:                    now,
		UpdatedAt:                    now,
	}
	if _, err := db.NewInsert().Model(billingControl).Exec(ctx); err != nil {
		return fmt.Errorf("create billing control: %w", err)
	}
	if err := scope.record(ctx, "billing_controls", billingControl.ID); err != nil {
		return err
	}

	invoiceAdjustmentControl := &tenant.InvoiceAdjustmentControl{
		ID:                                  pulid.MustNew("iac_"),
		OrganizationID:                      scope.OrganizationID,
		BusinessUnitID:                      scope.BusinessUnitID,
		StandardAdjustmentApprovalThreshold: decimal.NewFromFloat(0.01),
		WriteOffApprovalThreshold:           decimal.NewFromFloat(0.01),
		CreatedAt:                           now,
		UpdatedAt:                           now,
	}
	if _, err := db.NewInsert().Model(invoiceAdjustmentControl).Exec(ctx); err != nil {
		return fmt.Errorf("create invoice adjustment control: %w", err)
	}
	if err := scope.record(ctx, "invoice_adjustment_controls", invoiceAdjustmentControl.ID); err != nil {
		return err
	}

	dispatchControl := &dispatchcontrol.DispatchControl{
		ID:             pulid.MustNew("dc_"),
		OrganizationID: scope.OrganizationID,
		BusinessUnitID: scope.BusinessUnitID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if _, err := db.NewInsert().Model(dispatchControl).Exec(ctx); err != nil {
		return fmt.Errorf("create dispatch control: %w", err)
	}
	if err := scope.record(ctx, "dispatch_controls", dispatchControl.ID); err != nil {
		return err
	}

	shipmentControl := &tenant.ShipmentControl{
		ID:                     pulid.MustNew("sc_"),
		OrganizationID:         scope.OrganizationID,
		BusinessUnitID:         scope.BusinessUnitID,
		CheckForDuplicateBOLs:  true,
		CheckHazmatSegregation: true,
		CreatedAt:              now,
		UpdatedAt:              now,
	}
	if _, err := db.NewInsert().Model(shipmentControl).Exec(ctx); err != nil {
		return fmt.Errorf("create shipment control: %w", err)
	}
	if err := scope.record(ctx, "shipment_controls", shipmentControl.ID); err != nil {
		return err
	}

	if err := createCostingControl(ctx, db, scope, now); err != nil {
		return err
	}

	documentControl := tenant.NewDefaultDocumentControl(scope.OrganizationID, scope.BusinessUnitID)
	if _, err := db.NewInsert().Model(documentControl).Exec(ctx); err != nil {
		return fmt.Errorf("create document control: %w", err)
	}
	if err := scope.record(ctx, "document_controls", documentControl.ID); err != nil {
		return err
	}

	dataEntryControl := &dataentrycontrol.DataEntryControl{
		ID:             pulid.MustNew("dec_"),
		OrganizationID: scope.OrganizationID,
		BusinessUnitID: scope.BusinessUnitID,
		CreatedAt:      now,
		UpdatedAt:      now,
		CodeCase:       dataentrycontrol.CaseFormatUpper,
		NameCase:       dataentrycontrol.CaseFormatTitleCase,
		EmailCase:      dataentrycontrol.CaseFormatLower,
		CityCase:       dataentrycontrol.CaseFormatTitleCase,
	}
	if _, err := db.NewInsert().Model(dataEntryControl).Exec(ctx); err != nil {
		return fmt.Errorf("create data entry control: %w", err)
	}

	return scope.record(ctx, "data_entry_controls", dataEntryControl.ID)
}

func createCostingControl(ctx context.Context, db bun.IDB, scope Scope, now int64) error {
	costingControl := &costingcontrol.CostingControl{
		ID:                   pulid.MustNew("cstc_"),
		OrganizationID:       scope.OrganizationID,
		BusinessUnitID:       scope.BusinessUnitID,
		UseLiveFuelPrice:     false,
		MilesPerGallon:       costingcontrol.DefaultMilesPerGallon(),
		IncludeDeadheadMiles: true,
		GLRollingMonths:      3,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	if _, err := db.NewInsert().Model(costingControl).Exec(ctx); err != nil {
		return fmt.Errorf("create costing control: %w", err)
	}
	if err := scope.record(ctx, "costing_controls", costingControl.ID); err != nil {
		return err
	}

	costCategories := costingcontrol.DefaultCategories()
	for _, category := range costCategories {
		category.ID = pulid.MustNew("ccat_")
		category.OrganizationID = scope.OrganizationID
		category.BusinessUnitID = scope.BusinessUnitID
		category.CostingControlID = costingControl.ID
		category.CreatedAt = now
		category.UpdatedAt = now
	}
	if _, err := db.NewInsert().Model(&costCategories).Exec(ctx); err != nil {
		return fmt.Errorf("create cost categories: %w", err)
	}
	for _, category := range costCategories {
		if err := scope.record(ctx, "cost_categories", category.ID); err != nil {
			return err
		}
	}

	return nil
}

var defaultSequenceTypes = []tenant.SequenceType{
	tenant.SequenceTypeProNumber,
	tenant.SequenceTypeConsolidation,
	tenant.SequenceTypeInvoice,
	tenant.SequenceTypeWorkOrder,
	tenant.SequenceTypeJournalBatch,
	tenant.SequenceTypeJournalEntry,
	tenant.SequenceTypeManualJournalRequest,
}

func DefaultSequenceTypes() []tenant.SequenceType {
	types := make([]tenant.SequenceType, len(defaultSequenceTypes))
	copy(types, defaultSequenceTypes)
	return types
}

func CreateSequences(ctx context.Context, db bun.IDB, scope Scope) error {
	now := scope.now()
	year, month := sequencePeriod(now)

	for _, sequenceType := range defaultSequenceTypes {
		sequence := &tenant.Sequence{
			ID:              pulid.MustNew("seq_"),
			SequenceType:    sequenceType,
			OrganizationID:  scope.OrganizationID,
			BusinessUnitID:  scope.BusinessUnitID,
			Year:            year,
			Month:           month,
			CurrentSequence: 0,
			Version:         0,
			CreatedAt:       now,
			UpdatedAt:       now,
		}

		result, err := db.NewInsert().
			Model(sequence).
			On("CONFLICT (sequence_type, organization_id, business_unit_id, year, month) DO NOTHING").
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("create sequence seed: %w", err)
		}

		if rows, _ := result.RowsAffected(); rows > 0 {
			if err = scope.record(ctx, "sequences", sequence.ID); err != nil {
				return err
			}
		}
	}

	return nil
}

func sequencePeriod(now int64) (year, month int16) {
	at := time.Unix(now, 0)
	return int16(at.Year()), int16(at.Month())
}
