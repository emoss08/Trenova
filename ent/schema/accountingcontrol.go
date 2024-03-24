package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// AccountingControl holds the schema definition for the AccountingControl entity.
type AccountingControl struct {
	ent.Schema
}

// Fields of the AccountingControl.
func (AccountingControl) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("rec_threshold").
			Default(50).
			Positive().
			StructTag(`json:"recThreshold" validate:"required"`),
		field.Enum("rec_threshold_action").
			Values("Halt", "Warn").
			Default("Halt").
			StructTag(`json:"recThresholdAction" validate:"required"`),
		field.Bool("auto_create_journal_entries").
			Default(false).
			StructTag(`json:"autoCreateJournalEntries" validate:"omitempty"`),
		field.Enum("journal_entry_criteria").
			Values("OnShipmentBill", "OnReceiptOfPayment", "OnExpenseRecognition").
			Default("OnShipmentBill").
			StructTag(`json:"journalEntryCriteria" validate:"required"`),
		field.Bool("restrict_manual_journal_entries").
			Default(false).
			StructTag(`json:"restrictManualJournalEntries" validate:"omitempty"`),
		field.Bool("require_journal_entry_approval").
			Default(false).
			StructTag(`json:"requireJournalEntryApproval" validate:"required"`),
		field.Bool("enable_rec_notifications").
			Default(true).
			StructTag(`json:"enableRecNotifications" validate:"required"`),
		field.Bool("halt_on_pending_rec").
			Default(false).
			StructTag(`json:"haltOnPendingRec" validate:"omitempty"`),
		field.Text("critical_processes").
			Nillable().
			Optional().
			StructTag(`json:"criticalProcesses" validate:"omitempty"`),
		field.UUID("default_rev_account_id", uuid.UUID{}).
			Optional().
			Nillable().
			StructTag(`json:"defaultRevAccountId" validate:"omitempty"`),
		field.UUID("default_exp_account_id", uuid.UUID{}).
			Optional().
			Nillable().
			StructTag(`json:"defaultExpAccountId" validate:"omitempty"`),
	}
}

// Mixin for the AccountingControl.
func (AccountingControl) Mixin() []ent.Mixin {
	return []ent.Mixin{
		DefaultMixin{},
	}
}

// Edges of the AccountingControl.
func (AccountingControl) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).
			Ref("accounting_control").
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Required().
			Unique(),
		edge.To("business_unit", BusinessUnit.Type).
			StorageKey(edge.Column("business_unit_id")).
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Required().
			Unique(),
		edge.To("default_rev_account", GeneralLedgerAccount.Type).
			Field("default_rev_account_id").
			Unique(),
		edge.To("default_exp_account", GeneralLedgerAccount.Type).
			Field("default_exp_account_id").
			Unique(),
	}
}
