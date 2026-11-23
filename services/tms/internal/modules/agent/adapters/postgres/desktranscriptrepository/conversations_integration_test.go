//go:build integration

package desktranscriptrepository_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/agent/adapters/postgres/conversationrepository"
	"github.com/emoss08/trenova/internal/modules/agent/adapters/postgres/desktranscriptrepository"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/desktranscript"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/desktranscriptports"
	"github.com/emoss08/trenova/internal/testutil/entitlementtest"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestOversightReadsEveryOwnerButOnlyItsTenantAndOwnerReadsStayOwnerOnly(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	other := seedtest.SeedAdditionalTenant(t, ctx, db, "DT")
	tenantCtx := data.Ctx(ctx)
	otherCtx := other.Ctx(ctx)
	tenant := data.Info()
	params := desktranscriptrepository.Params{
		Entitlements: entitlementtest.Instance(),
		DB:           postgres.WrapDB(db),
		Logger:       zap.NewNop(),
	}
	owned := conversationrepository.New(conversationrepository.Params{
		Entitlements: params.Entitlements,
		DB:           params.DB,
		Logger:       params.Logger,
	})
	oversight := desktranscriptrepository.NewConversations(params)

	thread, err := owned.CreateThread(tenantCtx, &conversation.Thread{
		OrganizationID:    tenant.OrgID,
		BusinessUnitID:    tenant.BuID,
		UserID:            data.User.ID,
		AgentDefinitionID: pulid.MustNew("agdef_"),
		Title:             "Rate check for Northwind",
		Status:            conversation.ThreadStatusArchived,
		Origin:            conversation.ThreadOriginDesk,
	})
	require.NoError(t, err)
	_, err = owned.AppendTurn(tenantCtx, &conversationports.AppendTurnRequest{
		ThreadID:   thread.ID,
		TenantInfo: tenant,
		Messages: []conversation.Message{
			{Role: conversation.RoleUser, Content: "What did we quote Northwind?"},
			{Role: conversation.RoleAssistant, Content: "Two lanes at 2.10 a mile."},
		},
	})
	require.NoError(t, err)

	someoneElse := pulid.MustNew("usr_")
	_, err = owned.GetThread(tenantCtx, &conversationports.GetThreadRequest{
		ID:         thread.ID,
		UserID:     someoneElse,
		TenantInfo: tenant,
	})
	assert.True(t, errortypes.IsNotFoundError(err), "the owner's read is still the owner's alone")
	page, err := owned.ListThreads(tenantCtx, &conversationports.ListThreadsRequest{
		UserID:     someoneElse,
		TenantInfo: tenant,
	})
	require.NoError(t, err)
	assert.Empty(t, page.Items, "another person's rail never lists it")

	read, err := oversight.Get(tenantCtx, desktranscriptports.GetConversationRequest{
		TenantInfo: tenant,
		ID:         thread.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, data.User.ID, read.UserID)
	assert.Empty(
		t,
		read.PinnedFacts,
		"the oversight read picks metadata, not the thread's working state",
	)

	listed, err := oversight.List(tenantCtx, &desktranscriptports.ListConversationsRequest{
		TenantInfo: tenant,
		Filter:     desktranscript.ConversationFilter{Timeframe: desktranscript.AllTime()},
		Cursor:     pagination.CursorInfo{Limit: 10, IncludeTotalCount: true},
	})
	require.NoError(t, err)
	require.Len(t, listed.Items, 1, "archived conversations are listed too")
	require.NotNil(t, listed.TotalCount)
	assert.Equal(t, 1, *listed.TotalCount)

	counts, err := oversight.CountMessages(tenantCtx, desktranscriptports.ConversationsByIDsRequest{
		TenantInfo: tenant,
		IDs:        []pulid.ID{thread.ID},
	})
	require.NoError(t, err)
	assert.Equal(t, 2, counts[thread.ID])

	_, err = oversight.Get(otherCtx, desktranscriptports.GetConversationRequest{
		TenantInfo: other.Info(),
		ID:         thread.ID,
	})
	assert.True(t, errortypes.IsNotFoundError(err), "another organization finds nothing")
	crossTenant, err := oversight.ListByIDs(otherCtx, desktranscriptports.ConversationsByIDsRequest{
		TenantInfo: other.Info(),
		IDs:        []pulid.ID{thread.ID},
	})
	require.NoError(t, err)
	assert.Empty(t, crossTenant)
	messages, err := oversight.ListMessages(otherCtx, &desktranscriptports.ThreadMessagesRequest{
		TenantInfo: other.Info(),
		ThreadIDs:  []pulid.ID{thread.ID},
	})
	require.NoError(t, err)
	assert.Empty(t, messages)
}
