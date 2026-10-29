package tenantbootstrap

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/tenantscope"
	"github.com/emoss08/trenova/internal/modules/billing/domain/costingcontrol"
	"github.com/emoss08/trenova/internal/modules/billing/domain/costingcontrol/costingcontroldb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol/accountingcontroldb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/billingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/billingcontrol/billingcontroldb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/documentcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/documentcontrol/documentcontroldb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/invoiceadjustmentcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/invoiceadjustmentcontrol/invoiceadjustmentcontroldb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/sequenceconfig"
	"github.com/emoss08/trenova/internal/modules/identity/domain/sequenceconfig/sequenceconfigdb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/shipmentcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/shipmentcontrol/shipmentcontroldb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol/dispatchcontroldb"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/dataentrycontrol"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/dataentrycontrol/dataentrycontroldb"
	"github.com/emoss08/trenova/shared/pulid"
)

func CreateControls(ctx context.Context, db *keel.DB, scope tenantscope.Scope) error {
	now := scope.NowUnix()

	org, bu := scope.OrganizationID, scope.BusinessUnitID

	accountingControl := accountingcontrol.NewDefaultAccountingControl(org, bu)
	accountingControl.ID = pulid.MustNew("ac_")
	if err := accountingcontroldb.AccountingControls.Create(ctx, db, accountingControl); err != nil {
		return fmt.Errorf("create accounting control: %w", err)
	}

	billingControl := billingcontrol.NewDefaultBillingControl(org, bu)
	billingControl.ID = pulid.MustNew("bc_")
	if err := billingcontroldb.BillingControls.Create(ctx, db, billingControl); err != nil {
		return fmt.Errorf("create billing control: %w", err)
	}

	invoiceAdjustmentControl := invoiceadjustmentcontrol.NewDefaultInvoiceAdjustmentControl(org, bu)
	invoiceAdjustmentControl.ID = pulid.MustNew("iac_")
	if err := invoiceadjustmentcontroldb.InvoiceAdjustmentControls.Create(
		ctx,
		db,
		invoiceAdjustmentControl,
	); err != nil {
		return fmt.Errorf("create invoice adjustment control: %w", err)
	}

	if err := dispatchcontroldb.DispatchControls.Create(
		ctx,
		db,
		newDispatchControl(scope),
	); err != nil {
		return fmt.Errorf("create dispatch control: %w", err)
	}

	shipmentControl := shipmentcontrol.NewDefaultShipmentControl(org, bu)
	shipmentControl.ID = pulid.MustNew("sc_")
	if err := shipmentcontroldb.ShipmentControls.Create(ctx, db, shipmentControl); err != nil {
		return fmt.Errorf("create shipment control: %w", err)
	}

	if err := createCostingControl(ctx, db, scope, now); err != nil {
		return err
	}

	documentControl := documentcontrol.NewDefaultDocumentControl(
		scope.OrganizationID,
		scope.BusinessUnitID,
	)
	if err := documentcontroldb.DocumentControls.Create(ctx, db, documentControl); err != nil {
		return fmt.Errorf("create document control: %w", err)
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
	if err := dataentrycontroldb.DataEntryControls.Create(ctx, db, dataEntryControl); err != nil {
		return fmt.Errorf("create data entry control: %w", err)
	}

	return nil
}

func createCostingControl(
	ctx context.Context,
	db *keel.DB,
	scope tenantscope.Scope,
	now int64,
) error {
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
	if err := costingcontroldb.CostingControls.Create(ctx, db, costingControl); err != nil {
		return fmt.Errorf("create costing control: %w", err)
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
	if err := costingcontroldb.CostCategories.CreateMany(ctx, db, costCategories); err != nil {
		return fmt.Errorf("create cost categories: %w", err)
	}

	return nil
}

var defaultSequenceTypes = []sequenceconfig.SequenceType{
	sequenceconfig.SequenceTypeProNumber,
	sequenceconfig.SequenceTypeConsolidation,
	sequenceconfig.SequenceTypeInvoice,
	sequenceconfig.SequenceTypeWorkOrder,
	sequenceconfig.SequenceTypeJournalBatch,
	sequenceconfig.SequenceTypeJournalEntry,
	sequenceconfig.SequenceTypeManualJournalRequest,
}

func DefaultSequenceTypes() []sequenceconfig.SequenceType {
	types := make([]sequenceconfig.SequenceType, len(defaultSequenceTypes))
	copy(types, defaultSequenceTypes)
	return types
}

func CreateSequences(ctx context.Context, db *keel.DB, scope tenantscope.Scope) error {
	now := scope.NowUnix()
	year, month := sequencePeriod(now)

	sequences := make([]*sequenceconfig.Sequence, 0, len(defaultSequenceTypes))
	for _, sequenceType := range defaultSequenceTypes {
		sequences = append(sequences, &sequenceconfig.Sequence{
			ID:             pulid.MustNew("seq_"),
			SequenceType:   sequenceType,
			OrganizationID: scope.OrganizationID,
			BusinessUnitID: scope.BusinessUnitID,
			Year:           year,
			Month:          month,
			CreatedAt:      now,
			UpdatedAt:      now,
		})
	}

	s := sequenceconfigdb.Sequences
	if _, err := s.CreateOrSkip(
		ctx,
		db,
		sequences,
		s.SequenceType,
		s.OrganizationID,
		s.BusinessUnitID,
		s.Year,
		s.Month,
	); err != nil {
		return fmt.Errorf("create sequences: %w", err)
	}

	return nil
}

func sequencePeriod(now int64) (year, month int16) {
	at := time.Unix(now, 0)
	return int16(at.Year()), int16(at.Month())
}
