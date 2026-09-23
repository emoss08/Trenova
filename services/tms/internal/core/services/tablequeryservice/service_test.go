package tablequeryservice

import (
	"context"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

/*
The composer's contract is that a wrong guess is visible and harmless.

A model naming a column that does not exist, an operator the kind rejects or a
status nobody ever defined is the expected case, not the exceptional one — the
whole design assumes it, which is why nothing it produces reaches SQL without
going through the same compiler the list tools use. What these tests hold is
the consequence: every such guess lands in "unresolved" with a reason, the rest
of the question still compiles, and the explanation describes what was applied
rather than what was asked.

The failure this prevents is a filter that silently did not apply. That is the
worst outcome available here: the table comes back looking answered, and the
one condition the person cared about is the one that went missing.
*/

type stubCompletion struct {
	serviceports.CompletionService
	reply string
	err   error
	saw   *serviceports.StructuredCompletionRequest
}

func (s *stubCompletion) CompleteStructured(
	_ context.Context,
	req *serviceports.StructuredCompletionRequest,
) (*serviceports.StructuredCompletionResult, error) {
	s.saw = req
	if s.err != nil {
		return nil, s.err
	}

	return &serviceports.StructuredCompletionResult{Text: s.reply, ModelIdentifier: "stub"}, nil
}

type stubPermissions struct {
	serviceports.PermissionEngine
	allowed bool
	asked   string
}

func (s *stubPermissions) Check(
	_ context.Context,
	req *serviceports.PermissionCheckRequest,
) (*serviceports.PermissionCheckResult, error) {
	s.asked = req.Resource

	return &serviceports.PermissionCheckResult{Allowed: s.allowed}, nil
}

func shipmentCatalog() *filtercatalog.Catalog {
	return filtercatalog.New(filtercatalog.Resource{
		Tool:     "list_shipments",
		Resource: permission.ResourceShipment,
		Entity:   "shipments",
		Summary:  "Shipments and their status.",
		Fields: []filtercatalog.Field{
			{
				Name:   "status",
				Kind:   filtercatalog.KindEnum,
				Values: []string{"InTransit", "Completed"},
			},
			{Name: "proNumber", Kind: filtercatalog.KindText, Sortable: true},
			{Name: "actualShipDate", Kind: filtercatalog.KindDate, Sortable: true},
		},
	})
}

func newService(reply string, allowed bool) (*Service, *stubCompletion, *stubPermissions) {
	completion := &stubCompletion{reply: reply}
	permissions := &stubPermissions{allowed: allowed}

	return &Service{
		logger:      zap.NewNop(),
		completion:  completion,
		permissions: permissions,
		catalog:     shipmentCatalog(),
	}, completion, permissions
}

func request(prompt string) *ComposeRequest {
	return &ComposeRequest{
		Actor:    &serviceports.RequestActor{},
		Resource: permission.ResourceShipment,
		Prompt:   prompt,
		Timezone: "UTC",
	}
}

func reply(t *testing.T, body map[string]any) string {
	t.Helper()
	text, err := sonic.MarshalString(body)
	require.NoError(t, err)

	return text
}

func TestCompose_CompilesWhatTheModelNames(t *testing.T) {
	t.Parallel()

	service, _, _ := newService(reply(t, map[string]any{
		"filters": []any{
			map[string]any{"field": "status", "operator": "eq", "value": "InTransit"},
		},
	}), true)

	result, err := service.Compose(t.Context(), request("shipments in transit"))

	require.NoError(t, err)
	require.Len(t, result.FieldFilters, 1)
	assert.Equal(t, "status", result.FieldFilters[0].Field)
	assert.Equal(t, "InTransit", result.FieldFilters[0].Value)
	assert.Empty(t, result.Unresolved)
	assert.Equal(t, "shipments where status equals InTransit", result.Explanation)
}

// One bad guess costs its own condition, not the request. Somebody who asked
// for three things and got two, with a line saying why, has been served.
func TestCompose_KeepsTheGoodFiltersWhenOneIsRefused(t *testing.T) {
	t.Parallel()

	service, _, _ := newService(reply(t, map[string]any{
		"filters": []any{
			map[string]any{"field": "status", "operator": "eq", "value": "InTransit"},
			map[string]any{"field": "driverMood", "operator": "eq", "value": "cheerful"},
		},
	}), true)

	result, err := service.Compose(t.Context(), request("in transit and the driver is happy"))

	require.NoError(t, err)
	require.Len(t, result.FieldFilters, 1)
	require.Len(t, result.Unresolved, 1)
	assert.Contains(t, result.Unresolved[0].Phrase, "driverMood")
	assert.Contains(t, result.Unresolved[0].Reason, "proNumber")
}

func TestCompose_RefusesAValueOutsideTheEnumWithoutLosingTheRest(t *testing.T) {
	t.Parallel()

	service, _, _ := newService(reply(t, map[string]any{
		"filters": []any{
			map[string]any{"field": "status", "operator": "eq", "value": "Teleporting"},
			map[string]any{"field": "proNumber", "operator": "contains", "value": "SEED"},
		},
	}), true)

	result, err := service.Compose(t.Context(), request("teleporting shipments starting SEED"))

	require.NoError(t, err)
	require.Len(t, result.FieldFilters, 1)
	assert.Equal(t, "proNumber", result.FieldFilters[0].Field)
	require.Len(t, result.Unresolved, 1)
	assert.Contains(t, result.Unresolved[0].Reason, "InTransit, Completed")
}

// The explanation is built from what compiled, never from the prompt, so it
// cannot describe a filter that is not on the table.
func TestCompose_ExplainsOnlyWhatWasApplied(t *testing.T) {
	t.Parallel()

	service, _, _ := newService(reply(t, map[string]any{
		"filters": []any{
			map[string]any{"field": "driverMood", "operator": "eq", "value": "cheerful"},
		},
	}), true)

	result, err := service.Compose(t.Context(), request("happy drivers"))

	require.NoError(t, err)
	assert.Equal(t, "Every shipments, unfiltered.", result.Explanation)
	assert.Empty(t, result.FieldFilters)
	assert.NotEmpty(t, result.Unresolved)
}

func TestCompose_CarriesWhatTheModelCouldNotExpress(t *testing.T) {
	t.Parallel()

	service, _, _ := newService(reply(t, map[string]any{
		"filters": []any{},
		"unresolved": []any{
			map[string]any{"phrase": "profitable ones", "reason": "margin is not a field here"},
		},
	}), true)

	result, err := service.Compose(t.Context(), request("the profitable ones"))

	require.NoError(t, err)
	require.Len(t, result.Unresolved, 1)
	assert.Equal(t, "profitable ones", result.Unresolved[0].Phrase)
}

func TestCompose_ResolvesARelativeWindowOnTheServerClock(t *testing.T) {
	t.Parallel()

	service, _, _ := newService(reply(t, map[string]any{
		"filters": []any{
			map[string]any{"field": "actualShipDate", "operator": "nextndays", "days": 7},
		},
	}), true)

	result, err := service.Compose(t.Context(), request("shipping in the next week"))

	require.NoError(t, err)
	// Both ends, or "the next 7 days" also matches everything from years ago.
	require.Len(t, result.FieldFilters, 2)
	assert.Equal(t, dbtype.OpGreaterThanOrEqual, result.FieldFilters[0].Operator)
	assert.Equal(t, dbtype.OpLessThanOrEqual, result.FieldFilters[1].Operator)
}

func TestCompose_RefusesASortTheResourceCannotDo(t *testing.T) {
	t.Parallel()

	service, _, _ := newService(reply(t, map[string]any{
		"filters": []any{},
		"sortBy":  "status",
	}), true)

	result, err := service.Compose(t.Context(), request("sorted by status"))

	require.NoError(t, err)
	assert.Empty(t, result.Sort)
	require.Len(t, result.Unresolved, 1)
	assert.Contains(t, result.Unresolved[0].Phrase, "status")
}

/*
The permission check comes before the model call, not after it.

A composed filter is a read of the resource by another name. Checking
afterwards would mean the organization had already paid for the call and the
model had already been shown the entity's whole field catalogue — which is a
description of data the asker is not allowed to see.
*/
func TestCompose_RefusesBeforePayingForTheCall(t *testing.T) {
	t.Parallel()

	service, completion, permissions := newService("{}", false)

	_, err := service.Compose(t.Context(), request("shipments in transit"))

	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
	assert.Nil(t, completion.saw, "the model was called despite the refusal")
	assert.Equal(t, permission.ResourceShipment.String(), permissions.asked)
}

func TestCompose_RefusesAResourceWithNoCatalogue(t *testing.T) {
	t.Parallel()

	service, completion, _ := newService("{}", true)
	req := request("anything")
	req.Resource = permission.ResourceWorker

	_, err := service.Compose(t.Context(), req)

	require.Error(t, err)
	assert.Nil(t, completion.saw)
}

func TestCompose_RefusesAnEmptyOrOversizedPrompt(t *testing.T) {
	t.Parallel()

	service, completion, _ := newService("{}", true)

	_, err := service.Compose(t.Context(), request("   "))
	require.Error(t, err)

	long := make([]byte, maxPromptRunes+1)
	for i := range long {
		long[i] = 'a'
	}
	_, err = service.Compose(t.Context(), request(string(long)))
	require.Error(t, err)

	assert.Nil(t, completion.saw, "a prompt that fails its bounds never reaches the model")
}

// A reply that is not the object asked for is a failure to translate, not a
// filter. Returning it half-read would put a partial narrowing on the table.
func TestCompose_RefusesAReplyItCannotRead(t *testing.T) {
	t.Parallel()

	service, _, _ := newService("I think you want shipments!", true)

	_, err := service.Compose(t.Context(), request("shipments in transit"))

	require.Error(t, err)
}

// The catalogue is the model's whole vocabulary, so it has to be in front of
// it — and the person's words have to be fenced apart from it as untrusted.
func TestCompose_ShowsTheModelTheCatalogueAndFencesTheQuestion(t *testing.T) {
	t.Parallel()

	service, completion, _ := newService(reply(t, map[string]any{"filters": []any{}}), true)

	_, err := service.Compose(t.Context(), request("shipments in transit"))
	require.NoError(t, err)
	require.NotNil(t, completion.saw)

	sections := map[string]serviceports.ContextSection{}
	for _, section := range completion.saw.Context.Sections {
		sections[section.Title] = section
	}

	entity, ok := sections["entity"]
	require.True(t, ok)
	assert.Contains(t, entity.Content, "status")
	assert.Contains(t, entity.Content, "InTransit, Completed")
	assert.True(t, entity.Trusted)

	question, ok := sections["question"]
	require.True(t, ok)
	assert.False(t, question.Trusted, "a person's words are not instructions")
}

// The schema constrains the field names too, so a provider that enforces it
// cannot return a column that does not exist in the first place.
func TestOutputSchema_NamesOnlyTheResourcesOwnFields(t *testing.T) {
	t.Parallel()

	resource, ok := shipmentCatalog().For(permission.ResourceShipment)
	require.True(t, ok)

	schema := outputSchema(resource)
	properties, _ := schema["properties"].(map[string]any)
	filters, _ := properties["filters"].(map[string]any)
	items, _ := filters["items"].(map[string]any)
	itemProps, _ := items["properties"].(map[string]any)
	field, _ := itemProps["field"].(map[string]any)

	assert.Equal(t, []string{"status", "proNumber", "actualShipDate"}, field["enum"])

	sortBy, _ := properties["sortBy"].(map[string]any)
	assert.Equal(t, []string{"proNumber", "actualShipDate"}, sortBy["enum"])
}
