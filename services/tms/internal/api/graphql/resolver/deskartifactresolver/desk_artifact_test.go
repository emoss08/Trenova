package deskartifactresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantartifactports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type artifacts struct {
	assistantports.AssistantArtifactService

	thread  *conversationports.GetThreadRequest
	rewrite assistantartifactports.DocumentRewriteRequest
	rows    *assistantartifactports.ArtifactRowsQuery
}

func (a *artifacts) RewriteDocument(
	_ context.Context,
	thread *conversationports.GetThreadRequest,
	_ pulid.ID,
	rewrite assistantartifactports.DocumentRewriteRequest,
) (*assistantartifactports.DocumentRewriteSuggestion, error) {
	a.thread = thread
	a.rewrite = rewrite

	return &assistantartifactports.DocumentRewriteSuggestion{Text: "Shorter."}, nil
}

func (a *artifacts) ArtifactRows(
	_ context.Context,
	thread *conversationports.GetThreadRequest,
	_ pulid.ID,
	query *assistantartifactports.ArtifactRowsQuery,
) (*assistantartifactports.ArtifactRows, error) {
	a.thread = thread
	a.rows = query

	return &assistantartifactports.ArtifactRows{}, nil
}

func core(engine permissionports.PermissionEngine) *resolvercore.Core {
	return &resolvercore.Core{L: zap.NewNop(), PermissionEngine: engine}
}

func granting(grants ...string) *resolvertest.GrantingPermissionEngine {
	granted := make(map[string]bool, len(grants))
	for _, grant := range grants {
		granted[grant] = true
	}

	return &resolvertest.GrantingPermissionEngine{Granted: granted}
}

var assistantRead = permission.ResourceAssistant.String() + "|" + string(permission.OpRead)

func TestEditingAConversationsArtifactNeedsOnlyTheAssistant(t *testing.T) {
	t.Parallel()

	engine := resolvertest.RefusingEngine()
	resolver := &MutationResolver{&Deps{Core: core(engine)}}
	id := pulid.MustNew("aart_").String()

	_, err := resolver.PinAssistantArtifact(
		resolvertest.AuthedContext(t, resolvertest.UserAuthContext()),
		id,
		id,
		true,
	)

	resolvertest.RequireRefused(t, err, engine, permission.ResourceAssistant, permission.OpRead)
}

/* Running a report kept in a conversation exports it, which the person must be allowed. */
func TestRunAssistantArtifactReportAlsoNeedsReportExport(t *testing.T) {
	t.Parallel()

	engine := granting(assistantRead)
	resolver := &MutationResolver{&Deps{Core: core(engine)}}
	id := pulid.MustNew("aart_").String()

	_, err := resolver.RunAssistantArtifactReport(
		resolvertest.AuthedContext(t, resolvertest.UserAuthContext()),
		id,
		id,
	)

	require.Error(t, err)
	require.Len(t, engine.Requests, 2)
	assert.Equal(t, permission.ResourceReport.String(), engine.Requests[1].Resource)
	assert.Equal(t, permission.OpExport, engine.Requests[1].Operation)
}

func TestRewriteAssistantDocumentAnswersTheSuggestedText(t *testing.T) {
	t.Parallel()

	auth := resolvertest.UserAuthContext()
	service := &artifacts{}
	resolver := &MutationResolver{&Deps{
		Core:                     core(granting(assistantRead)),
		AssistantArtifactService: service,
	}}
	threadID := pulid.MustNew("athr_")

	text, err := resolver.RewriteAssistantDocument(
		resolvertest.AuthedContext(t, auth),
		threadID.String(),
		pulid.MustNew("aart_").String(),
		"A long passage.",
		"shorter",
		"",
	)

	require.NoError(t, err)
	assert.Equal(t, "Shorter.", text)
	assert.Equal(t, assistantartifactports.DocumentRewriteMode("shorter"), service.rewrite.Mode)
	assert.Equal(t, "A long passage.", service.rewrite.Text)
	assert.Equal(t, threadID, service.thread.ID)
	assert.Equal(t, auth.UserID, service.thread.UserID)
}

func TestAssistantArtifactRowsMapsThePageAsked(t *testing.T) {
	t.Parallel()

	service := &artifacts{}
	resolver := &QueryResolver{&Deps{
		Core:                     core(granting(assistantRead)),
		AssistantArtifactService: service,
	}}
	after, sortBy, direction, total := "cur_1", "pro", "desc", true

	_, err := resolver.AssistantArtifactRows(
		resolvertest.AuthedContext(t, resolvertest.UserAuthContext()),
		pulid.MustNew("athr_").String(),
		pulid.MustNew("aart_").String(),
		gqlmodel.AssistantArtifactRowsInput{
			Limit:         50,
			After:         &after,
			SortBy:        &sortBy,
			SortDirection: &direction,
			Total:         &total,
		},
	)

	require.NoError(t, err)
	assert.Equal(t, &assistantartifactports.ArtifactRowsQuery{
		After:         "cur_1",
		Limit:         50,
		SortBy:        "pro",
		SortDirection: "desc",
		IncludeTotal:  true,
	}, service.rows)
}
