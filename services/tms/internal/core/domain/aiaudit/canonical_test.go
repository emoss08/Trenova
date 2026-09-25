package aiaudit

import (
	"encoding/json" //nolint:depguard // the test decodes as bun does, with json.Number
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sampleEvent() *AIAuditEvent {
	version := int64(7)
	cost := decimal.RequireFromString("0.012300")
	latency := int64(1234)

	return &AIAuditEvent{
		ID:                     pulid.ID("aiae_01J9AAAAAAAAAAAAAAAAAAAAAA"),
		OrganizationID:         pulid.ID("org_01J9AAAAAAAAAAAAAAAAAAAAAA"),
		BusinessUnitID:         pulid.ID("bu_01J9AAAAAAAAAAAAAAAAAAAAAA"),
		Seq:                    1,
		SourceKey:              "step:ar_1:key",
		OccurredAt:             1_760_000_000,
		RecordedAt:             1_760_000_010,
		Kind:                   KindToolCall,
		Outcome:                OutcomeRan,
		PrincipalType:          PrincipalUser,
		PrincipalID:            "usr_1",
		AgentDefinitionVersion: &version,
		CostUSD:                &cost,
		LatencyMs:              &latency,
		ToolName:               "assign_worker",
		HeldBy:                 []string{"tainted"},
		Arguments: map[string]any{
			"shipmentId": "shp_1",
			"weight":     1.5,
			"count":      json.Number("3"),
			"nested":     map[string]any{"big": 1e21, "list": []any{1, "a", true, nil}},
		},
		ArgumentSensitivity: &ArgumentSensitivity{
			Resource: permission.ResourceShipment,
			Levels: map[string]permission.FieldSensitivity{
				"weight": permission.SensitivityRestricted,
			},
		},
		Taint: &agent.RunTaint{Marks: []agent.TaintMark{{
			Source: agent.TaintSource(
				"web",
			), ToolName: "web_search", CallID: "c1", At: 1_760_000_000,
		}}},
		Purpose: PurposeLive,
	}
}

func TestCanonicalBytes_IsStableAcrossAJSONBRoundTrip(t *testing.T) {
	t.Parallel()

	original := sampleEvent()
	before, err := CanonicalBytes(original)
	require.NoError(t, err)

	stored, err := sonic.Marshal(original.Arguments)
	require.NoError(t, err)
	var reread map[string]any
	decoder := json.NewDecoder(strings.NewReader(strings.ReplaceAll(
		string(stored), "1e+21", "1000000000000000000000",
	)))
	decoder.UseNumber()
	require.NoError(t, decoder.Decode(&reread))

	roundTripped := sampleEvent()
	roundTripped.Arguments = reread
	cost := decimal.RequireFromString("0.0123")
	roundTripped.CostUSD = &cost

	after, err := CanonicalBytes(roundTripped)
	require.NoError(t, err)
	assert.Equal(t, string(before), string(after))
}

func TestCanonicalBytes_SortsKeysAndWritesExactNumbers(t *testing.T) {
	t.Parallel()

	content, err := CanonicalBytes(sampleEvent())
	require.NoError(t, err)

	text := string(content)
	assert.Contains(
		t,
		text,
		`"arguments":{"count":3,"nested":{"big":1000000000000000000000,"list":[1,"a",true,null]},"shipmentId":"shp_1","weight":1.5}`,
	)
	assert.Contains(t, text, `"costUsd":"0.012300"`)
	assert.Contains(t, text, `"heldBy":["tainted"]`)
	assert.Contains(t, text, `"redactedPaths":[]`)
}

func TestSeal_SignsAndVerifies(t *testing.T) {
	t.Parallel()

	key := &ChainKey{ID: "k1", Secret: []byte(strings.Repeat("s", 32))}
	event := sampleEvent()
	require.NoError(t, Seal(event, GenesisHash, key))

	assert.Equal(t, HashVersionSigned, event.HashVersion)
	assert.Equal(t, "k1", event.HashKeyID)
	assert.Len(t, event.Hash, 64)

	ok, err := VerifyHash(event, key)
	require.NoError(t, err)
	assert.True(t, ok)

	other := &ChainKey{ID: "k1", Secret: []byte(strings.Repeat("t", 32))}
	ok, err = VerifyHash(event, other)
	require.NoError(t, err)
	assert.False(t, ok, "another secret under the same id does not verify")

	_, err = VerifyHash(event, nil)
	require.ErrorIs(t, err, ErrChainKeyRequired)
}

func TestSeal_EveryContentColumnIsCovered(t *testing.T) {
	t.Parallel()

	key := &ChainKey{ID: "k1", Secret: []byte(strings.Repeat("s", 32))}
	tamper := map[string]func(*AIAuditEvent){
		"outcome":    func(e *AIAuditEvent) { e.Outcome = OutcomeFailed },
		"seq":        func(e *AIAuditEvent) { e.Seq = 2 },
		"org":        func(e *AIAuditEvent) { e.OrganizationID = "org_other" },
		"arguments":  func(e *AIAuditEvent) { e.Arguments["shipmentId"] = "shp_2" },
		"cost":       func(e *AIAuditEvent) { c := decimal.RequireFromString("9"); e.CostUSD = &c },
		"prev":       func(e *AIAuditEvent) { e.PrevHash = strings.Repeat("1", 64) },
		"keyId":      func(e *AIAuditEvent) { e.HashKeyID = "k2" },
		"taint":      func(e *AIAuditEvent) { e.Taint = nil },
		"heldBy":     func(e *AIAuditEvent) { e.HeldBy = nil },
		"simulated":  func(e *AIAuditEvent) { e.Simulated = true },
		"occurredAt": func(e *AIAuditEvent) { e.OccurredAt++ },
	}

	for name, mutate := range tamper {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			event := sampleEvent()
			require.NoError(t, Seal(event, GenesisHash, key))
			mutate(event)

			ok, err := VerifyHash(event, &ChainKey{ID: event.HashKeyID, Secret: key.Secret})
			require.NoError(t, err)
			assert.False(t, ok)
		})
	}
}

func TestSeal_WithoutAKeyWritesAnUnsignedLink(t *testing.T) {
	t.Parallel()

	event := sampleEvent()
	require.NoError(t, Seal(event, GenesisHash, nil))

	assert.Equal(t, HashVersionUnsigned, event.HashVersion)
	assert.Empty(t, event.HashKeyID)

	ok, err := VerifyHash(event, nil)
	require.NoError(t, err)
	assert.True(t, ok)

	signed := sampleEvent()
	require.NoError(t, Seal(signed, GenesisHash, &ChainKey{ID: "k", Secret: []byte("x")}))
	assert.NotEqual(t, event.Hash, signed.Hash)
}
