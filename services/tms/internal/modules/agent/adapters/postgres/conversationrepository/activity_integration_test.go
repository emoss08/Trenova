//go:build integration

package conversationrepository_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/agent/adapters/postgres/conversationrepository"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentwaitports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestRecordReadSinceFindsOnlyALaterToolResultThatCarriesTheRecord(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)
	data := seedtest.SeedFullTestData(t, ctx, db)
	other := seedtest.SeedAdditionalTenant(t, ctx, db, "CV")
	params := conversationrepository.Params{DB: postgres.WrapDB(db), Logger: zap.NewNop()}
	repo := conversationrepository.New(params)
	activity := conversationrepository.NewConversationActivity(params)
	tenantCtx := data.Ctx(ctx)

	thread, err := repo.CreateThread(tenantCtx, &conversation.Thread{
		OrganizationID:    data.Info().OrgID,
		BusinessUnitID:    data.Info().BuID,
		UserID:            data.User.ID,
		AgentDefinitionID: pulid.MustNew("agd_"),
		Status:            conversation.ThreadStatusActive,
		Origin:            conversation.ThreadOriginPanel,
	})
	require.NoError(t, err)

	watched := pulid.MustNew("bqi_")
	_, err = repo.AppendTurn(tenantCtx, &conversationports.AppendTurnRequest{
		ThreadID:   thread.ID,
		TenantInfo: data.Info(),
		Messages: []conversation.Message{
			{Role: conversation.RoleUser, Content: "ok " + watched.String(), CreatedAt: 300},
			{Role: conversation.RoleTool, ToolName: "billing_get_queue_item",
				Content: `{"id":"` + watched.String() + `"}`, CreatedAt: 200},
			{Role: conversation.RoleTool, ToolName: "billing_get_queue_item", ToolFailed: true,
				Content: `{"error":"` + watched.String() + `"}`, CreatedAt: 400},
		},
	})
	require.NoError(t, err)

	read := func(since int64, record pulid.ID) bool {
		t.Helper()
		found, readErr := activity.RecordReadSince(tenantCtx, &agentwaitports.RecordReadSinceRequest{
			ThreadID:   thread.ID,
			RecordID:   record,
			TenantInfo: data.Info(),
			Since:      since,
		})
		require.NoError(t, readErr)

		return found
	}

	assert.True(t, read(100, watched), "a tool result after the change read the record")
	assert.False(t, read(250, watched),
		"the person naming it and a failed read are not the agent reading it")
	assert.False(t, read(100, pulid.MustNew("bqi_")), "another record was not read")

	elsewhere, err := activity.RecordReadSince(other.Ctx(ctx), &agentwaitports.RecordReadSinceRequest{
		ThreadID:   thread.ID,
		RecordID:   watched,
		TenantInfo: other.Info(),
		Since:      100,
	})
	require.NoError(t, err)
	assert.False(t, elsewhere, "another tenant reads nothing of the conversation")
}
