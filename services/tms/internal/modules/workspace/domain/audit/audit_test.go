package audit

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validEntry() *Entry {
	return &Entry{
		ID:             pulid.MustNew("ae_"),
		UserID:         pulid.MustNew("usr_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		Resource:       permission.Resource("shipment"),
		ResourceID:     "shp_01HXYZ",
		Operation:      permission.OpCreate,
		Category:       CategoryUser,
	}
}

func TestEntry_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		modify  func(e *Entry)
		wantErr bool
	}{
		{
			name:    "valid entity passes",
			modify:  func(_ *Entry) {},
			wantErr: false,
		},
		{
			name: "missing organization ID fails",
			modify: func(e *Entry) {
				e.OrganizationID = pulid.ID("")
			},
			wantErr: true,
		},
		{
			name: "missing business unit ID fails",
			modify: func(e *Entry) {
				e.BusinessUnitID = pulid.ID("")
			},
			wantErr: true,
		},
		{
			name: "missing resource fails",
			modify: func(e *Entry) {
				e.Resource = permission.Resource("")
			},
			wantErr: true,
		},
		{
			name: "missing resource ID fails",
			modify: func(e *Entry) {
				e.ResourceID = ""
			},
			wantErr: true,
		},
		{
			name: "missing operation fails",
			modify: func(e *Entry) {
				e.Operation = permission.Operation("")
			},
			wantErr: true,
		},
		{
			name: "missing user ID fails",
			modify: func(e *Entry) {
				e.UserID = pulid.ID("")
			},
			wantErr: true,
		},
		{
			name: "missing category defaults to system",
			modify: func(e *Entry) {
				e.Category = Category("")
			},
			wantErr: false,
		},
		{
			name: "system principal allows null user",
			modify: func(e *Entry) {
				e.PrincipalType = "system"
				e.PrincipalID = pulid.ID("system")
				e.UserID = pulid.Nil
				e.APIKeyID = pulid.Nil
			},
			wantErr: false,
		},
		{
			name: "system principal rejects user id",
			modify: func(e *Entry) {
				e.PrincipalType = "system"
				e.PrincipalID = pulid.ID("system")
				e.APIKeyID = pulid.Nil
			},
			wantErr: true,
		},
		{
			name: "agent principal allows the system user it ran as",
			modify: func(e *Entry) {
				e.PrincipalType = "agent"
				e.PrincipalID = pulid.MustNew("agdef_")
			},
			wantErr: false,
		},
		{
			name: "agent principal allows null user",
			modify: func(e *Entry) {
				e.PrincipalType = "agent"
				e.PrincipalID = pulid.ID("agent")
				e.UserID = pulid.Nil
			},
			wantErr: false,
		},
		{
			name: "agent principal rejects its user as principal",
			modify: func(e *Entry) {
				e.PrincipalType = "agent"
				e.PrincipalID = e.UserID
			},
			wantErr: true,
		},
		{
			name: "agent principal rejects api key id",
			modify: func(e *Entry) {
				e.PrincipalType = "agent"
				e.PrincipalID = pulid.MustNew("agdef_")
				e.APIKeyID = pulid.MustNew("ak_")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			e := validEntry()
			tt.modify(e)

			err := e.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEntry_CreditAgentNamesTheAgentInTheDescription(t *testing.T) {
	t.Parallel()

	described := validEntry()
	described.Comment = "Shipment updated"
	described.CreditAgent("Dispatch Agent")
	assert.Equal(t, "Shipment updated (Ran by Dispatch Agent)", described.Comment)

	bare := validEntry()
	bare.CreditAgent("  Dispatch Agent ")
	assert.Equal(t, "Ran by Dispatch Agent", bare.Comment)

	unknown := validEntry()
	unknown.CreditAgent("")
	assert.Equal(t, "Ran by an agent", unknown.Comment)
}

func TestEntry_BeforeCreate(t *testing.T) {
	t.Parallel()

	t.Run("insert sets Timestamp", func(t *testing.T) {
		t.Parallel()

		e := &Entry{}

		err := e.BeforeCreate(t.Context())
		require.NoError(t, err)

		assert.NotZero(t, e.Timestamp)
	})

	t.Run("insert does not overwrite existing Timestamp", func(t *testing.T) {
		t.Parallel()

		e := &Entry{Timestamp: 1234567890}

		err := e.BeforeCreate(t.Context())
		require.NoError(t, err)

		assert.Equal(t, int64(1234567890), e.Timestamp)
	})

	t.Run("insert sets default category if empty", func(t *testing.T) {
		t.Parallel()

		e := &Entry{}

		err := e.BeforeCreate(t.Context())
		require.NoError(t, err)

		assert.Equal(t, CategorySystem, e.Category)
	})

	t.Run("insert does not overwrite existing category", func(t *testing.T) {
		t.Parallel()

		e := &Entry{Category: CategoryUser}

		err := e.BeforeCreate(t.Context())
		require.NoError(t, err)

		assert.Equal(t, CategoryUser, e.Category)
	})
}

func TestEntry_GetTableName(t *testing.T) {
	t.Parallel()

	e := &Entry{}
	assert.Equal(t, "audit_entries", e.GetTableName())
}

func TestEntry_SearchesTheEntityReferenceAndComment(t *testing.T) {
	t.Parallel()

	config := (&Entry{}).GetPostgresSearchConfig()

	assert.Equal(t, "ae", config.TableAlias)
	assert.False(t, config.UseSearchVector)
	require.Len(t, config.SearchableFields, 2)
	assert.Equal(t, "entity_ref", config.SearchableFields[0].Name)
	assert.Equal(t, "comment", config.SearchableFields[1].Name)
	assert.Equal(t, domaintypes.FieldTypeText, config.SearchableFields[0].Type)
}

func TestEntry_EntityReference(t *testing.T) {
	tests := []struct {
		name  string
		entry *Entry
		want  string
	}{
		{
			name:  "nil entry",
			entry: nil,
			want:  "",
		},
		{
			name:  "empty states",
			entry: &Entry{},
			want:  "",
		},
		{
			name: "current state pro number wins",
			entry: &Entry{
				CurrentState: map[string]any{"proNumber": "S-104829", "name": "Ignored"},
			},
			want: "S-104829",
		},
		{
			name: "key priority within state",
			entry: &Entry{
				CurrentState: map[string]any{"name": "Archer Daniels", "orderNumber": "ORD-0042"},
			},
			want: "ORD-0042",
		},
		{
			name: "falls back to previous state on delete",
			entry: &Entry{
				PreviousState: map[string]any{"invoiceNumber": "INV-2026-0847"},
			},
			want: "INV-2026-0847",
		},
		{
			name: "falls back to metadata",
			entry: &Entry{
				Metadata: map[string]any{"code": "DRYVAN26"},
			},
			want: "DRYVAN26",
		},
		{
			name: "skips non-string and blank values",
			entry: &Entry{
				CurrentState: map[string]any{"number": 42, "name": "   "},
				Metadata:     map[string]any{"username": "maria.chen"},
			},
			want: "maria.chen",
		},
		{
			name: "trims whitespace",
			entry: &Entry{
				CurrentState: map[string]any{"name": "  Walmart DC 6094  "},
			},
			want: "Walmart DC 6094",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.entry.EntityReference())
		})
	}
}

func TestEntry_NormalizeDerivesActionClassChannelAndReference(t *testing.T) {
	t.Parallel()

	e := &Entry{
		Resource:      permission.ResourceInvoice,
		Operation:     permission.OpUpdate,
		PrincipalType: "api_key",
		CurrentState:  map[string]any{"invoiceNumber": " INV-42 "},
	}
	e.Normalize()

	assert.Equal(t, "invoice.update", e.Action)
	assert.Equal(t, RetentionFinancial, e.RetentionClass)
	assert.Equal(t, ChannelAPI, e.Channel)
	assert.Equal(t, "INV-42", e.EntityRef)
}

func TestEntry_ValidateRefusesAnUnknownChannelOrMalformedAction(t *testing.T) {
	t.Parallel()

	e := validEntry()
	e.Channel = Channel("fax")
	require.Error(t, e.Validate())

	e = validEntry()
	e.Action = "Shipment Created"
	require.Error(t, e.Validate())
}

func TestDefaultAction_MakesEachSegmentAValidCode(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "shipment_move.update", DefaultAction("shipment-move", "Update"))
	assert.Equal(t, "x3pl.create", DefaultAction("3pl", "create"))
	assert.True(t, ValidAction(DefaultAction("", "")))
}

func TestRetentionClass_FloorDaysMatchTheDatabase(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 90, RetentionAccess.FloorDays())
	assert.Equal(t, 730, RetentionOperational.FloorDays())
	assert.Equal(t, 2555, RetentionFinancial.FloorDays())
	assert.Equal(t, 730, RetentionSecurity.FloorDays())
}

func TestLegalHold_CoversItsScopeOnlyWhileActive(t *testing.T) {
	t.Parallel()

	org := &LegalHold{}
	assert.True(t, org.Covers(permission.ResourceShipment, "shp_1"))

	kind := &LegalHold{Resource: permission.ResourceShipment}
	assert.True(t, kind.Covers(permission.ResourceShipment, "shp_2"))
	assert.False(t, kind.Covers(permission.ResourceInvoice, "inv_1"))

	record := &LegalHold{Resource: permission.ResourceShipment, ResourceID: "shp_1"}
	assert.True(t, record.Covers(permission.ResourceShipment, "shp_1"))
	assert.False(t, record.Covers(permission.ResourceShipment, "shp_2"))

	released := int64(1)
	record.ReleasedAt = &released
	assert.False(t, record.Covers(permission.ResourceShipment, "shp_1"))
}

func TestAlertKindOf_RaisesSecurityEntriesOnly(t *testing.T) {
	t.Parallel()

	role := &Entry{Category: CategorySecurity, Resource: permission.ResourceRole, Operation: permission.OpUpdate}
	kind, ok := AlertKindOf(role)
	require.True(t, ok)
	assert.Equal(t, AlertRoleChanged, kind)

	_, ok = AlertKindOf(&Entry{Category: CategoryUser, Resource: permission.ResourceRole})
	assert.False(t, ok, "only security entries raise an alert on their own")

	malware := &Entry{Category: CategorySecurity, Action: ActionMalwareRefused}
	kind, ok = AlertKindOf(malware)
	require.True(t, ok)
	assert.Equal(t, AlertMalwareRefused, kind)

	counted, ok := CountedAlertKindOf(&Entry{Operation: permission.OpDelete})
	require.True(t, ok)
	assert.Equal(t, AlertBulkDelete, counted)
	_, ok = CountedAlertKindOf(&Entry{Operation: permission.OpUpdate})
	assert.False(t, ok)
}
