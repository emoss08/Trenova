package shipmentimportassistantservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/shipmentimportchat"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/shipmentimportchatports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger        *zap.Logger
	ChatRepo      shipmentimportchatports.ShipmentImportChatRepository
	ChatCacheRepo shipmentimportchatports.ShipmentImportChatCacheRepository
	PageThreads   agentports.PageThreadRepository `optional:"true"`
}

type Service struct {
	logger        *zap.Logger
	chatRepo      shipmentimportchatports.ShipmentImportChatRepository
	chatCacheRepo shipmentimportchatports.ShipmentImportChatCacheRepository
	pageThreads   agentports.PageThreadRepository
}

var _ shipmentports.ShipmentImportAssistantService = (*Service)(nil)

func New(p Params) *Service {
	return &Service{
		logger:        p.Logger.Named("service.shipment-import-assistant"),
		chatRepo:      p.ChatRepo,
		chatCacheRepo: p.ChatCacheRepo,
		pageThreads:   p.PageThreads,
	}
}

func toHistoryMessages(
	messages []shipmentimportchat.HistoryMessage,
) []shipmentports.ShipmentImportChatMessage {
	converted := make([]shipmentports.ShipmentImportChatMessage, 0, len(messages))
	for _, message := range messages {
		toolCalls := make([]shipmentports.ShipmentImportToolCallRecord, 0, len(message.ToolCalls))
		for _, toolCall := range message.ToolCalls {
			toolCalls = append(toolCalls, shipmentports.ShipmentImportToolCallRecord{
				Name:   toolCall.Name,
				CallID: toolCall.CallID,
				Status: toolCall.Status,
				Input:  toolCall.Input,
				Output: toolCall.Output,
			})
		}

		suggestions := make([]shipmentports.ShipmentImportSuggestion, 0, len(message.Suggestions))
		for _, suggestion := range message.Suggestions {
			suggestions = append(suggestions, shipmentports.ShipmentImportSuggestion{
				Label:       suggestion.Label,
				Prompt:      suggestion.Prompt,
				Type:        suggestion.Type,
				Placeholder: suggestion.Placeholder,
				Action:      suggestion.Action,
				SubmitLabel: suggestion.SubmitLabel,
			})
		}

		converted = append(converted, shipmentports.ShipmentImportChatMessage{
			ID:          message.ID,
			Role:        message.Role,
			Text:        message.Text,
			ToolCalls:   toolCalls,
			Suggestions: suggestions,
			CreatedAt:   message.CreatedAt,
		})
	}

	return converted
}

func (s *Service) getHistorySnapshot(
	ctx context.Context,
	documentID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*shipmentimportchat.HistorySnapshot, error) {
	if s.chatCacheRepo != nil {
		snapshot, err := s.chatCacheRepo.GetHistory(ctx, documentID, tenantInfo)
		if err == nil && snapshot != nil {
			return snapshot, nil
		}
		if err != nil {
			s.logger.Warn(
				"failed to read shipment import chat cache; rebuilding from postgres",
				zap.Error(err),
			)
		}
	}

	chat, err := s.chatRepo.GetConversationByDocument(
		ctx,
		shipmentimportchatports.GetShipmentImportConversationRequest{
			DocumentID: documentID,
			TenantInfo: tenantInfo,
		},
	)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return &shipmentimportchat.HistorySnapshot{
				DocumentID: documentID.String(),
				Status:     shipmentimportchat.ConversationStatusActive,
				Messages:   []shipmentimportchat.HistoryMessage{},
				UpdatedAt:  timeutils.NowUnix(),
			}, nil
		}
		return nil, err
	}

	turns, err := s.chatRepo.ListTurns(ctx, shipmentimportchatports.ListShipmentImportTurnsRequest{
		ConversationID: chat.ID,
		TenantInfo:     tenantInfo,
	})
	if err != nil {
		return nil, err
	}

	messages := make([]shipmentimportchat.HistoryMessage, 0, len(turns)*2)
	for _, turn := range turns {
		messages = append(messages, shipmentimportchat.HistoryMessage{
			ID:        turn.ID.String() + ":user",
			Role:      "user",
			Text:      turn.UserMessage,
			CreatedAt: turn.CreatedAt,
		})

		payload := shipmentimportchat.DecodeTurnPayload(turn)

		messages = append(messages, shipmentimportchat.HistoryMessage{
			ID:          turn.ID.String() + ":assistant",
			Role:        "assistant",
			Text:        turn.AssistantMessage,
			ToolCalls:   payload.ToolCalls,
			Suggestions: payload.Suggestions,
			CreatedAt:   turn.CreatedAt,
		})
	}

	snapshot := &shipmentimportchat.HistorySnapshot{
		DocumentID:     documentID.String(),
		ConversationID: chat.ExternalConversationID,
		Status:         chat.Status,
		StatusReason:   chat.StatusReason,
		TurnCount:      chat.TurnCount,
		LastMessageAt:  chat.LastMessageAt,
		Messages:       messages,
		UpdatedAt:      chat.UpdatedAt,
	}

	if s.chatCacheRepo != nil {
		if cacheErr := s.chatCacheRepo.SetHistory(ctx, snapshot, tenantInfo); cacheErr != nil {
			s.logger.Warn("failed to cache shipment import chat history", zap.Error(cacheErr))
		}
	}

	return snapshot, nil
}

func (s *Service) updateConversationStatus(
	ctx context.Context,
	documentID string,
	tenantInfo pagination.TenantInfo,
	status shipmentimportchat.ConversationStatus,
	reason shipmentimportchat.ConversationStatusReason,
) error {
	id, err := pulid.Parse(documentID)
	if err != nil {
		return errortypes.NewValidationError(
			"documentId",
			errortypes.ErrInvalid,
			"Invalid document ID",
		)
	}

	if err = s.chatRepo.UpdateActiveConversationStatusByDocument(
		ctx,
		id,
		tenantInfo,
		status,
		reason,
	); err != nil {
		return err
	}

	if s.chatCacheRepo != nil {
		if cacheErr := s.chatCacheRepo.DeleteHistory(ctx, id, tenantInfo); cacheErr != nil {
			s.logger.Warn("failed to clear shipment import chat cache", zap.Error(cacheErr))
		}
	}

	return nil
}

func (s *Service) GetHistory(
	ctx context.Context,
	documentID string,
	tenantInfo pagination.TenantInfo,
) (*shipmentports.ShipmentImportChatHistoryResponse, error) {
	id, err := pulid.Parse(documentID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"documentId",
			errortypes.ErrInvalid,
			"Invalid document ID",
		)
	}

	snapshot, err := s.getHistorySnapshot(ctx, id, tenantInfo)
	if err != nil {
		return nil, err
	}

	return &shipmentports.ShipmentImportChatHistoryResponse{
		DocumentID:     documentID,
		ConversationID: snapshot.ConversationID,
		Status:         string(snapshot.Status),
		StatusReason:   string(snapshot.StatusReason),
		TurnCount:      snapshot.TurnCount,
		LastMessageAt:  snapshot.LastMessageAt,
		UpdatedAt:      snapshot.UpdatedAt,
		Messages:       toHistoryMessages(snapshot.Messages),
	}, nil
}

func (s *Service) ArchiveHistory(
	ctx context.Context,
	documentID string,
	tenantInfo pagination.TenantInfo,
) error {
	return s.closeConversations(
		ctx,
		documentID,
		tenantInfo,
		shipmentimportchat.ConversationStatusSuperseded,
		shipmentimportchat.ConversationStatusReasonReextract,
	)
}

func (s *Service) CompleteHistory(
	ctx context.Context,
	documentID string,
	tenantInfo pagination.TenantInfo,
) error {
	return s.closeConversations(
		ctx,
		documentID,
		tenantInfo,
		shipmentimportchat.ConversationStatusCompleted,
		shipmentimportchat.ConversationStatusReasonShipmentCreated,
	)
}

func (s *Service) closeConversations(
	ctx context.Context,
	documentID string,
	tenantInfo pagination.TenantInfo,
	status shipmentimportchat.ConversationStatus,
	reason shipmentimportchat.ConversationStatusReason,
) error {
	if err := s.updateConversationStatus(ctx, documentID, tenantInfo, status, reason); err != nil {
		return err
	}
	if s.pageThreads == nil {
		return nil
	}

	id, err := pulid.Parse(documentID)
	if err != nil {
		return errortypes.NewValidationError(
			"documentId",
			errortypes.ErrInvalid,
			"Invalid document ID",
		)
	}

	closed, err := s.pageThreads.ArchiveSubjectThreads(
		ctx,
		agentports.ArchiveSubjectThreadsRequest{
			TenantInfo:  tenantInfo,
			Origin:      conversation.ThreadOriginImport,
			SubjectType: agent.SubjectDocument,
			SubjectID:   id,
		},
	)
	if err != nil {
		return fmt.Errorf("close the import assistant's conversations about the document: %w", err)
	}
	if closed > 0 {
		s.logger.Info("closed the import assistant's conversations about a document",
			zap.String("documentId", documentID),
			zap.String("status", string(status)),
			zap.Int("closed", closed),
		)
	}

	return nil
}
