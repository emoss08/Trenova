package auditservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/auditctx"
	"github.com/emoss08/trenova/pkg/requestmeta"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

func TestLogAction_FillsTheRequestTraceAndChannelFromTheContext(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	srv := newTestService(repo, &noopRealtimeService{})

	traceID, err := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	require.NoError(t, err)
	spanID, err := trace.SpanIDFromHex("00f067aa0ba902b7")
	require.NoError(t, err)
	ctx := trace.ContextWithSpanContext(
		requestmeta.With(t.Context(), requestmeta.New("req-1", "198.51.100.7", "Mozilla/5.0")),
		trace.NewSpanContext(trace.SpanContextConfig{TraceID: traceID, SpanID: spanID}),
	)

	require.NoError(t, srv.LogAction(ctx, validLogActionParams()))

	entry := repo.entries()[0]
	assert.Equal(t, "198.51.100.7", entry.IPAddress)
	assert.Equal(t, "Mozilla/5.0", entry.UserAgent)
	assert.Equal(t, "req-1", entry.CorrelationID)
	assert.Equal(t, "4bf92f3577b34da6a3ce929d0e0e4736", entry.TraceID)
	assert.Equal(t, audit.ChannelWeb, entry.Channel)
	assert.Equal(t, "shipment.update", entry.Action)
	assert.Equal(t, audit.RetentionOperational, entry.RetentionClass)
}

func TestLogAction_AWriteWithNoRequestIsABackgroundJob(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	srv := newTestService(repo, &noopRealtimeService{})

	require.NoError(t, srv.LogAction(t.Context(), validLogActionParams()))

	entry := repo.entries()[0]
	assert.Equal(t, audit.ChannelJob, entry.Channel)
	assert.Empty(t, entry.IPAddress)
	assert.Empty(t, entry.CorrelationID)
}

func TestLogAction_AnAgentToolWriteNamesTheRunAndTheConversation(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	srv := newTestService(repo, &noopRealtimeService{})

	ctx := auditctx.WithAIRun(t.Context(), "atrn_01", true)
	require.NoError(t, srv.LogAction(ctx, validLogActionParams()))

	entry := repo.entries()[0]
	assert.Equal(t, audit.ChannelAgentConversation, entry.Channel)
	assert.Equal(t, "atrn_01", entry.AIRunID)
}

func TestLogAction_AnAPIKeyWriteArrivesThroughTheAPI(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	srv := newTestService(repo, &noopRealtimeService{})

	keyID := pulid.MustNew("ak_")
	params := validLogActionParams()
	params.UserID = pulid.Nil
	params.PrincipalType = permissionports.PrincipalTypeAPIKey
	params.PrincipalID = keyID
	params.APIKeyID = keyID

	require.NoError(t, srv.LogAction(t.Context(), params))
	assert.Equal(t, audit.ChannelAPI, repo.entries()[0].Channel)
}

func TestLogAction_AnUpdateWithADiffKeepsOnlyTheDiff(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	srv := newTestService(repo, &noopRealtimeService{})

	before := map[string]any{"proNumber": "PRO-1", "status": "New"}
	after := map[string]any{"proNumber": "PRO-1", "status": "Assigned"}
	params := validLogActionParams()
	params.PreviousState = before
	params.CurrentState = after

	require.NoError(t, srv.LogAction(t.Context(), params, WithDiff(before, after)))

	entry := repo.entries()[0]
	assert.Nil(t, entry.PreviousState)
	assert.Nil(t, entry.CurrentState)
	assert.Contains(t, entry.Changes, "status")
	assert.Equal(t, "PRO-1", entry.EntityRef, "the entity reference is taken before the states go")
}

func TestLogAction_ACreateKeepsTheNewStateAndADeleteTheOld(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	srv := newTestService(repo, &noopRealtimeService{})

	created := validLogActionParams()
	created.Operation = permission.OpCreate
	created.PreviousState = map[string]any{"status": "x"}
	require.NoError(t, srv.LogAction(t.Context(), created))

	deleted := validLogActionParams()
	deleted.Operation = permission.OpDelete
	deleted.PreviousState = map[string]any{"status": "Canceled"}
	require.NoError(t, srv.LogAction(t.Context(), deleted))

	entries := repo.entries()
	assert.Nil(t, entries[0].PreviousState)
	assert.Equal(t, map[string]any{"status": "Assigned"}, entries[0].CurrentState)
	assert.Nil(t, entries[1].CurrentState)
	assert.Equal(t, map[string]any{"status": "Canceled"}, entries[1].PreviousState)
}

func TestLogAction_KeepsIdentifiersDatesAndContactsReadable(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	srv := newTestService(repo, &noopRealtimeService{})

	params := validLogActionParams()
	params.Operation = permission.OpCreate
	params.CurrentState = map[string]any{
		"customerId":      "cus_01JBXQ4ZP7M3N8K2R5T6V9W0YA",
		"bol":             "123456789",
		"pickupDate":      "03/14/2026",
		"contactEmail":    "ops@acme.com",
		"incomeAccountId": "gla_01JBXQ4ZP7M3N8K2R5T6V9W0YA",
		"healthScore":     float64(92),
	}

	require.NoError(t, srv.LogAction(t.Context(), params))

	assert.Equal(t, params.CurrentState, repo.entries()[0].CurrentState)
	assert.False(t, repo.entries()[0].SensitiveData)
}

func TestLogAction_DropsSecretsAndFingerprintsIdentifiers(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	srv := newTestService(repo, &noopRealtimeService{})

	params := validLogActionParams()
	params.Operation = permission.OpCreate
	params.CurrentState = map[string]any{
		"name":          "Acme",
		"password":      "hunter22",
		"clientSecret":  "abc",
		"taxId":         "12-3456789",
		"tokenCount":    float64(120),
		"configuration": map[string]any{"apiKey": "sk_live_x", "region": "us"},
	}

	require.NoError(t, srv.LogAction(t.Context(), params))

	entry := repo.entries()[0]
	state := entry.CurrentState
	assert.Equal(t, RedactedValue, state["password"])
	assert.Equal(t, RedactedValue, state["clientSecret"])
	assert.Equal(t, float64(120), state["tokenCount"], "a count of tokens is not a token")
	assert.Equal(t, "Acme", state["name"])
	assert.Regexp(t, `^\[REDACTED\] …6789 #[0-9a-f]{8}$`, state["taxId"])
	nested, ok := state["configuration"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, RedactedValue, nested["apiKey"])
	assert.Equal(t, "us", nested["region"])
	assert.True(t, entry.SensitiveData)
}

func TestLogAction_TheSameIdentifierAlwaysHasTheSameFingerprint(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	srv := newTestService(repo, &noopRealtimeService{})

	for range 2 {
		params := validLogActionParams()
		params.Operation = permission.OpCreate
		params.CurrentState = map[string]any{"taxId": "12-3456789"}
		require.NoError(t, srv.LogAction(t.Context(), params))
	}

	entries := repo.entries()
	assert.Equal(t, entries[0].CurrentState["taxId"], entries[1].CurrentState["taxId"])
}

func TestLogAction_WithSecurityIsACriticalSecurityEntryKeptAsSecurity(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	srv := newTestService(repo, &noopRealtimeService{})

	params := validLogActionParams()
	params.Resource = permission.ResourceRole
	require.NoError(t, srv.LogAction(t.Context(), params, WithSecurity(), WithAction("role.assigned")))

	entry := repo.entries()[0]
	assert.Equal(t, audit.CategorySecurity, entry.Category)
	assert.True(t, entry.Critical)
	assert.Equal(t, audit.RetentionSecurity, entry.RetentionClass)
	assert.Equal(t, "role.assigned", entry.Action)
}

func TestLogAction_ClassifiesReadsAsAccessAndMoneyAsFinancial(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	srv := newTestService(repo, &noopRealtimeService{})

	read := validLogActionParams()
	read.Resource = permission.ResourceDocument
	read.Operation = permission.OpRead
	require.NoError(t, srv.LogAction(t.Context(), read))

	invoice := validLogActionParams()
	invoice.Resource = permission.ResourceInvoice
	require.NoError(t, srv.LogAction(t.Context(), invoice))

	entries := repo.entries()
	assert.Equal(t, audit.RetentionAccess, entries[0].RetentionClass)
	assert.Equal(t, audit.RetentionFinancial, entries[1].RetentionClass)
}

func TestLogAction_ReturnsTheErrorWhenTheEntryCannotBeStored(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{insertErr: errors.New("connection refused")}
	srv := newTestService(repo, &noopRealtimeService{})

	err := srv.LogAction(t.Context(), validLogActionParams())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "connection refused")
}

func TestLogAction_RefusesAnEntryWithNoTenant(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	srv := newTestService(repo, &noopRealtimeService{})

	params := validLogActionParams()
	params.OrganizationID = pulid.Nil
	require.Error(t, srv.LogAction(t.Context(), params))
	assert.Empty(t, repo.entries())
	require.ErrorIs(t, srv.LogAction(t.Context(), nil), ErrAuditParamsRequired)
}

func TestLogActions_StoresTheValidEntriesInOneBatchAndRefusesWhenNoneAre(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	srv := newTestService(repo, &noopRealtimeService{})

	invalid := validLogActionParams()
	invalid.ResourceID = ""
	require.NoError(t, srv.LogActions(t.Context(), []auditports.BulkLogEntry{
		{Params: validLogActionParams()},
		{Params: invalid},
		{Params: validLogActionParams()},
	}))
	require.Len(t, repo.inserted, 1)
	assert.Len(t, repo.inserted[0], 2)

	err := srv.LogActions(t.Context(), []auditports.BulkLogEntry{{Params: invalid}})
	require.ErrorIs(t, err, ErrNoValidEntries)
}

func TestLogAction_PublishesRealtimeOnlyAfterTheTransactionCommits(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	realtime := &recordingRealtimeService{}
	srv := newTestService(repo, realtime)

	ctx, hooks := kernelports.WithAfterCommitHooks(t.Context())
	require.NoError(t, srv.LogAction(ctx, validLogActionParams()))
	assert.Empty(t, realtime.requests, "nothing is published inside the transaction")

	hooks.Run(context.WithoutCancel(ctx))
	assert.NotEmpty(t, realtime.requests)
}

func TestListConnection_PassesTheAccessThrough(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	srv := newTestService(repo, &noopRealtimeService{})

	access := auditports.EntryAccess{Resources: []permission.Resource{permission.ResourceShipment}}
	_, err := srv.ListConnection(t.Context(), &auditports.ListAuditEntriesRequest{Access: access})
	require.NoError(t, err)
	assert.Equal(t, access, repo.listReq.Access)
}
