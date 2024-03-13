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
		field.UUID("organization_id", uuid.UUID{}).
			StructTag(`json:"organizationId"`),
		field.UUID("business_unit_id", uuid.UUID{}).
			StructTag(`json:"businessUnitId"`),
		field.Int64("rec_threshold").
			Default(50).
			Positive().
			StructTag(`json:"recThreshold"`),
		field.Enum("rec_threshold_action").
			Values("Halt", "Warn").
			Default("Halt").
			StructTag(`json:"recThresholdAction"`),
		field.Bool("auto_create_journal_entries").
			Default(false).
			StructTag(`json:"autoCreateJournalEntries"`),
		field.Bool("restrict_manual_journal_entries").
			Default(false).
			StructTag(`json:"restrictManualJournalEntries"`),
		field.Bool("require_journal_entry_approval").
			Default(false).
			StructTag(`json:"requireJournalEntryApproval"`),
		field.Bool("enable_rec_notifications").
			Default(true).
			StructTag(`json:"enableRecNotifications"`),
		field.Bool("halt_on_pending_rec").
			Default(false).
			StructTag(`json:"haltOnPendingRec"`),
		field.Text("critical_processes").StructTag(`json:"criticalProcesses"`),
		field.UUID("default_rev_account_id", uuid.UUID{}).
			Optional().
			StructTag(`json:"defaultRevAccountId"`),
		field.UUID("default_exp_account_id", uuid.UUID{}).
			Optional().
			StructTag(`json:"defaultExpAccountId"`),
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
		edge.To("organization", Organization.Type).
			Field("organization_id").
			Annotations(entsql.OnDelete(entsql.Cascade)).
			Required().
			Unique(),
		edge.To("business_unit", BusinessUnit.Type).
			Field("business_unit_id").
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
