package aiauditservice

import (
	"context"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiaudit"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakePolicies map[string]agenttoolports.ToolPolicy

func (f fakePolicies) Get(name string) (agenttoolports.ToolPolicy, bool) {
	policy, ok := f[name]

	return policy, ok
}

func testPolicies() fakePolicies {
	return fakePolicies{
		"update_worker": {
			Name:     "update_worker",
			Kind:     agenttooltrust.ToolKindAction,
			Resource: permission.ResourceWorker,
		},
		"pin_my_home": {
			Name:  "pin_my_home",
			Kind:  agenttooltrust.ToolKindAction,
			Scope: agenttooltrust.ToolScopeSelf,
		},
		"shipment_get": {
			Name:     "shipment_get",
			Kind:     agenttooltrust.ToolKindQuery,
			Resource: permission.ResourceShipment,
		},
	}
}

func testRedactor() *Redactor {
	return NewRedactor(
		permissioncatalog.NewRegistry(),
		testPolicies(),
		auditservice.NewSensitiveDataManager(config.EncryptionConfig{}),
	)
}

type fixedCeiling permission.FieldSensitivity

func (c fixedCeiling) For(context.Context, permission.Resource) permission.FieldSensitivity {
	return permission.FieldSensitivity(c)
}

func TestRedactor_ReplacesConfidentialFieldsAndRecordsLevels(t *testing.T) {
	t.Parallel()

	redacted, err := testRedactor().Arguments("update_worker", map[string]any{
		"workerId":  "wrk_01J9AAAAAAAAAAAAAAAAAAAAAA",
		"firstName": "Ada",
		"dob":       "1990-01-01",
		"email":     "ada@example.com",
		"address":   map[string]any{"licenseNumber": "D1234567"},
	})
	require.NoError(t, err)

	assert.Equal(t, aiaudit.ConfidentialPlaceholder, redacted.Values["dob"])
	assert.Equal(t, "wrk_01J9AAAAAAAAAAAAAAAAAAAAAA", redacted.Values["workerId"],
		"a record id is kept whole")
	assert.Equal(t, "Ada", redacted.Values["firstName"])
	assert.NotEqual(t, "ada@example.com", redacted.Values["email"], "an email is masked")
	nested, _ := redacted.Values["address"].(map[string]any)
	assert.Equal(t, aiaudit.ConfidentialPlaceholder, nested["licenseNumber"])

	require.NotNil(t, redacted.Sensitivity)
	assert.Equal(t, permission.ResourceWorker, redacted.Sensitivity.Resource)
	assert.Equal(t, permission.SensitivityRestricted, redacted.Sensitivity.Levels["email"])
	assert.Equal(t, permission.SensitivityConfidential, redacted.Sensitivity.Levels["dob"])
	assert.Contains(t, redacted.RedactedPaths, "dob")
	assert.Contains(t, redacted.RedactedPaths, "address.licenseNumber")
	assert.Contains(t, redacted.RedactedPaths, "email")
}

func TestRedactor_DropsTheSelfScopeOwner(t *testing.T) {
	t.Parallel()

	redacted, err := testRedactor().Arguments("pin_my_home", map[string]any{
		agenttoolports.SelfScopeOwnerParam: "usr_01J9AAAAAAAAAAAAAAAAAAAAAA",
		"widget":                           "shipments",
	})
	require.NoError(t, err)

	assert.NotContains(t, redacted.Values, agenttoolports.SelfScopeOwnerParam)
	assert.Equal(t, "shipments", redacted.Values["widget"])
}

func TestRedactor_BoundsTheArguments(t *testing.T) {
	t.Parallel()

	redacted, err := testRedactor().Arguments("shipment_get", map[string]any{
		"a": strings.Repeat("x ", aiaudit.MaxArgumentsBytes),
		"b": "small",
	})
	require.NoError(t, err)

	assert.True(t, redacted.Truncated)
	assert.Equal(t, aiaudit.TruncatedPlaceholder, redacted.Values["a"])
	assert.Equal(t, "small", redacted.Values["b"])
	encoded, err := canonicalJSON.Marshal(redacted.Values)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(encoded), aiaudit.MaxArgumentsBytes)
}

func TestRedactor_TextMasksInsideSentencesAndStripsNUL(t *testing.T) {
	t.Parallel()

	text := testRedactor().Text("could not reach bob@example.com\x00 today", 500)

	assert.NotContains(t, text, "bob@example.com")
	assert.NotContains(t, text, "\x00")
	assert.True(t, strings.HasPrefix(text, "could not reach "))
	assert.Equal(t, "héllo", testRedactor().Text("héllo wörld", 6), "cut on a character boundary")
}

func TestReaderArguments_WithholdsWhatTheReaderCannotReach(t *testing.T) {
	t.Parallel()

	redacted, err := testRedactor().Arguments("update_worker", map[string]any{
		"firstName": "Ada",
		"email":     "ops@example.com",
		"stops":     []any{map[string]any{"phoneNumber": "555-123-4567", "city": "Reno"}},
	})
	require.NoError(t, err)
	event := &aiaudit.AIAuditEvent{
		Arguments:           redacted.Values,
		ArgumentSensitivity: redacted.Sensitivity,
	}

	internal := ReaderArguments(t.Context(), event, fixedCeiling(permission.SensitivityInternal))
	assert.Equal(t, "Ada", internal["firstName"])
	assert.Equal(t, aiaudit.WithheldPlaceholder, internal["email"])
	assert.Equal(t, aiaudit.WithheldPlaceholder, internal["stops"],
		"a field the registry does not name takes the resource's default, Restricted for a worker")

	restricted := ReaderArguments(
		t.Context(),
		event,
		fixedCeiling(permission.SensitivityRestricted),
	)
	assert.Equal(t, redacted.Values["email"], restricted["email"])
	stops, _ := restricted["stops"].([]any)
	require.Len(t, stops, 1)
	stop, _ := stops[0].(map[string]any)
	assert.Equal(t, "Reno", stop["city"])
	assert.Equal(
		t,
		permission.SensitivityRestricted,
		redacted.Sensitivity.Levels["stops[].phoneNumber"],
	)

	assert.Equal(t, aiaudit.WithheldPlaceholder,
		ReaderArguments(t.Context(), event, nil)["email"],
		"with no ceiling at all the reader is held to Internal")
}

// An alert tool's arguments, read back from the trail, show at Internal only
// what table_change_alert declares Internal (its event types, condition match
// and status); its name, table, conditions and watched columns and any custom
// text stay Restricted.
func TestReaderArguments_KeepsAnAlertsDetailsRestricted(t *testing.T) {
	t.Parallel()

	redactor := NewRedactor(
		permissioncatalog.NewRegistry(),
		fakePolicies{"report_create_table_change_alert": {
			Name:     "report_create_table_change_alert",
			Kind:     agenttooltrust.ToolKindAction,
			Resource: permission.ResourceTableChangeAlert,
		}},
		auditservice.NewSensitiveDataManager(config.EncryptionConfig{}),
	)
	redacted, err := redactor.Arguments("report_create_table_change_alert", map[string]any{
		"name":           "Big invoices",
		"tableName":      "invoices",
		"eventTypes":     []any{"UPDATE"},
		"conditions":     []any{map[string]any{"field": "total", "operator": "gt", "value": 5000}},
		"conditionMatch": "all",
		"watchedColumns": []any{"total"},
		"customTitle":    "Heads up",
		"customMessage":  "A large invoice changed",
		"status":         "Active",
	})
	require.NoError(t, err)
	event := &aiaudit.AIAuditEvent{
		Arguments:           redacted.Values,
		ArgumentSensitivity: redacted.Sensitivity,
	}

	internal := ReaderArguments(t.Context(), event, fixedCeiling(permission.SensitivityInternal))
	for _, shown := range []string{"eventTypes", "conditionMatch", "status"} {
		assert.Equal(t, redacted.Values[shown], internal[shown], shown)
	}
	for _, hidden := range []string{
		"name", "tableName", "conditions", "watchedColumns", "customTitle", "customMessage",
	} {
		assert.Equal(t, aiaudit.WithheldPlaceholder, internal[hidden], hidden)
	}

	restricted := ReaderArguments(
		t.Context(), event, fixedCeiling(permission.SensitivityRestricted),
	)
	assert.Equal(t, "Big invoices", restricted["name"])
}
