package agentevalcaseservice

import (
	"context"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentruntime"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentquality"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissioncatalog"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permissionspec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	resourceCustomer = permission.Resource("test_customer")
	resourceLedger   = permission.Resource("test_ledger")
)

type fakeQueryTool struct {
	name     string
	resource permission.Resource
}

func (t fakeQueryTool) Name() string              { return t.name }
func (fakeQueryTool) Description() string         { return "" }
func (fakeQueryTool) ParamSchema() map[string]any { return nil }

func (t fakeQueryTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          t.name,
		Kind:          agenttooltrust.ToolKindQuery,
		Resource:      t.resource,
		Operation:     permission.OpRead,
		ReadsExternal: agenttooltrust.ExternalReadNever,
	}
}
func (fakeQueryTool) Query(context.Context, *agenttoolports.QueryToolParams) (any, error) {
	return nil, nil
}

type fakeQueryRegistry struct {
	tools map[string]agenttoolports.AgentQueryTool
}

func (r fakeQueryRegistry) Get(name string) (agenttoolports.AgentQueryTool, bool) {
	tool, ok := r.tools[name]
	return tool, ok
}

func (r fakeQueryRegistry) All() []agenttoolports.AgentQueryTool { return nil }

func (r fakeQueryRegistry) Descriptors() []agenttoolports.AgentToolDescriptor { return nil }

func testRegistry(t *testing.T) *permissioncatalog.Registry {
	t.Helper()

	registry := permissioncatalog.NewEmptyRegistry()
	require.NoError(t, registry.Register(&permissionspec.ResourceDefinition{
		Resource:    resourceCustomer.String(),
		DisplayName: "Customer",
		Category:    "Test",
		Operations:  []permissionspec.OperationDefinition{{Operation: permission.OpRead}},
		FieldSensitivities: map[string]permission.FieldSensitivity{
			"name":                permission.SensitivityPublic,
			"creditLimit":         permission.SensitivityRestricted,
			"taxId":               permission.SensitivityConfidential,
			"billing.bankAccount": permission.SensitivityRestricted,
			"contacts":            permission.SensitivityInternal,
			"email":               permission.SensitivityInternal,
			"personalPhone":       permission.SensitivityRestricted,
		},
		DefaultSensitivity: permission.SensitivityInternal,
	}))
	require.NoError(t, registry.Register(&permissionspec.ResourceDefinition{
		Resource:    resourceLedger.String(),
		DisplayName: "Ledger",
		Category:    "Test",
		Operations:  []permissionspec.OperationDefinition{{Operation: permission.OpRead}},
		FieldSensitivities: map[string]permission.FieldSensitivity{
			"id": permission.SensitivityPublic,
		},
		DefaultSensitivity: permission.SensitivityRestricted,
	}))

	return registry
}

func testRedactor(t *testing.T) *Redactor {
	t.Helper()

	tools := fakeQueryRegistry{tools: map[string]agenttoolports.AgentQueryTool{
		"customer_get": fakeQueryTool{name: "customer_get", resource: resourceCustomer},
		"get_ledger":   fakeQueryTool{name: "get_ledger", resource: resourceLedger},
	}}

	return NewRedactor(testRegistry(t), tools, nil)
}

func customerRecord() map[string]any {
	return map[string]any{
		"name":        "Acme Foods",
		"creditLimit": 50000.0,
		"taxId":       "12-3456789",
		"billing": map[string]any{
			"bankAccount": "000123456",
			"terms":       "Net 30",
		},
		"contacts": []any{
			map[string]any{"email": "ap@acme.test", "personalPhone": "555-0100"},
		},
	}
}

func TestRedactor_ReplacesRestrictedAndConfidentialFields(t *testing.T) {
	t.Parallel()

	out := &redaction{}
	redacted, ok := testRedactor(t).value("customer_get", customerRecord(), out).(map[string]any)
	require.True(t, ok)

	assert.Equal(t, "Acme Foods", redacted["name"])
	assert.Equal(t, "[restricted:creditLimit]", redacted["creditLimit"])
	assert.Equal(t, "[restricted:taxId]", redacted["taxId"], "confidential is above restricted")

	paths := make([]string, 0, len(out.fields))
	for _, field := range out.fields {
		assert.Equal(t, "customer_get", field.Tool)
		paths = append(paths, field.Path)
	}
	assert.Contains(t, paths, "creditLimit")
	assert.Contains(t, paths, "taxId")
}

func TestRedactor_WalksNestedObjectsAndLists(t *testing.T) {
	t.Parallel()

	out := &redaction{}
	redacted, ok := testRedactor(t).value("customer_get", customerRecord(), out).(map[string]any)
	require.True(t, ok)

	billing, ok := redacted["billing"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "[restricted:bankAccount]", billing["bankAccount"],
		"a nested field is matched by its dotted path")
	assert.Equal(t, "Net 30", billing["terms"])

	contacts, ok := redacted["contacts"].([]any)
	require.True(t, ok)
	contact, ok := contacts[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "ap@acme.test", contact["email"])
	assert.Equal(t, "[restricted:personalPhone]", contact["personalPhone"],
		"a field inside a list is matched by its own name")
}

func TestRedactor_UnknownFieldsTakeTheResourceDefault(t *testing.T) {
	t.Parallel()

	redactor := testRedactor(t)

	internalDefault := &redaction{}
	notes := map[string]any{"notes": "call first"}
	customer, ok := redactor.value("customer_get", notes, internalDefault).(map[string]any)
	require.True(t, ok)
	assert.Equal(
		t,
		"call first",
		customer["notes"],
		"an undeclared field on an internal resource stays",
	)
	assert.Empty(t, internalDefault.fields)

	restrictedDefault := &redaction{}
	ledger, ok := redactor.value("get_ledger", map[string]any{
		"id":      "gl_1",
		"balance": 1200.5,
	}, restrictedDefault).(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "gl_1", ledger["id"])
	assert.Equal(t, "[restricted:balance]", ledger["balance"],
		"an undeclared field on a restricted resource is redacted")
}

func TestRedactor_UnknownToolIsRedactedWhole(t *testing.T) {
	t.Parallel()

	out := &redaction{}
	value := testRedactor(t).value("mystery_tool", map[string]any{"anything": 1}, out)

	assert.Equal(t, "[restricted:result]", value)
	require.Len(t, out.fields, 1)
	assert.Equal(t, "result", out.fields[0].Path)
}

func TestRedactor_RuntimeToolsPassThrough(t *testing.T) {
	t.Parallel()

	out := &redaction{}
	value := testRedactor(t).value(
		"find_tools",
		map[string]any{"tools": []any{"customer_get"}},
		out,
	)

	assert.Equal(t, map[string]any{"tools": []any{"customer_get"}}, value)
	assert.Empty(t, out.fields)
}

func TestRedactor_HistoryToolResultsAndArguments(t *testing.T) {
	t.Parallel()

	payload, err := sonic.MarshalString(customerRecord())
	require.NoError(t, err)

	out := &redaction{}
	history := testRedactor(t).History([]agentquality.HistoryMessage{
		{Role: conversation.RoleUser, Content: "What is Acme's credit limit?"},
		{
			Role: conversation.RoleAssistant,
			ToolCalls: []conversation.ToolCallRecord{{
				ID:        "call_1",
				Name:      "customer_get",
				Arguments: map[string]any{"name": "Acme Foods", "taxId": "12-3456789"},
			}},
		},
		{
			Role:       conversation.RoleTool,
			ToolName:   "customer_get",
			ToolCallID: "call_1",
			Content:    agentruntime.FenceToolResult("customer_get", payload),
		},
		{
			Role:       conversation.RoleTool,
			ToolName:   "customer_get",
			ToolCallID: "call_2",
			ToolFailed: true,
			Content:    "The tool failed: customer not found",
		},
	}, out)

	require.Len(t, history, 4)
	assert.Equal(t, "What is Acme's credit limit?", history[0].Content)
	assert.Equal(t, "[restricted:taxId]", history[1].ToolCalls[0].Arguments["taxId"])
	assert.Equal(t, "Acme Foods", history[1].ToolCalls[0].Arguments["name"])

	name, body, fenced := agentruntime.UnfenceToolResult(history[2].Content)
	require.True(t, fenced)
	assert.Equal(t, "customer_get", name)
	assert.Contains(t, body, "[restricted:creditLimit]")
	assert.NotContains(t, body, "50000")
	assert.NotContains(t, body, "000123456")

	assert.Equal(t, "The tool failed: customer not found", history[3].Content)
}

func TestRedactor_UnreadablePayloadIsRedactedWhole(t *testing.T) {
	t.Parallel()

	out := &redaction{}
	content := testRedactor(t).toolContent(
		"customer_get",
		agentruntime.FenceToolResult("customer_get", `{"creditLimit": 50000, "name": "Ac`),
		out,
	)

	_, body, fenced := agentruntime.UnfenceToolResult(content)
	require.True(t, fenced)
	assert.Equal(t, "[restricted:result]", body)
}

func TestRedactor_DecodedFixtureResult(t *testing.T) {
	t.Parallel()

	payload, err := sonic.MarshalString(customerRecord())
	require.NoError(t, err)

	redactor := testRedactor(t)
	out := &redaction{}
	result, ok := redactor.decodedResult(
		"customer_get",
		agentruntime.FenceToolResult("customer_get", payload),
		false,
		out,
	).(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "[restricted:creditLimit]", result["creditLimit"])

	failed := redactor.decodedResult("customer_get", " The tool failed ", true, out)
	assert.Equal(t, "The tool failed", failed)
}
