package agentevalcaseservice

import (
	"context"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentquality"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime"
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

func (t fakeQueryTool) Policy() serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
		Name:          t.name,
		Kind:          agent.ToolKindQuery,
		Resource:      t.resource,
		Operation:     permission.OpRead,
		ReadsExternal: agent.ExternalReadNever,
	}
}
func (fakeQueryTool) Query(context.Context, serviceports.QueryToolParams) (any, error) {
	return nil, nil
}

type fakeQueryRegistry struct {
	tools map[string]serviceports.AgentQueryTool
}

func (r fakeQueryRegistry) Get(name string) (serviceports.AgentQueryTool, bool) {
	tool, ok := r.tools[name]
	return tool, ok
}

func (r fakeQueryRegistry) All() []serviceports.AgentQueryTool { return nil }

func (r fakeQueryRegistry) Descriptors() []serviceports.AgentToolDescriptor { return nil }

func testRegistry(t *testing.T) *permission.Registry {
	t.Helper()

	registry := permission.NewEmptyRegistry()
	require.NoError(t, registry.Register(&permission.ResourceDefinition{
		Resource:    resourceCustomer.String(),
		DisplayName: "Customer",
		Category:    "Test",
		Operations:  []permission.OperationDefinition{{Operation: permission.OpRead}},
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
	require.NoError(t, registry.Register(&permission.ResourceDefinition{
		Resource:    resourceLedger.String(),
		DisplayName: "Ledger",
		Category:    "Test",
		Operations:  []permission.OperationDefinition{{Operation: permission.OpRead}},
		FieldSensitivities: map[string]permission.FieldSensitivity{
			"id": permission.SensitivityPublic,
		},
		DefaultSensitivity: permission.SensitivityRestricted,
	}))

	return registry
}

func testRedactor(t *testing.T) *Redactor {
	t.Helper()

	tools := fakeQueryRegistry{tools: map[string]serviceports.AgentQueryTool{
		"get_customer": fakeQueryTool{name: "get_customer", resource: resourceCustomer},
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
	redacted, ok := testRedactor(t).value("get_customer", customerRecord(), out).(map[string]any)
	require.True(t, ok)

	assert.Equal(t, "Acme Foods", redacted["name"])
	assert.Equal(t, "[restricted:creditLimit]", redacted["creditLimit"])
	assert.Equal(t, "[restricted:taxId]", redacted["taxId"], "confidential is above restricted")

	paths := make([]string, 0, len(out.fields))
	for _, field := range out.fields {
		assert.Equal(t, "get_customer", field.Tool)
		paths = append(paths, field.Path)
	}
	assert.Contains(t, paths, "creditLimit")
	assert.Contains(t, paths, "taxId")
}

func TestRedactor_WalksNestedObjectsAndLists(t *testing.T) {
	t.Parallel()

	out := &redaction{}
	redacted, ok := testRedactor(t).value("get_customer", customerRecord(), out).(map[string]any)
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
	customer, ok := redactor.value("get_customer", notes, internalDefault).(map[string]any)
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
		map[string]any{"tools": []any{"get_customer"}},
		out,
	)

	assert.Equal(t, map[string]any{"tools": []any{"get_customer"}}, value)
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
				Name:      "get_customer",
				Arguments: map[string]any{"name": "Acme Foods", "taxId": "12-3456789"},
			}},
		},
		{
			Role:       conversation.RoleTool,
			ToolName:   "get_customer",
			ToolCallID: "call_1",
			Content:    agentruntime.FenceToolResult("get_customer", payload),
		},
		{
			Role:       conversation.RoleTool,
			ToolName:   "get_customer",
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
	assert.Equal(t, "get_customer", name)
	assert.Contains(t, body, "[restricted:creditLimit]")
	assert.NotContains(t, body, "50000")
	assert.NotContains(t, body, "000123456")

	assert.Equal(t, "The tool failed: customer not found", history[3].Content)
}

func TestRedactor_UnreadablePayloadIsRedactedWhole(t *testing.T) {
	t.Parallel()

	out := &redaction{}
	content := testRedactor(t).toolContent(
		"get_customer",
		agentruntime.FenceToolResult("get_customer", `{"creditLimit": 50000, "name": "Ac`),
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
		"get_customer",
		agentruntime.FenceToolResult("get_customer", payload),
		false,
		out,
	).(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "[restricted:creditLimit]", result["creditLimit"])

	failed := redactor.decodedResult("get_customer", " The tool failed ", true, out)
	assert.Equal(t, "The tool failed", failed)
}
