package shipmentimportassistantservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/shipmentimportchat"
	repoports "github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type chatRepoStub struct {
	getConversationByDocumentFn      func(context.Context, repoports.GetShipmentImportConversationRequest) (*shipmentimportchat.Conversation, error)
	listTurnsFn                      func(context.Context, repoports.ListShipmentImportTurnsRequest) ([]*shipmentimportchat.Turn, error)
	updateActiveConversationStatusFn func(context.Context, pulid.ID, pagination.TenantInfo, shipmentimportchat.ConversationStatus, shipmentimportchat.ConversationStatusReason) error
}

func (s *chatRepoStub) GetConversationByDocument(
	ctx context.Context,
	req repoports.GetShipmentImportConversationRequest,
) (*shipmentimportchat.Conversation, error) {
	return s.getConversationByDocumentFn(ctx, req)
}

func (s *chatRepoStub) CreateConversation(
	context.Context,
	*shipmentimportchat.Conversation,
) (*shipmentimportchat.Conversation, error) {
	return nil, errors.New("unexpected CreateConversation call")
}

func (s *chatRepoStub) UpdateConversation(
	context.Context,
	*shipmentimportchat.Conversation,
) (*shipmentimportchat.Conversation, error) {
	return nil, errors.New("unexpected UpdateConversation call")
}

func (s *chatRepoStub) AppendTurn(
	context.Context,
	*shipmentimportchat.Turn,
) (*shipmentimportchat.Turn, error) {
	return nil, errors.New("unexpected AppendTurn call")
}

func (s *chatRepoStub) ListTurns(
	ctx context.Context,
	req repoports.ListShipmentImportTurnsRequest,
) ([]*shipmentimportchat.Turn, error) {
	return s.listTurnsFn(ctx, req)
}

func (s *chatRepoStub) UpdateActiveConversationStatusByDocument(
	ctx context.Context,
	documentID pulid.ID,
	tenantInfo pagination.TenantInfo,
	status shipmentimportchat.ConversationStatus,
	reason shipmentimportchat.ConversationStatusReason,
) error {
	return s.updateActiveConversationStatusFn(ctx, documentID, tenantInfo, status, reason)
}

type chatCacheRepoStub struct {
	deleteHistoryFn func(context.Context, pulid.ID, pagination.TenantInfo) error
}

func (s *chatCacheRepoStub) GetHistory(
	context.Context,
	pulid.ID,
	pagination.TenantInfo,
) (*shipmentimportchat.HistorySnapshot, error) {
	return nil, errors.New("cache miss")
}

func (s *chatCacheRepoStub) SetHistory(
	context.Context,
	*shipmentimportchat.HistorySnapshot,
	pagination.TenantInfo,
) error {
	return nil
}

func (s *chatCacheRepoStub) DeleteHistory(
	ctx context.Context,
	documentID pulid.ID,
	tenantInfo pagination.TenantInfo,
) error {
	if s.deleteHistoryFn != nil {
		return s.deleteHistoryFn(ctx, documentID, tenantInfo)
	}

	return nil
}

func TestGetHistoryIncludesCompletedConversationMetadata(t *testing.T) {
	t.Parallel()

	documentID := pulid.MustNew("doc_")
	conversationID := pulid.MustNew("sic_")
	lastMessageAt := int64(1_717_171_717)
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}

	payload := shipmentimportchat.TurnPayload{
		Suggestions: []shipmentimportchat.HistorySuggestion{{
			Label:       "Confirm rate",
			Prompt:      "Yes, the freight rate is correct.",
			Type:        "prompt",
			SubmitLabel: "Confirm",
		}},
		ToolCalls: []shipmentimportchat.HistoryToolCall{{
			Name:   "search_customers",
			CallID: "call_123",
			Status: "completed",
			Input:  `{"query":"Acme"}`,
			Output: `{"availableCustomers":[]}`,
		}},
	}.Encode()

	svc := &Service{
		logger: zap.NewNop(),
		chatRepo: &chatRepoStub{
			getConversationByDocumentFn: func(_ context.Context, req repoports.GetShipmentImportConversationRequest) (*shipmentimportchat.Conversation, error) {
				require.Equal(t, shipmentimportchat.ConversationStatus(""), req.Status)

				return &shipmentimportchat.Conversation{
					ID:                     conversationID,
					DocumentID:             documentID,
					ExternalConversationID: "resp_123",
					Status:                 shipmentimportchat.ConversationStatusCompleted,
					StatusReason:           shipmentimportchat.ConversationStatusReasonShipmentCreated,
					TurnCount:              1,
					LastMessageAt:          &lastMessageAt,
					UpdatedAt:              1_717_171_800,
				}, nil
			},
			listTurnsFn: func(_ context.Context, req repoports.ListShipmentImportTurnsRequest) ([]*shipmentimportchat.Turn, error) {
				require.Equal(t, conversationID, req.ConversationID)

				return []*shipmentimportchat.Turn{{
					ID:               pulid.MustNew("sit_"),
					UserMessage:      "Create the shipment",
					AssistantMessage: "Everything is set. Ready to create.",
					SuggestionsJSON:  payload.SuggestionsJSON,
					ToolCallsJSON:    payload.ToolCallsJSON,
					CreatedAt:        1_717_171_750,
				}}, nil
			},
			updateActiveConversationStatusFn: func(context.Context, pulid.ID, pagination.TenantInfo, shipmentimportchat.ConversationStatus, shipmentimportchat.ConversationStatusReason) error {
				return errors.New("unexpected status update")
			},
		},
	}

	resp, err := svc.GetHistory(context.Background(), documentID.String(), tenantInfo)
	require.NoError(t, err)

	assert.Equal(t, documentID.String(), resp.DocumentID)
	assert.Equal(t, "resp_123", resp.ConversationID)
	assert.Equal(t, string(shipmentimportchat.ConversationStatusCompleted), resp.Status)
	assert.Equal(t, string(shipmentimportchat.ConversationStatusReasonShipmentCreated), resp.StatusReason)
	assert.Equal(t, 1, resp.TurnCount)
	require.NotNil(t, resp.LastMessageAt)
	assert.Equal(t, lastMessageAt, *resp.LastMessageAt)
	assert.Equal(t, int64(1_717_171_800), resp.UpdatedAt)
	require.Len(t, resp.Messages, 2)
	assert.Equal(t, "assistant", resp.Messages[1].Role)
	require.Len(t, resp.Messages[1].ToolCalls, 1)
	assert.Equal(t, "call_123", resp.Messages[1].ToolCalls[0].CallID)
	require.Len(t, resp.Messages[1].Suggestions, 1)
	assert.Equal(t, "Confirm", resp.Messages[1].Suggestions[0].SubmitLabel)
}

func TestCompleteHistoryMarksConversationCompletedAndClearsCache(t *testing.T) {
	t.Parallel()

	documentID := pulid.MustNew("doc_")
	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}

	var (
		gotStatus shipmentimportchat.ConversationStatus
		gotReason shipmentimportchat.ConversationStatusReason
		deleted   bool
	)

	svc := &Service{
		logger: zap.NewNop(),
		chatRepo: &chatRepoStub{
			getConversationByDocumentFn: func(context.Context, repoports.GetShipmentImportConversationRequest) (*shipmentimportchat.Conversation, error) {
				return nil, errors.New("unexpected GetConversationByDocument call")
			},
			listTurnsFn: func(context.Context, repoports.ListShipmentImportTurnsRequest) ([]*shipmentimportchat.Turn, error) {
				return nil, errors.New("unexpected ListTurns call")
			},
			updateActiveConversationStatusFn: func(_ context.Context, gotDocumentID pulid.ID, gotTenant pagination.TenantInfo, status shipmentimportchat.ConversationStatus, reason shipmentimportchat.ConversationStatusReason) error {
				assert.Equal(t, documentID, gotDocumentID)
				assert.Equal(t, tenantInfo, gotTenant)
				gotStatus = status
				gotReason = reason
				return nil
			},
		},
		chatCacheRepo: &chatCacheRepoStub{
			deleteHistoryFn: func(_ context.Context, gotDocumentID pulid.ID, gotTenant pagination.TenantInfo) error {
				assert.Equal(t, documentID, gotDocumentID)
				assert.Equal(t, tenantInfo, gotTenant)
				deleted = true
				return nil
			},
		},
	}

	err := svc.CompleteHistory(context.Background(), documentID.String(), tenantInfo)
	require.NoError(t, err)

	assert.Equal(t, shipmentimportchat.ConversationStatusCompleted, gotStatus)
	assert.Equal(t, shipmentimportchat.ConversationStatusReasonShipmentCreated, gotReason)
	assert.True(t, deleted)
}

var _ repoports.ShipmentImportChatRepository = (*chatRepoStub)(nil)
var _ repoports.ShipmentImportChatCacheRepository = (*chatCacheRepoStub)(nil)
var _ serviceports.ShipmentImportAssistantService = (*Service)(nil)
