//nolint:cyclop // existing legacy workflow/API shape is intentionally kept stable
package shipmentimportassistantservice

import (
	"context"
	"encoding/json" //nolint:depguard // external API payloads
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/shipmentimportchat"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/locationservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const maxLocationCodeLength = 10

type Params struct {
	fx.In

	Logger               *zap.Logger
	Config               *config.Config
	DB                   *postgres.Connection
	Completion           serviceports.CompletionService
	ChatRepo             repositories.ShipmentImportChatRepository
	ChatCacheRepo        repositories.ShipmentImportChatCacheRepository
	CustomerRepo         repositories.CustomerRepository
	LocationRepo         repositories.LocationRepository
	ServiceRepo          repositories.ServiceTypeRepository
	ShipmentTypeRepo     repositories.ShipmentTypeRepository
	FormulaTemplateRepo  repositories.FormulaTemplateRepository
	ShipmentService      serviceports.ShipmentService
	ShipmentControlRepo  repositories.ShipmentControlRepository
	LocationService      *locationservice.Service
	UsStateRepo          repositories.UsStateRepository
	LocationCategoryRepo repositories.LocationCategoryRepository
	Permissions          serviceports.PermissionEngine
	PageThreads          repositories.PageThreadRepository `optional:"true"`
}

type Service struct {
	logger               *zap.Logger
	cfg                  *config.AIConfig
	db                   *postgres.Connection
	completion           serviceports.CompletionService
	chatRepo             repositories.ShipmentImportChatRepository
	chatCacheRepo        repositories.ShipmentImportChatCacheRepository
	customerRepo         repositories.CustomerRepository
	locationRepo         repositories.LocationRepository
	serviceRepo          repositories.ServiceTypeRepository
	shipmentTypeRepo     repositories.ShipmentTypeRepository
	formulaTemplateRepo  repositories.FormulaTemplateRepository
	shipmentService      serviceports.ShipmentService
	shipmentControlRepo  repositories.ShipmentControlRepository
	locationService      *locationservice.Service
	usStateRepo          repositories.UsStateRepository
	locationCategoryRepo repositories.LocationCategoryRepository
	permissions          serviceports.PermissionEngine
	pageThreads          repositories.PageThreadRepository
}

var _ serviceports.ShipmentImportAssistantService = (*Service)(nil)

func New(
	p Params, //nolint:gocritic // stable API shape
) *Service {
	return &Service{
		logger:               p.Logger.Named("service.shipment-import-assistant"),
		cfg:                  p.Config.GetAIConfig(),
		db:                   p.DB,
		completion:           p.Completion,
		chatRepo:             p.ChatRepo,
		chatCacheRepo:        p.ChatCacheRepo,
		customerRepo:         p.CustomerRepo,
		locationRepo:         p.LocationRepo,
		serviceRepo:          p.ServiceRepo,
		shipmentTypeRepo:     p.ShipmentTypeRepo,
		formulaTemplateRepo:  p.FormulaTemplateRepo,
		shipmentService:      p.ShipmentService,
		shipmentControlRepo:  p.ShipmentControlRepo,
		locationService:      p.LocationService,
		usStateRepo:          p.UsStateRepo,
		locationCategoryRepo: p.LocationCategoryRepo,
		permissions:          p.Permissions,
		pageThreads:          p.PageThreads,
	}
}

const systemPrompt = `You are a shipment import assistant helping an operator convert a rate confirmation into a shipment.

TONE: Warm, brief, colleague-like. Say "Found equipment type and a pickup window" not "Extraction: Equipment Type = Van; Pickup Window = 06:00-22:00".

CORE RULES:
- 2-3 sentences max per response. NEVER repeat the same sentence or idea twice.
- ONE FIELD PER TURN. Finish one completely before moving to the next.
- When the user selects a suggestion, that IS their confirmation. Apply it immediately via the appropriate tool. Do NOT ask again.
- If a value was already extracted from the document (check extractedFields/shipmentData), PRESENT it for confirmation first. Don't ask the user to re-enter a value that was already found.
- Prefer the predefined options in defaultOptions before asking the user to type or search.
- Check the requiredFields and settledFields in the context — if a field already has a value, it's been set. Skip it.
- NEVER repeat yourself. If you already said something, do not say it again in the same response.

FLOW (strictly sequential, one turn per step):

PHASE 1 — ENTITY FIELDS (check settledFields — skip any already set):
1. Customer: If shipper name exists, search_customers proactively. If the search result includes "availableCustomers", present those as prompt-type suggestion buttons so the user can pick one. If exact match found, suggest that match.
2. Service type: Prefer defaultOptions.serviceTypes. If needed, call search_service_types with an empty query and present each result as a prompt-type suggestion button. Do NOT default to a generic search input.
3. Shipment type: Prefer defaultOptions.shipmentTypes. If needed, call search_shipment_types with an empty query and present each as a prompt button.
4. Rating method: Prefer defaultOptions.formulaTemplates. If needed, call search_formula_templates with an empty query and present each as a prompt button.

PHASE 2 — STOPS (CRITICAL — DO NOT SKIP):
Check shipmentData._stopsSummary to see how many stops exist and how many need attention.
EVERY stop MUST have a locationId and a scheduledWindowStart before the shipment can be created.
If stopsNeedingAttention > 0, you MUST work through each stop before offering to create.

For EACH stop that needs attention (hasLocation=false or hasValidDate=false):
5. Show the extracted address to the user. Search locations by the stop's city, name, or address.
6. If match found: set via set_stop_location and confirm/set the date.
7. If NO match found: offer to create a new location using add_location with the extracted address data. After creation, set the stop location.
8. Extracted times like "06:00-22:00" are time RANGES, not actual dates. Ask the user for the real pickup/delivery date+time. Use type="date" suggestions for date input.
9. Work through stops ONE AT A TIME. First stop = Pickup, last = Delivery.

DO NOT offer to create the shipment until ALL stops have locations and dates set.

VALIDATION CONTEXT:
The context includes "validationRules" and "shipmentControl" objects.
Before calling create_shipment, verify:
- All 4 required entity fields are set (check settledFields)
- Every stop has a locationId (non-empty) — check each stop's hasLocation field
- Every stop has a scheduledWindowStart > 0 — check each stop's hasValidDate field
- First stop type is Pickup, last is Delivery
- Weight does not exceed maxShipmentWeightLimit (check shipmentControl)
- If customer requires BOL (call get_customer_requirements after setting customer), ensure BOL is set
If ANY of these fail, do NOT call create_shipment. Instead, guide the user to fix the issue.

PHASE 3 — SHIPMENT DETAILS (confirm extracted values):
9. If rate or freightChargeAmount is extracted, say "The freight rate is $X. Is that correct?" with confirm/edit suggestions. Do not ask the user to type it from scratch unless they reject the extracted value.
10. If weight/pieces extracted, present and ask to confirm before requesting manual entry.
11. If BOL is needed and missing, ask for it.

PHASE 4 — CREATE:
12. Everything set → offer create_shipment via action button.

SUGGEST_QUICK_ACTIONS:
- MUST be called at the end of EVERY response — in the LAST round of tool execution, not intermediate rounds.
- Four button types:
  - type="prompt": Click sends a message. For confirmations.
  - type="input": Click shows a text input. Set submitLabel to describe the action (e.g. "Confirm" for values, "Search" for search queries). Set placeholder for hint text.
  - type="date": Click shows a date+time picker. Use when asking for pickup/delivery dates. The selected datetime is sent as ISO 8601 appended to the prompt prefix.
  - type="action": Triggers an app action. Use action="create_shipment" for the final step.
- Suggestions must be DIRECT ANSWERS to the question you just asked.`

func (s *Service) buildConversationContextMap(
	ctx context.Context,
	req *serviceports.ShipmentImportChatRequest,
) map[string]any {
	settledFields := make(map[string]string)
	for k, v := range req.RequiredFields {
		if v != "" {
			settledFields[k] = v
		}
	}

	contextMap := map[string]any{
		"extractedFields": req.ReconciliationState,
		"requiredFields":  req.RequiredFields,
		"settledFields":   settledFields,
		"stops":           req.Stops,
		"shipmentData":    req.ShipmentData,
		"defaultOptions":  s.buildDefaultOptions(ctx, req.TenantInfo),
	}

	control, controlErr := s.shipmentControlRepo.Get(
		ctx,
		repositories.GetShipmentControlRequest{TenantInfo: req.TenantInfo},
	)
	if controlErr == nil && control != nil {
		contextMap["shipmentControl"] = map[string]any{
			"maxShipmentWeightLimit": control.MaxShipmentWeightLimit,
			"checkForDuplicateBols":  control.CheckForDuplicateBOLs,
			"checkHazmatSegregation": control.CheckHazmatSegregation,
		}
		contextMap["validationRules"] = map[string]any{
			"shipment": map[string]any{
				"serviceTypeId":     "REQUIRED",
				"customerId":        "REQUIRED",
				"shipmentTypeId":    "REQUIRED",
				"formulaTemplateId": "REQUIRED",
				"weight": fmt.Sprintf(
					"optional, max %d lbs",
					control.MaxShipmentWeightLimit,
				),
				"bol": "required if customer billing profile requireBOLNumber=true",
			},
			"moves": map[string]any{
				"minMoveCount":    1,
				"minStopsPerMove": 2,
				"firstStopType":   "MUST be Pickup or SplitPickup",
				"lastStopType":    "MUST be Delivery or SplitDelivery",
			},
			"stops": map[string]any{
				"locationId":           "REQUIRED — must be a valid location ID, cannot be empty string",
				"scheduledWindowStart": "REQUIRED — Unix timestamp in seconds, must be > 0",
				"scheduledWindowEnd":   "optional, if set must be >= scheduledWindowStart",
				"type":                 "Pickup | Delivery | SplitPickup | SplitDelivery",
				"scheduleType":         "Open | Appointment",
			},
		}
	}

	return contextMap
}

func (s *Service) buildConversationContext(
	ctx context.Context,
	req *serviceports.ShipmentImportChatRequest,
) []byte {
	data, _ := sonic.Marshal(s.buildConversationContextMap(ctx, req))
	return data
}

// systemMessage carries the instructions and the state of the import. The
// state is rebuilt every turn rather than replayed from the conversation,
// because a field the operator accepted two turns ago is current fact, and
// showing the model its own older view of it is how it contradicts itself.
func (s *Service) systemMessage(
	ctx context.Context,
	req *serviceports.ShipmentImportChatRequest,
) string {
	return systemPrompt + "\n\nCurrent state:\n" + string(s.buildConversationContext(ctx, req))
}

func (s *Service) buildDefaultOptions(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) map[string]any {
	serviceTypes := make([]map[string]string, 0)
	shipmentTypes := make([]map[string]string, 0)
	formulaTemplates := make([]map[string]string, 0)

	if result, err := s.serviceRepo.SelectOptions(
		ctx,
		&repositories.ServiceTypeSelectOptionsRequest{
			SelectQueryRequest: &pagination.SelectQueryRequest{
				TenantInfo: tenantInfo,
				Pagination: pagination.Info{Limit: 6},
			},
		}); err == nil {
		for _, item := range result.Items {
			serviceTypes = append(serviceTypes, map[string]string{
				"id":    item.ID.String(),
				"code":  item.Code,
				"name":  item.Description,
				"label": strings.TrimSpace(item.Code + " — " + item.Description),
			})
		}
	}

	if result, err := s.shipmentTypeRepo.SelectOptions(
		ctx,
		&repositories.ShipmentTypeSelectOptionsRequest{
			SelectQueryRequest: &pagination.SelectQueryRequest{
				TenantInfo: tenantInfo,
				Pagination: pagination.Info{Limit: 6},
			},
		}); err == nil {
		for _, item := range result.Items {
			shipmentTypes = append(shipmentTypes, map[string]string{
				"id":    item.ID.String(),
				"code":  item.Code,
				"name":  item.Description,
				"label": strings.TrimSpace(item.Code + " — " + item.Description),
			})
		}
	}

	if result, err := s.formulaTemplateRepo.SelectOptions(
		ctx,
		&repositories.FormulaTemplateSelectOptionsRequest{
			SelectQueryRequest: &pagination.SelectQueryRequest{
				TenantInfo: tenantInfo,
				Pagination: pagination.Info{Limit: 6},
			},
		}); err == nil {
		for _, item := range result.Items {
			formulaTemplates = append(formulaTemplates, map[string]string{
				"id":   item.ID.String(),
				"name": item.Name,
			})
		}
	}

	return map[string]any{
		"serviceTypes":     serviceTypes,
		"shipmentTypes":    shipmentTypes,
		"formulaTemplates": formulaTemplates,
	}
}

func (s *Service) ensureConversation(
	ctx context.Context,
	req *serviceports.ShipmentImportChatRequest,
) (*shipmentimportchat.Conversation, error) {
	documentID, err := pulid.Parse(req.DocumentID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"documentId",
			errortypes.ErrInvalid,
			"Invalid document ID",
		)
	}

	conversation, err := s.chatRepo.GetConversationByDocument(
		ctx,
		repositories.GetShipmentImportConversationRequest{
			DocumentID: documentID,
			TenantInfo: req.TenantInfo,
			Status:     shipmentimportchat.ConversationStatusActive,
		},
	)
	if err == nil {
		if req.ConversationID == "" && conversation.ExternalConversationID != "" {
			req.ConversationID = conversation.ExternalConversationID
		}
		return conversation, nil
	}
	if !errortypes.IsNotFoundError(err) {
		return nil, err
	}

	req.ConversationID = ""

	return s.chatRepo.CreateConversation(ctx, &shipmentimportchat.Conversation{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		DocumentID:     documentID,
		UserID:         req.TenantInfo.UserID,
		Status:         shipmentimportchat.ConversationStatusActive,
	})
}

// NormalizeSuggestions cleans the chips a model suggested for display.
func NormalizeSuggestions(
	suggestions []serviceports.ShipmentImportSuggestion,
) []serviceports.ShipmentImportSuggestion {
	normalized := make([]serviceports.ShipmentImportSuggestion, 0, len(suggestions))
	for _, suggestion := range suggestions {
		if suggestion.Type == "" {
			suggestion.Type = "prompt"
		}
		if suggestion.Type == "input" && suggestion.SubmitLabel == "" {
			if strings.Contains(strings.ToLower(suggestion.Prompt), "search") {
				suggestion.SubmitLabel = "Search"
			} else {
				suggestion.SubmitLabel = "Confirm"
			}
		}
		if suggestion.Type == "date" && suggestion.SubmitLabel == "" {
			suggestion.SubmitLabel = "Confirm"
		}
		normalized = append(normalized, suggestion)
	}

	return normalized
}

func toHistoryMessages(
	messages []shipmentimportchat.HistoryMessage,
) []serviceports.ShipmentImportChatMessage {
	converted := make([]serviceports.ShipmentImportChatMessage, 0, len(messages))
	for _, message := range messages {
		toolCalls := make([]serviceports.ShipmentImportToolCallRecord, 0, len(message.ToolCalls))
		for _, toolCall := range message.ToolCalls {
			toolCalls = append(toolCalls, serviceports.ShipmentImportToolCallRecord{
				Name:   toolCall.Name,
				CallID: toolCall.CallID,
				Status: toolCall.Status,
				Input:  toolCall.Input,
				Output: toolCall.Output,
			})
		}

		suggestions := make([]serviceports.ShipmentImportSuggestion, 0, len(message.Suggestions))
		for _, suggestion := range message.Suggestions {
			suggestions = append(suggestions, serviceports.ShipmentImportSuggestion{
				Label:       suggestion.Label,
				Prompt:      suggestion.Prompt,
				Type:        suggestion.Type,
				Placeholder: suggestion.Placeholder,
				Action:      suggestion.Action,
				SubmitLabel: suggestion.SubmitLabel,
			})
		}

		converted = append(converted, serviceports.ShipmentImportChatMessage{
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

func toHistorySuggestions(
	suggestions []serviceports.ShipmentImportSuggestion,
) []shipmentimportchat.HistorySuggestion {
	converted := make([]shipmentimportchat.HistorySuggestion, 0, len(suggestions))
	for _, suggestion := range suggestions {
		converted = append(converted, shipmentimportchat.HistorySuggestion{
			Label:       suggestion.Label,
			Prompt:      suggestion.Prompt,
			Type:        suggestion.Type,
			Placeholder: suggestion.Placeholder,
			Action:      suggestion.Action,
			SubmitLabel: suggestion.SubmitLabel,
		})
	}

	return converted
}

func toHistoryToolCalls(
	toolCalls []serviceports.ShipmentImportToolCallRecord,
) []shipmentimportchat.HistoryToolCall {
	converted := make([]shipmentimportchat.HistoryToolCall, 0, len(toolCalls))
	for _, toolCall := range toolCalls {
		converted = append(converted, shipmentimportchat.HistoryToolCall{
			Name:   toolCall.Name,
			CallID: toolCall.CallID,
			Status: toolCall.Status,
			Input:  toolCall.Input,
			Output: toolCall.Output,
		})
	}

	return converted
}

func toHistoryActions(
	actions []serviceports.ShipmentImportAction,
) []shipmentimportchat.HistoryAction {
	converted := make([]shipmentimportchat.HistoryAction, 0, len(actions))
	for _, action := range actions {
		converted = append(converted, shipmentimportchat.HistoryAction{
			Type:     action.Type,
			FieldKey: action.FieldKey,
			Value:    action.Value,
			Metadata: action.Metadata,
		})
	}

	return converted
}

func (s *Service) persistConversationTurn(
	ctx context.Context,
	req *serviceports.ShipmentImportChatRequest,
	conversation *shipmentimportchat.Conversation,
	responseConversationID string,
	assistantMessage string,
	suggestions []serviceports.ShipmentImportSuggestion,
	toolCalls []serviceports.ShipmentImportToolCallRecord,
	actions []serviceports.ShipmentImportAction,
	model string,
	resultStatus shipmentimportchat.TurnResultStatus,
	errorMessage string,
) error {
	if conversation == nil {
		return nil
	}

	now := timeutils.NowUnix()
	encodedPayload := shipmentimportchat.TurnPayload{
		Context:     s.buildConversationContextMap(ctx, req),
		Suggestions: toHistorySuggestions(suggestions),
		ToolCalls:   toHistoryToolCalls(toolCalls),
		Actions:     toHistoryActions(actions),
	}.Encode()

	if err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		conversation.UserID = req.TenantInfo.UserID
		conversation.ExternalConversationID = responseConversationID
		conversation.TurnCount++
		conversation.LastMessageAt = &now

		if _, err := s.chatRepo.UpdateConversation(txCtx, conversation); err != nil {
			return err
		}

		_, err := s.chatRepo.AppendTurn(txCtx, &shipmentimportchat.Turn{
			ConversationID:         conversation.ID,
			OrganizationID:         req.TenantInfo.OrgID,
			BusinessUnitID:         req.TenantInfo.BuID,
			DocumentID:             conversation.DocumentID,
			UserID:                 req.TenantInfo.UserID,
			TurnIndex:              conversation.TurnCount,
			UserMessage:            req.UserMessage,
			AssistantMessage:       assistantMessage,
			RequestConversationID:  req.ConversationID,
			ResponseConversationID: responseConversationID,
			Model:                  model,
			ResultStatus:           resultStatus,
			ErrorMessage:           errorMessage,
			ContextJSON:            encodedPayload.ContextJSON,
			SuggestionsJSON:        encodedPayload.SuggestionsJSON,
			ToolCallsJSON:          encodedPayload.ToolCallsJSON,
			ActionsJSON:            encodedPayload.ActionsJSON,
		})
		return err
	}); err != nil {
		return err
	}

	if s.chatCacheRepo != nil {
		if cacheErr := s.chatCacheRepo.DeleteHistory(
			ctx,
			conversation.DocumentID,
			req.TenantInfo,
		); cacheErr != nil {
			s.logger.Warn("failed to invalidate shipment import chat cache", zap.Error(cacheErr))
		}
	}

	snapshot, historyErr := s.getHistorySnapshot(ctx, conversation.DocumentID, req.TenantInfo)
	if historyErr == nil && snapshot != nil && s.chatCacheRepo != nil {
		if cacheErr := s.chatCacheRepo.SetHistory(ctx, snapshot, req.TenantInfo); cacheErr != nil {
			s.logger.Warn("failed to cache shipment import chat history", zap.Error(cacheErr))
		}
	}

	return nil
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

	conversation, err := s.chatRepo.GetConversationByDocument(
		ctx,
		repositories.GetShipmentImportConversationRequest{
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

	turns, err := s.chatRepo.ListTurns(ctx, repositories.ListShipmentImportTurnsRequest{
		ConversationID: conversation.ID,
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
		ConversationID: conversation.ExternalConversationID,
		Status:         conversation.Status,
		StatusReason:   conversation.StatusReason,
		TurnCount:      conversation.TurnCount,
		LastMessageAt:  conversation.LastMessageAt,
		Messages:       messages,
		UpdatedAt:      conversation.UpdatedAt,
	}

	if s.chatCacheRepo != nil {
		if cacheErr := s.chatCacheRepo.SetHistory(ctx, snapshot, tenantInfo); cacheErr != nil {
			s.logger.Warn("failed to cache shipment import chat history", zap.Error(cacheErr))
		}
	}

	return snapshot, nil
}

func (s *Service) recordFailedTurn(
	ctx context.Context,
	req *serviceports.ShipmentImportChatRequest,
	conversation *shipmentimportchat.Conversation,
	conversationID string,
	assistantMessage string,
	toolCalls []serviceports.ShipmentImportToolCallRecord,
	actions []serviceports.ShipmentImportAction,
	errorMessage string,
) {
	if conversation == nil {
		return
	}

	if err := s.persistConversationTurn(
		ctx,
		req,
		conversation,
		conversationID,
		assistantMessage,
		nil,
		toolCalls,
		actions,
		"",
		shipmentimportchat.TurnResultStatusFailed,
		errorMessage,
	); err != nil {
		s.logger.Warn("failed to persist shipment import assistant error turn", zap.Error(err))
	}
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

func toolCallStatusFromResult(result string) string {
	if result == "" || result[0] != '{' {
		return toolStatusCompleted
	}

	var check map[string]any
	if json.Unmarshal([]byte(result), &check) != nil {
		return toolStatusCompleted
	}
	if _, hasErr := check["error"]; hasErr {
		return toolStatusError
	}

	return toolStatusCompleted
}

type shipmentImportToolCallArgs struct {
	m map[string]any
}

func (a shipmentImportToolCallArgs) Str(key string) string {
	v, _ := a.m[key].(string)
	return v
}

func (a shipmentImportToolCallArgs) IntFromFloat64(key string) (int, bool) {
	v, ok := a.m[key].(float64)
	if !ok {
		return 0, false
	}
	return int(v), true
}

type shipmentImportToolCallHandler func(
	s *Service,
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	args shipmentImportToolCallArgs,
) (string, []serviceports.ShipmentImportAction)

var shipmentImportToolCallHandlers = map[string]shipmentImportToolCallHandler{
	"accept_field": func(
		_ *Service,
		_ context.Context,
		_ pagination.TenantInfo,
		args shipmentImportToolCallArgs,
	) (string, []serviceports.ShipmentImportAction) {
		key := args.Str("field_key")
		return fmt.Sprintf(
				`{"accepted":"%q"}`,
				key,
			), []serviceports.ShipmentImportAction{
				{Type: "accept_field", FieldKey: key},
			}
	},

	"accept_all_confident": func(
		_ *Service,
		_ context.Context,
		_ pagination.TenantInfo,
		_ shipmentImportToolCallArgs,
	) (string, []serviceports.ShipmentImportAction) {
		return `{"accepted":"all_confident"}`, []serviceports.ShipmentImportAction{
			{Type: "accept_all_confident"},
		}
	},

	"set_field_value": func(
		_ *Service,
		_ context.Context,
		_ pagination.TenantInfo,
		args shipmentImportToolCallArgs,
	) (string, []serviceports.ShipmentImportAction) {
		key, value := args.Str("field_key"), args.Str("value")
		return fmt.Sprintf(
				`{"set":"%q","value":"%q"}`,
				key,
				value,
			), []serviceports.ShipmentImportAction{
				{Type: "set_field", FieldKey: key, Value: value},
			}
	},

	"set_required_field": func(
		_ *Service,
		_ context.Context,
		_ pagination.TenantInfo,
		args shipmentImportToolCallArgs,
	) (string, []serviceports.ShipmentImportAction) {
		key, entityID, label := args.Str("field_key"), args.Str("entity_id"), args.Str("label")
		return fmt.Sprintf(
				`{"set_required":"%q","entity_id":"%q"}`,
				key,
				entityID,
			), []serviceports.ShipmentImportAction{
				{
					Type:     "set_required_field",
					FieldKey: key,
					Value:    entityID,
					Metadata: map[string]any{"label": label},
				},
			}
	},

	"search_customers": func(
		s *Service,
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		args shipmentImportToolCallArgs,
	) (string, []serviceports.ShipmentImportAction) {
		return s.searchCustomers(ctx, tenantInfo, args.Str("query")), nil
	},
	"search_locations": func(
		s *Service,
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		args shipmentImportToolCallArgs,
	) (string, []serviceports.ShipmentImportAction) {
		return s.searchLocations(ctx, tenantInfo, args.Str("query")), nil
	},
	"search_service_types": func(
		s *Service,
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		args shipmentImportToolCallArgs,
	) (string, []serviceports.ShipmentImportAction) {
		return s.searchServiceTypes(ctx, tenantInfo, args.Str("query")), nil
	},
	"search_shipment_types": func(
		s *Service,
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		args shipmentImportToolCallArgs,
	) (string, []serviceports.ShipmentImportAction) {
		return s.searchShipmentTypes(ctx, tenantInfo, args.Str("query")), nil
	},
	"search_formula_templates": func(
		s *Service,
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		args shipmentImportToolCallArgs,
	) (string, []serviceports.ShipmentImportAction) {
		return s.searchFormulaTemplates(ctx, tenantInfo, args.Str("query")), nil
	},

	"set_stop_location": func(
		_ *Service,
		_ context.Context,
		_ pagination.TenantInfo,
		args shipmentImportToolCallArgs,
	) (string, []serviceports.ShipmentImportAction) {
		idx, _ := args.IntFromFloat64("stop_index")
		locID := args.Str("location_id")
		return fmt.Sprintf(
				`{"set_stop_location":true,"stop_index":%d,"location_id":"%q"}`,
				idx,
				locID,
			),
			[]serviceports.ShipmentImportAction{
				{Type: "set_stop_location", FieldKey: strconv.Itoa(idx), Value: locID},
			}
	},

	"set_stop_schedule": func(
		_ *Service,
		_ context.Context,
		_ pagination.TenantInfo,
		args shipmentImportToolCallArgs,
	) (string, []serviceports.ShipmentImportAction) {
		idx, _ := args.IntFromFloat64("stop_index")
		start := args.Str("window_start")
		end := args.Str("window_end")

		unixStart := parseShipmentImportAssistantTimeToUnix(start)

		metadata := map[string]any{"window_start_iso": start}
		var unixEnd int64
		if end != "" {
			unixEnd = parseShipmentImportAssistantTimeToUnix(end)
			if unixEnd > 0 {
				metadata["window_end"] = strconv.Itoa(int(unixEnd))
				metadata["window_end_iso"] = end
			}
		}

		result := fmt.Sprintf(
			`{"set_stop_schedule":true,"stop_index":%d,"unix_start":%d,"unix_end":%d}`,
			idx,
			unixStart,
			unixEnd,
		)
		return result, []serviceports.ShipmentImportAction{
			{
				Type:     "set_stop_schedule",
				FieldKey: strconv.Itoa(idx),
				Value:    strconv.FormatInt(unixStart, 10),
				Metadata: metadata,
			},
		}
	},

	"set_shipment_field": func(
		_ *Service,
		_ context.Context,
		_ pagination.TenantInfo,
		args shipmentImportToolCallArgs,
	) (string, []serviceports.ShipmentImportAction) {
		field := args.Str("field")
		value := args.Str("value")
		return fmt.Sprintf(`{"set_shipment_field":true,"field":"%q","value":"%q"}`, field, value),
			[]serviceports.ShipmentImportAction{
				{Type: "set_shipment_field", FieldKey: field, Value: value},
			}
	},

	"get_customer_requirements": func(
		s *Service,
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		args shipmentImportToolCallArgs,
	) (string, []serviceports.ShipmentImportAction) {
		return s.getCustomerRequirements(ctx, tenantInfo, args.Str("customer_id")), nil
	},

	"get_shipment_control": func(
		s *Service,
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		_ shipmentImportToolCallArgs,
	) (string, []serviceports.ShipmentImportAction) {
		return s.getShipmentControl(ctx, tenantInfo), nil
	},

	"add_location": func(
		s *Service,
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		args shipmentImportToolCallArgs,
	) (string, []serviceports.ShipmentImportAction) {
		return s.addLocation(ctx, tenantInfo, args.m), nil
	},

	"create_shipment": func(
		_ *Service,
		_ context.Context,
		_ pagination.TenantInfo,
		_ shipmentImportToolCallArgs,
	) (string, []serviceports.ShipmentImportAction) {
		return `{"create_shipment":true}`, []serviceports.ShipmentImportAction{
			{Type: "create_shipment"},
		}
	},
}

func parseShipmentImportAssistantTimeToUnix(s string) int64 {
	layouts := [...]string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02",
	}
	for i := range layouts {
		t, err := time.Parse(layouts[i], s)
		if err == nil {
			return t.Unix()
		}
	}
	return 0
}

type entityMatch struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (s *Service) queryCustomers(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	query string,
	limit int,
) ([]entityMatch, int, error) {
	result, err := s.customerRepo.SelectOptions(ctx, &repositories.CustomerSelectOptionsRequest{
		SelectQueryRequest: &pagination.SelectQueryRequest{
			TenantInfo: tenantInfo,
			Pagination: pagination.Info{Limit: limit},
			Query:      query,
		},
	})
	if err != nil {
		return nil, 0, err
	}

	matches := make([]entityMatch, 0, len(result.Items))
	for _, c := range result.Items {
		matches = append(matches, entityMatch{ID: c.ID.String(), Name: c.Name})
	}

	return matches, result.Total, nil
}

func (s *Service) searchCustomers(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	query string,
) string {
	matches, total, err := s.queryCustomers(ctx, tenantInfo, query, 5)
	if err != nil {
		return `{"error":"search failed"}` //nolint:goconst // local literal
	}

	if len(matches) > 0 || query == "" {
		data, _ := json.Marshal(map[string]any{"customers": matches, "total": total})
		return string(data)
	}

	// No exact match — fallback to showing all available customers
	available, availTotal, fallbackErr := s.queryCustomers(ctx, tenantInfo, "", 10)
	if fallbackErr != nil || len(available) == 0 {
		data, _ := json.Marshal(map[string]any{"customers": matches, "total": total})
		return string(data)
	}

	data, _ := json.Marshal(map[string]any{
		"customers":          matches,
		"total":              total,
		"noExactMatch":       true,
		"searchQuery":        query,
		"availableCustomers": available,
		"availableTotal":     availTotal,
	})
	return string(data)
}

type locationMatch struct {
	ID           string `json:"id"`
	Code         string `json:"code"`
	Name         string `json:"name"`
	AddressLine1 string `json:"addressLine1,omitempty"`
	City         string `json:"city,omitempty"`
	PostalCode   string `json:"postalCode,omitempty"`
}

func (s *Service) queryLocations(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	query string,
	limit int,
) ([]locationMatch, int, error) {
	result, err := s.locationRepo.SelectOptions(ctx, &repositories.LocationSelectOptionsRequest{
		SelectQueryRequest: &pagination.SelectQueryRequest{
			TenantInfo: tenantInfo,
			Pagination: pagination.Info{Limit: limit},
			Query:      query,
		},
	})
	if err != nil {
		return nil, 0, err
	}

	matches := make([]locationMatch, 0, len(result.Items))
	for _, l := range result.Items {
		matches = append(matches, locationMatch{
			ID:           l.ID.String(),
			Code:         l.Code,
			Name:         l.Name,
			AddressLine1: l.AddressLine1,
			City:         l.City,
			PostalCode:   l.PostalCode,
		})
	}
	return matches, result.Total, nil
}

func (s *Service) searchLocations(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	query string,
) string {
	matches, total, err := s.queryLocations(ctx, tenantInfo, query, 5)
	if err != nil {
		return `{"error":"search failed"}`
	}

	if len(matches) > 0 || query == "" {
		data, _ := json.Marshal(map[string]any{"locations": matches, "total": total})
		return string(data)
	}

	available, availTotal, fallbackErr := s.queryLocations(ctx, tenantInfo, "", 10)
	if fallbackErr != nil || len(available) == 0 {
		data, _ := json.Marshal(map[string]any{"locations": matches, "total": total})
		return string(data)
	}

	data, _ := json.Marshal(map[string]any{
		"locations":          matches,
		"total":              total,
		"noExactMatch":       true,
		"searchQuery":        query,
		"availableLocations": available,
		"availableTotal":     availTotal,
	})
	return string(data)
}

func (s *Service) searchServiceTypes(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	query string,
) string {
	result, err := s.serviceRepo.SelectOptions(ctx, &repositories.ServiceTypeSelectOptionsRequest{
		SelectQueryRequest: &pagination.SelectQueryRequest{
			TenantInfo: tenantInfo,
			Pagination: pagination.Info{Limit: 5},
			Query:      query,
		},
	})
	if err != nil {
		return `{"error":"search failed"}`
	}

	type m struct {
		ID   string `json:"id"`
		Code string `json:"code"`
		Name string `json:"name"`
	}
	matches := make([]m, 0, len(result.Items))
	for _, st := range result.Items {
		matches = append(matches, m{ID: st.ID.String(), Code: st.Code, Name: st.Description})
	}
	data, _ := json.Marshal(map[string]any{"serviceTypes": matches, "total": result.Total})
	return string(data)
}

func (s *Service) searchShipmentTypes(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	query string,
) string {
	result, err := s.shipmentTypeRepo.SelectOptions(
		ctx,
		&repositories.ShipmentTypeSelectOptionsRequest{
			SelectQueryRequest: &pagination.SelectQueryRequest{
				TenantInfo: tenantInfo,
				Pagination: pagination.Info{Limit: 5},
				Query:      query,
			},
		},
	)
	if err != nil {
		return `{"error":"search failed"}`
	}

	matches := make([]entityMatch, 0, len(result.Items))
	for _, st := range result.Items {
		matches = append(
			matches,
			entityMatch{ID: st.ID.String(), Name: st.Code + " — " + st.Description},
		)
	}
	data, _ := json.Marshal(map[string]any{"shipmentTypes": matches, "total": result.Total})
	return string(data)
}

func (s *Service) searchFormulaTemplates(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	query string,
) string {
	result, err := s.formulaTemplateRepo.SelectOptions(
		ctx,
		&repositories.FormulaTemplateSelectOptionsRequest{
			SelectQueryRequest: &pagination.SelectQueryRequest{
				TenantInfo: tenantInfo,
				Pagination: pagination.Info{Limit: 5},
				Query:      query,
			},
		},
	)
	if err != nil {
		return `{"error":"search failed"}`
	}

	matches := make([]entityMatch, 0, len(result.Items))
	for _, ft := range result.Items {
		matches = append(matches, entityMatch{ID: ft.ID.String(), Name: ft.Name})
	}
	data, _ := json.Marshal(map[string]any{"formulaTemplates": matches, "total": result.Total})
	return string(data)
}

func (s *Service) addLocation(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	args map[string]any,
) string {
	str := func(key string) string { v, _ := args[key].(string); return v }

	name := str("name")
	addr := str("address_line1")
	city := str("city")
	stateAbbrev := str("state_abbrev")
	postalCode := str("postal_code")

	if name == "" || addr == "" || city == "" || stateAbbrev == "" || postalCode == "" {
		return `{"error":"missing required fields for location"}`
	}

	// Look up the state ID from abbreviation
	state, stateErr := s.usStateRepo.GetByAbbreviation(ctx, stateAbbrev)
	if stateErr != nil || state == nil {
		return fmt.Sprintf(`{"error":"could not find US state for abbreviation '%s'"}`, stateAbbrev)
	}

	// Get the first available location category
	catResult, catErr := s.locationCategoryRepo.SelectOptions(ctx, &pagination.SelectQueryRequest{
		TenantInfo: tenantInfo,
		Pagination: pagination.Info{Limit: 1},
		Query:      "",
	})
	if catErr != nil || len(catResult.Items) == 0 {
		return `{"error":"no location categories found — please create one in Settings first"}`
	}
	locationCategoryID := catResult.Items[0].ID

	entity := &location.Location{
		OrganizationID:     tenantInfo.OrgID,
		BusinessUnitID:     tenantInfo.BuID,
		LocationCategoryID: locationCategoryID,
		StateID:            state.ID,
		Status:             domaintypes.StatusActive,
		Code:               stringutils.TruncateRunes(name, maxLocationCodeLength),
		Name:               name,
		AddressLine1:       addr,
		City:               city,
		PostalCode:         postalCode,
	}

	created, createErr := s.locationService.Create(ctx, entity, actingUser(tenantInfo))
	if createErr != nil {
		s.logger.Error("failed to create location", zap.Error(createErr))
		return fmt.Sprintf(`{"error":"failed to create location: %s"}`, createErr.Error())
	}

	data, _ := json.Marshal(map[string]any{
		"id":           created.ID.String(),
		"code":         created.Code,
		"name":         created.Name,
		"addressLine1": created.AddressLine1,
		"city":         created.City,
		"postalCode":   created.PostalCode,
	})
	return string(data)
}

func (s *Service) getCustomerRequirements(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	customerID string,
) string {
	id, parseErr := pulid.Parse(customerID)
	if parseErr != nil {
		return `{"error":"invalid customer ID"}`
	}

	customer, err := s.customerRepo.GetByID(ctx, repositories.GetCustomerByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
		CustomerFilterOptions: repositories.CustomerFilterOptions{
			IncludeBillingProfile: true,
		},
	})
	if err != nil {
		return `{"error":"customer not found"}`
	}

	requireBOL := false
	if customer.BillingProfile != nil {
		requireBOL = customer.BillingProfile.RequireBOLNumber
	}

	data, _ := json.Marshal(map[string]any{
		"customerId":       customer.ID.String(),
		"name":             customer.Name,
		"code":             customer.Code,
		"requireBOLNumber": requireBOL,
	})
	return string(data)
}

func (s *Service) getShipmentControl(ctx context.Context, tenantInfo pagination.TenantInfo) string {
	control, err := s.shipmentControlRepo.Get(
		ctx,
		repositories.GetShipmentControlRequest{TenantInfo: tenantInfo},
	)
	if err != nil {
		return `{"error":"could not fetch shipment control"}`
	}

	data, _ := json.Marshal(map[string]any{
		"maxShipmentWeightLimit": control.MaxShipmentWeightLimit,
		"checkForDuplicateBols":  control.CheckForDuplicateBOLs,
		"checkHazmatSegregation": control.CheckHazmatSegregation,
		"trackDetentionTime":     control.TrackDetentionTime,
	})
	return string(data)
}

func (s *Service) GetHistory(
	ctx context.Context,
	documentID string,
	tenantInfo pagination.TenantInfo,
) (*serviceports.ShipmentImportChatHistoryResponse, error) {
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

	return &serviceports.ShipmentImportChatHistoryResponse{
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
		return errortypes.NewValidationError("documentId", errortypes.ErrInvalid, "Invalid document ID")
	}

	closed, err := s.pageThreads.ArchiveSubjectThreads(ctx, repositories.ArchiveSubjectThreadsRequest{
		TenantInfo:  tenantInfo,
		Origin:      conversation.ThreadOriginImport,
		SubjectType: agent.SubjectDocument,
		SubjectID:   id,
	})
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

var _ serviceports.ShipmentImportAssistantService = (*Service)(nil)
