package shipmentimportassistantservice

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json" //nolint:depguard // external API payloads
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/ailog"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/shipmentimportchat"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/integrationservice"
	"github.com/emoss08/trenova/internal/core/services/locationservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/responses"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger               *zap.Logger
	Config               *config.Config
	DB                   *postgres.Connection
	Integration          *integrationservice.Service
	AILogRepo            repositories.AILogRepository
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
}

type Service struct {
	logger               *zap.Logger
	cfg                  *config.DocumentIntelligenceConfig
	db                   *postgres.Connection
	integration          *integrationservice.Service
	aiLogRepo            repositories.AILogRepository
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
}

func New(p Params) serviceports.ShipmentImportAssistantService {
	return &Service{
		logger:               p.Logger.Named("service.shipment-import-assistant"),
		cfg:                  p.Config.GetDocumentIntelligenceConfig(),
		db:                   p.DB,
		integration:          p.Integration,
		aiLogRepo:            p.AILogRepo,
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

func buildTools() []responses.ToolUnionParam {
	return []responses.ToolUnionParam{
		{OfFunction: &responses.FunctionToolParam{
			Name:        "accept_field",
			Description: openai.String("Accept an extracted field value as correct"),
			Parameters: map[string]any{
				"type":                 "object",
				"properties":           map[string]any{"field_key": map[string]any{"type": "string"}},
				"required":             []string{"field_key"},
				"additionalProperties": false,
			},
		}},
		{OfFunction: &responses.FunctionToolParam{
			Name:        "accept_all_confident",
			Description: openai.String("Accept all high-confidence extracted fields at once"),
			Parameters: map[string]any{
				"type":                 "object",
				"properties":           map[string]any{},
				"additionalProperties": false,
			},
		}},
		{OfFunction: &responses.FunctionToolParam{
			Name:        "set_field_value",
			Description: openai.String("Set or override an extracted field value"),
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"field_key": map[string]any{"type": "string"},
					"value":     map[string]any{"type": "string"},
				},
				"required":             []string{"field_key", "value"},
				"additionalProperties": false,
			},
		}},
		{OfFunction: &responses.FunctionToolParam{
			Name:        "set_required_field",
			Description: openai.String("Set a required shipment field by entity ID after confirming with the user"),
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"field_key": map[string]any{"type": "string", "enum": []string{"customerId", "serviceTypeId", "shipmentTypeId", "formulaTemplateId"}},
					"entity_id": map[string]any{"type": "string"},
					"label":     map[string]any{"type": "string"},
				},
				"required":             []string{"field_key", "entity_id", "label"},
				"additionalProperties": false,
			},
		}},
		{OfFunction: &responses.FunctionToolParam{
			Name:        "search_customers",
			Description: openai.String("Search the customer database by name"),
			Parameters: map[string]any{
				"type":                 "object",
				"properties":           map[string]any{"query": map[string]any{"type": "string"}},
				"required":             []string{"query"},
				"additionalProperties": false,
			},
		}},
		{OfFunction: &responses.FunctionToolParam{
			Name:        "search_locations",
			Description: openai.String("Search the location database by name, city, or address"),
			Parameters: map[string]any{
				"type":                 "object",
				"properties":           map[string]any{"query": map[string]any{"type": "string"}},
				"required":             []string{"query"},
				"additionalProperties": false,
			},
		}},
		{OfFunction: &responses.FunctionToolParam{
			Name:        "search_service_types",
			Description: openai.String("Search available service types"),
			Parameters: map[string]any{
				"type":                 "object",
				"properties":           map[string]any{"query": map[string]any{"type": "string"}},
				"required":             []string{"query"},
				"additionalProperties": false,
			},
		}},
		{OfFunction: &responses.FunctionToolParam{
			Name:        "set_stop_location",
			Description: openai.String("Set a stop's location by matching to an existing location in the system"),
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"stop_index":  map[string]any{"type": "integer", "description": "0-based index of the stop"},
					"location_id": map[string]any{"type": "string", "description": "ID of the location to assign"},
				},
				"required":             []string{"stop_index", "location_id"},
				"additionalProperties": false,
			},
		}},
		{OfFunction: &responses.FunctionToolParam{
			Name:        "set_stop_schedule",
			Description: openai.String("Set a stop's scheduled pickup/delivery window. Provide ISO 8601 datetime strings."),
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"stop_index":   map[string]any{"type": "integer", "description": "0-based index of the stop"},
					"window_start": map[string]any{"type": "string", "description": "Start time as ISO 8601 (e.g. 2025-03-15T08:00:00Z)"},
					"window_end":   map[string]any{"type": "string", "description": "End time as ISO 8601 (optional)"},
				},
				"required":             []string{"stop_index", "window_start"},
				"additionalProperties": false,
			},
		}},
		{OfFunction: &responses.FunctionToolParam{
			Name:        "set_shipment_field",
			Description: openai.String("Set a top-level shipment field like bol, weight, pieces, freightChargeAmount"),
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"field": map[string]any{"type": "string", "description": "Field name (bol, weight, pieces, freightChargeAmount, proNumber)"},
					"value": map[string]any{"type": "string", "description": "Value to set"},
				},
				"required":             []string{"field", "value"},
				"additionalProperties": false,
			},
		}},
		{OfFunction: &responses.FunctionToolParam{
			Name:        "get_customer_requirements",
			Description: openai.String("Check if a customer requires BOL for invoicing. Call this after setting the customer."),
			Parameters: map[string]any{
				"type":                 "object",
				"properties":           map[string]any{"customer_id": map[string]any{"type": "string"}},
				"required":             []string{"customer_id"},
				"additionalProperties": false,
			},
		}},
		{OfFunction: &responses.FunctionToolParam{
			Name:        "get_shipment_control",
			Description: openai.String("Get the organization's shipment control settings (weight limits, BOL checking, etc.)"),
			Parameters: map[string]any{
				"type":                 "object",
				"properties":           map[string]any{},
				"additionalProperties": false,
			},
		}},
		{OfFunction: &responses.FunctionToolParam{
			Name:        "search_shipment_types",
			Description: openai.String("Search available shipment types"),
			Parameters: map[string]any{
				"type":                 "object",
				"properties":           map[string]any{"query": map[string]any{"type": "string"}},
				"required":             []string{"query"},
				"additionalProperties": false,
			},
		}},
		{OfFunction: &responses.FunctionToolParam{
			Name:        "search_formula_templates",
			Description: openai.String("Search available rating methods / formula templates"),
			Parameters: map[string]any{
				"type":                 "object",
				"properties":           map[string]any{"query": map[string]any{"type": "string"}},
				"required":             []string{"query"},
				"additionalProperties": false,
			},
		}},
		{OfFunction: &responses.FunctionToolParam{
			Name:        "add_location",
			Description: openai.String("Create a new location in the system from extracted address data. Use this when no matching location exists. The location will be created and its ID returned so you can assign it to a stop."),
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name":          map[string]any{"type": "string", "description": "Location name (e.g. company/facility name)"},
					"address_line1": map[string]any{"type": "string", "description": "Street address"},
					"city":          map[string]any{"type": "string", "description": "City name"},
					"state_abbrev":  map[string]any{"type": "string", "description": "Two-letter US state abbreviation (e.g. CA, TX, NY)"},
					"postal_code":   map[string]any{"type": "string", "description": "ZIP code"},
				},
				"required":             []string{"name", "address_line1", "city", "state_abbrev", "postal_code"},
				"additionalProperties": false,
			},
		}},
		{OfFunction: &responses.FunctionToolParam{
			Name:        "create_shipment",
			Description: openai.String("Create the shipment. Only call this when ALL required fields and stop locations are set. This triggers the actual shipment creation."),
			Parameters: map[string]any{
				"type":                 "object",
				"properties":           map[string]any{},
				"additionalProperties": false,
			},
		}},
		{OfFunction: &responses.FunctionToolParam{
			Name:        "suggest_quick_actions",
			Description: openai.String("Provide 2-3 action buttons. Call at the end of every response. type='prompt' for confirmations, type='input' when user needs to type a value, type='action' for triggering app actions like creating the shipment."),
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"suggestions": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"label":       map[string]any{"type": "string", "description": "Button label"},
								"prompt":      map[string]any{"type": "string", "description": "For type=prompt: message to send. For type=input: prefix before user's typed value (e.g. 'Search for customer ')"},
								"type":        map[string]any{"type": "string", "enum": []string{"prompt", "input", "action", "date"}, "description": "prompt = sends message. input = shows text field. action = triggers app action. date = shows a date+time picker."},
								"action":      map[string]any{"type": "string", "description": "For type=action: the action ID (e.g. 'create_shipment')"},
								"placeholder": map[string]any{"type": "string", "description": "For type=input: placeholder text in the input field"},
								"submitLabel": map[string]any{"type": "string", "description": "For type=input: submit button label. Use 'Confirm' for values, 'Search' for queries. Default: 'Search'"},
							},
							"required":             []string{"label", "prompt", "type"},
							"additionalProperties": false,
						},
						"maxItems": 3,
					},
				},
				"required":             []string{"suggestions"},
				"additionalProperties": false,
			},
		}},
	}
}

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

	control, controlErr := s.shipmentControlRepo.Get(ctx, repositories.GetShipmentControlRequest{TenantInfo: req.TenantInfo})
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
				"weight":            fmt.Sprintf("optional, max %d lbs", control.MaxShipmentWeightLimit),
				"bol":               "required if customer billing profile requireBOLNumber=true",
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

func (s *Service) buildConversationContext(ctx context.Context, req *serviceports.ShipmentImportChatRequest) []byte {
	data, _ := json.Marshal(s.buildConversationContextMap(ctx, req))
	return data
}

func (s *Service) buildDefaultOptions(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) map[string]any {
	serviceTypes := make([]map[string]string, 0)
	shipmentTypes := make([]map[string]string, 0)
	formulaTemplates := make([]map[string]string, 0)

	if result, err := s.serviceRepo.SelectOptions(ctx, &repositories.ServiceTypeSelectOptionsRequest{
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

	if result, err := s.shipmentTypeRepo.SelectOptions(ctx, &repositories.ShipmentTypeSelectOptionsRequest{
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

	if result, err := s.formulaTemplateRepo.SelectOptions(ctx, &repositories.FormulaTemplateSelectOptionsRequest{
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

func (s *Service) buildConversationInput(
	req *serviceports.ShipmentImportChatRequest,
	contextJSON []byte,
) responses.ResponseNewParamsInputUnion {
	items := []responses.ResponseInputItemUnionParam{
		{OfMessage: &responses.EasyInputMessageParam{
			Role: "system",
			Content: responses.EasyInputMessageContentUnionParam{
				OfString: openai.String(systemPrompt + "\n\nCurrent state:\n" + string(contextJSON)),
			},
		}},
		{OfMessage: &responses.EasyInputMessageParam{
			Role:    "user",
			Content: responses.EasyInputMessageContentUnionParam{OfString: openai.String(req.UserMessage)},
		}},
	}

	return responses.ResponseNewParamsInputUnion{OfInputItemList: items}
}

func (s *Service) ensureConversation(
	ctx context.Context,
	req *serviceports.ShipmentImportChatRequest,
) (*shipmentimportchat.Conversation, error) {
	documentID, err := pulid.Parse(req.DocumentID)
	if err != nil {
		return nil, errortypes.NewValidationError("documentId", errortypes.ErrInvalid, "Invalid document ID")
	}

	conversation, err := s.chatRepo.GetConversationByDocument(ctx, repositories.GetShipmentImportConversationRequest{
		DocumentID: documentID,
		TenantInfo: req.TenantInfo,
		Status:     shipmentimportchat.ConversationStatusActive,
	})
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

func friendlyStreamError(err error) string {
	var apiErr *openai.Error
	if errors.As(err, &apiErr) {
		switch apiErr.StatusCode {
		case 429:
			return "The AI provider is temporarily rate-limited. Please wait a few seconds and try again."
		case 401, 403:
			return "AI authentication failed. Please check the API key configuration."
		case 400:
			return "The AI request was invalid. Try starting a new conversation."
		default:
			if apiErr.StatusCode >= 500 {
				return "The AI provider is experiencing issues. Please try again shortly."
			}
		}
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return "The AI request timed out. Please try again."
	}
	if errors.Is(err, context.Canceled) {
		return "The request was canceled."
	}

	// Mid-stream errors aren't typed — fall back to string matching on the raw message.
	msg := err.Error()
	if strings.Contains(msg, "rate_limit") {
		return "The AI provider is temporarily rate-limited. Please wait a few seconds and try again."
	}
	if strings.Contains(msg, "context_length_exceeded") {
		return "The conversation is too long for the AI to process. Try starting a new import."
	}

	return "AI assistant encountered an error. Please try again."
}

func normalizeSuggestions(
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
			Model:                  string(openai.ChatModelGPT5_4),
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
		if cacheErr := s.chatCacheRepo.DeleteHistory(ctx, conversation.DocumentID, req.TenantInfo); cacheErr != nil {
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
			s.logger.Warn("failed to read shipment import chat cache; rebuilding from postgres", zap.Error(err))
		}
	}

	conversation, err := s.chatRepo.GetConversationByDocument(ctx, repositories.GetShipmentImportConversationRequest{
		DocumentID: documentID,
		TenantInfo: tenantInfo,
	})
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
		return errortypes.NewValidationError("documentId", errortypes.ErrInvalid, "Invalid document ID")
	}

	if err = s.chatRepo.UpdateActiveConversationStatusByDocument(ctx, id, tenantInfo, status, reason); err != nil {
		return err
	}

	if s.chatCacheRepo != nil {
		if cacheErr := s.chatCacheRepo.DeleteHistory(ctx, id, tenantInfo); cacheErr != nil {
			s.logger.Warn("failed to clear shipment import chat cache", zap.Error(cacheErr))
		}
	}

	return nil
}

func (s *Service) Chat(ctx context.Context, req *serviceports.ShipmentImportChatRequest) (*serviceports.ShipmentImportChatResponse, error) {
	conversation, err := s.ensureConversation(ctx, req)
	if err != nil {
		return nil, err
	}

	runtimeCfg, err := s.integration.GetRuntimeConfig(ctx, req.TenantInfo, integration.TypeOpenAI)
	if err != nil {
		s.recordFailedTurn(ctx, req, conversation, req.ConversationID, "", nil, nil, "OpenAI integration is not configured")
		return nil, errortypes.NewBusinessError("OpenAI integration is not configured")
	}

	apiKey := runtimeCfg.Config["apiKey"]
	if apiKey == "" {
		s.recordFailedTurn(ctx, req, conversation, req.ConversationID, "", nil, nil, "OpenAI API key is missing")
		return nil, errortypes.NewBusinessError("OpenAI API key is missing")
	}

	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithMaxRetries(5),
	)

	contextJSON := s.buildConversationContext(ctx, req)

	params := responses.ResponseNewParams{
		Model: openai.ChatModelGPT5_4,
		Input: s.buildConversationInput(req, contextJSON),
		Tools: buildTools(),
	}
	if req.ConversationID != "" {
		params.PreviousResponseID = openai.String(req.ConversationID)
	}

	var (
		actions        []serviceports.ShipmentImportAction
		suggestions    []serviceports.ShipmentImportSuggestion
		toolCallLog    []serviceports.ShipmentImportToolCallRecord
		finalText      string
		conversationID = req.ConversationID
	)

	for range 5 {
		resp, respErr := client.Responses.New(ctx, params)
		if respErr != nil {
			userMsg := friendlyStreamError(respErr)
			s.logger.Error("OpenAI API error", zap.Error(respErr))
			s.recordFailedTurn(ctx, req, conversation, conversationID, finalText, toolCallLog, actions, userMsg)
			return nil, errortypes.NewBusinessError(userMsg)
		}

		conversationID = resp.ID

		var toolOutputs []responses.ResponseInputItemUnionParam
		hasToolCalls := false

		for _, item := range resp.Output {
			switch item.Type {
			case "message":
				for _, content := range item.Content {
					if content.Type == "output_text" {
						finalText = content.Text
					}
				}
			case "function_call":
				hasToolCalls = true
				fc := item.AsFunctionCall()

				if fc.Name == "suggest_quick_actions" {
					// Parse suggestions — don't record as visible tool call
					var sugArgs struct {
						Suggestions []serviceports.ShipmentImportSuggestion `json:"suggestions"`
					}
					if err := json.Unmarshal([]byte(fc.Arguments), &sugArgs); err == nil {
						suggestions = sugArgs.Suggestions
					}
					toolOutputs = append(toolOutputs, responses.ResponseInputItemUnionParam{
						OfFunctionCallOutput: &responses.ResponseInputItemFunctionCallOutputParam{
							CallID: fc.CallID,
							Output: responses.ResponseInputItemFunctionCallOutputOutputUnionParam{
								OfString: openai.String(`{"ok":true}`),
							},
						},
					})
					continue
				}

				result, toolActions := s.executeToolCall(ctx, req.TenantInfo, fc.Name, fc.Arguments)
				actions = append(actions, toolActions...)

				// Record tool call for display
				status := "completed"
				if len(result) > 0 && result[0] == '{' {
					var check map[string]any
					if json.Unmarshal([]byte(result), &check) == nil {
						if _, hasErr := check["error"]; hasErr {
							status = "error"
						}
					}
				}
				toolCallLog = append(toolCallLog, serviceports.ShipmentImportToolCallRecord{
					Name:   fc.Name,
					CallID: fc.CallID,
					Status: status,
					Input:  fc.Arguments,
					Output: result,
				})

				toolOutputs = append(toolOutputs, responses.ResponseInputItemUnionParam{
					OfFunctionCallOutput: &responses.ResponseInputItemFunctionCallOutputParam{
						CallID: fc.CallID,
						Output: responses.ResponseInputItemFunctionCallOutputOutputUnionParam{
							OfString: openai.String(result),
						},
					},
				})
			}
		}

		if !hasToolCalls {
			break
		}

		params = responses.ResponseNewParams{
			Model:              openai.ChatModelGPT5_4,
			PreviousResponseID: openai.String(conversationID),
			Input:              responses.ResponseNewParamsInputUnion{OfInputItemList: toolOutputs},
			Tools:              buildTools(),
		}
	}

	suggestions = normalizeSuggestions(suggestions)
	if err = s.persistConversationTurn(
		ctx,
		req,
		conversation,
		conversationID,
		finalText,
		suggestions,
		toolCallLog,
		actions,
		shipmentimportchat.TurnResultStatusCompleted,
		"",
	); err != nil {
		return nil, err
	}

	s.logAICall(ctx, req, finalText)

	return &serviceports.ShipmentImportChatResponse{
		Message:        finalText,
		ConversationID: conversationID,
		Actions:        actions,
		Suggestions:    suggestions,
		ToolCalls:      toolCallLog,
	}, nil
}

func (s *Service) executeToolCall(ctx context.Context, tenantInfo pagination.TenantInfo, name, arguments string) (string, []serviceports.ShipmentImportAction) {
	var args map[string]any
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return `{"error":"invalid arguments"}`, nil
	}

	str := func(key string) string { v, _ := args[key].(string); return v }

	switch name {
	case "accept_field":
		key := str("field_key")
		return fmt.Sprintf(`{"accepted":"%s"}`, key), []serviceports.ShipmentImportAction{{Type: "accept_field", FieldKey: key}}

	case "accept_all_confident":
		return `{"accepted":"all_confident"}`, []serviceports.ShipmentImportAction{{Type: "accept_all_confident"}}

	case "set_field_value":
		key, value := str("field_key"), str("value")
		return fmt.Sprintf(`{"set":"%s","value":"%s"}`, key, value), []serviceports.ShipmentImportAction{{Type: "set_field", FieldKey: key, Value: value}}

	case "set_required_field":
		key, entityID, label := str("field_key"), str("entity_id"), str("label")
		return fmt.Sprintf(`{"set_required":"%s","entity_id":"%s"}`, key, entityID), []serviceports.ShipmentImportAction{
			{Type: "set_required_field", FieldKey: key, Value: entityID, Metadata: map[string]any{"label": label}},
		}

	case "search_customers":
		return s.searchCustomers(ctx, tenantInfo, str("query")), nil
	case "search_locations":
		return s.searchLocations(ctx, tenantInfo, str("query")), nil
	case "search_service_types":
		return s.searchServiceTypes(ctx, tenantInfo, str("query")), nil
	case "search_shipment_types":
		return s.searchShipmentTypes(ctx, tenantInfo, str("query")), nil
	case "search_formula_templates":
		return s.searchFormulaTemplates(ctx, tenantInfo, str("query")), nil

	case "set_stop_location":
		idx := int(args["stop_index"].(float64))
		locID := str("location_id")
		return fmt.Sprintf(`{"set_stop_location":true,"stop_index":%d,"location_id":"%s"}`, idx, locID),
			[]serviceports.ShipmentImportAction{{Type: "set_stop_location", FieldKey: fmt.Sprintf("%d", idx), Value: locID}}

	case "set_stop_schedule":
		idx := int(args["stop_index"].(float64))
		start := str("window_start")
		end := str("window_end")

		// Parse ISO 8601 to Unix timestamps
		var unixStart, unixEnd int64
		if t, parseErr := time.Parse(time.RFC3339, start); parseErr == nil {
			unixStart = t.Unix()
		} else if t, parseErr := time.Parse("2006-01-02T15:04:05", start); parseErr == nil {
			unixStart = t.Unix()
		} else if t, parseErr := time.Parse("2006-01-02", start); parseErr == nil {
			unixStart = t.Unix()
		}

		if end != "" {
			if t, parseErr := time.Parse(time.RFC3339, end); parseErr == nil {
				unixEnd = t.Unix()
			} else if t, parseErr := time.Parse("2006-01-02T15:04:05", end); parseErr == nil {
				unixEnd = t.Unix()
			}
		}

		metadata := map[string]any{"window_start_iso": start}
		if unixEnd > 0 {
			metadata["window_end"] = fmt.Sprintf("%d", unixEnd)
			metadata["window_end_iso"] = end
		}

		result := fmt.Sprintf(`{"set_stop_schedule":true,"stop_index":%d,"unix_start":%d,"unix_end":%d}`, idx, unixStart, unixEnd)
		return result, []serviceports.ShipmentImportAction{
			{Type: "set_stop_schedule", FieldKey: fmt.Sprintf("%d", idx), Value: fmt.Sprintf("%d", unixStart), Metadata: metadata},
		}

	case "set_shipment_field":
		field := str("field")
		value := str("value")
		return fmt.Sprintf(`{"set_shipment_field":true,"field":"%s","value":"%s"}`, field, value),
			[]serviceports.ShipmentImportAction{{Type: "set_shipment_field", FieldKey: field, Value: value}}

	case "get_customer_requirements":
		return s.getCustomerRequirements(ctx, tenantInfo, str("customer_id")), nil

	case "get_shipment_control":
		return s.getShipmentControl(ctx, tenantInfo), nil

	case "add_location":
		return s.addLocation(ctx, tenantInfo, args), nil

	case "create_shipment":
		return `{"create_shipment":true}`, []serviceports.ShipmentImportAction{{Type: "create_shipment"}}

	default:
		return `{"error":"unknown tool"}`, nil
	}
}

type entityMatch struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (s *Service) queryCustomers(ctx context.Context, tenantInfo pagination.TenantInfo, query string, limit int) ([]entityMatch, int, error) {
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

func (s *Service) searchCustomers(ctx context.Context, tenantInfo pagination.TenantInfo, query string) string {
	matches, total, err := s.queryCustomers(ctx, tenantInfo, query, 5)
	if err != nil {
		return `{"error":"search failed"}`
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

func (s *Service) queryLocations(ctx context.Context, tenantInfo pagination.TenantInfo, query string, limit int) ([]locationMatch, int, error) {
	result, err := s.locationRepo.SelectOptions(ctx, &repositories.LocationSelectOptionsRequest{
		SelectQueryRequest: &pagination.SelectQueryRequest{TenantInfo: tenantInfo, Pagination: pagination.Info{Limit: limit}, Query: query},
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

func (s *Service) searchLocations(ctx context.Context, tenantInfo pagination.TenantInfo, query string) string {
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

func (s *Service) searchServiceTypes(ctx context.Context, tenantInfo pagination.TenantInfo, query string) string {
	result, err := s.serviceRepo.SelectOptions(ctx, &repositories.ServiceTypeSelectOptionsRequest{
		SelectQueryRequest: &pagination.SelectQueryRequest{TenantInfo: tenantInfo, Pagination: pagination.Info{Limit: 5}, Query: query},
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

func (s *Service) searchShipmentTypes(ctx context.Context, tenantInfo pagination.TenantInfo, query string) string {
	result, err := s.shipmentTypeRepo.SelectOptions(ctx, &repositories.ShipmentTypeSelectOptionsRequest{
		SelectQueryRequest: &pagination.SelectQueryRequest{TenantInfo: tenantInfo, Pagination: pagination.Info{Limit: 5}, Query: query},
	})
	if err != nil {
		return `{"error":"search failed"}`
	}

	matches := make([]entityMatch, 0, len(result.Items))
	for _, st := range result.Items {
		matches = append(matches, entityMatch{ID: st.ID.String(), Name: st.Code + " — " + st.Description})
	}
	data, _ := json.Marshal(map[string]any{"shipmentTypes": matches, "total": result.Total})
	return string(data)
}

func (s *Service) searchFormulaTemplates(ctx context.Context, tenantInfo pagination.TenantInfo, query string) string {
	result, err := s.formulaTemplateRepo.SelectOptions(ctx, &repositories.FormulaTemplateSelectOptionsRequest{
		SelectQueryRequest: &pagination.SelectQueryRequest{TenantInfo: tenantInfo, Pagination: pagination.Info{Limit: 5}, Query: query},
	})
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

func (s *Service) addLocation(ctx context.Context, tenantInfo pagination.TenantInfo, args map[string]any) string {
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

	// Generate a short code from the name
	code := name
	if len(code) > 10 {
		code = code[:10]
	}

	entity := &location.Location{
		OrganizationID:     tenantInfo.OrgID,
		BusinessUnitID:     tenantInfo.BuID,
		LocationCategoryID: locationCategoryID,
		StateID:            state.ID,
		Status:             domaintypes.StatusActive,
		Code:               code,
		Name:               name,
		AddressLine1:       addr,
		City:               city,
		PostalCode:         postalCode,
	}

	actor := &serviceports.RequestActor{
		PrincipalType:  serviceports.PrincipalTypeUser,
		PrincipalID:    tenantInfo.UserID,
		UserID:         tenantInfo.UserID,
		BusinessUnitID: tenantInfo.BuID,
		OrganizationID: tenantInfo.OrgID,
	}

	created, createErr := s.locationService.Create(ctx, entity, actor)
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

func (s *Service) getCustomerRequirements(ctx context.Context, tenantInfo pagination.TenantInfo, customerID string) string {
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
	control, err := s.shipmentControlRepo.Get(ctx, repositories.GetShipmentControlRequest{TenantInfo: tenantInfo})
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

func (s *Service) logAICall(ctx context.Context, req *serviceports.ShipmentImportChatRequest, response string) {
	promptHash := sha256.Sum256([]byte(req.UserMessage))
	responseHash := sha256.Sum256([]byte(response))

	promptPreview := req.UserMessage
	if len(promptPreview) > 512 {
		promptPreview = promptPreview[:512]
	}
	responsePreview := response
	if len(responsePreview) > 1024 {
		responsePreview = responsePreview[:1024]
	}

	entry := &ailog.Log{
		ID:             pulid.MustNew("ail_"),
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		UserID:         req.TenantInfo.UserID,
		Prompt:         fmt.Sprintf("sha256=%s preview=%s", hex.EncodeToString(promptHash[:]), promptPreview),
		Response:       fmt.Sprintf("sha256=%s preview=%s", hex.EncodeToString(responseHash[:]), responsePreview),
		Model:          ailog.ModelGPT5Mini,
		Operation:      ailog.OperationShipmentImportChat,
		Object:         req.DocumentID,
		Timestamp:      timeutils.NowUnix(),
	}

	if _, err := s.aiLogRepo.Create(ctx, entry); err != nil {
		s.logger.Error("failed to log AI call", zap.Error(err))
	}
}

func (s *Service) ChatStream(ctx context.Context, req *serviceports.ShipmentImportChatRequest, emit func(serviceports.StreamEvent)) error {
	conversation, err := s.ensureConversation(ctx, req)
	if err != nil {
		return err
	}

	runtimeCfg, err := s.integration.GetRuntimeConfig(ctx, req.TenantInfo, integration.TypeOpenAI)
	if err != nil {
		s.recordFailedTurn(ctx, req, conversation, req.ConversationID, "", nil, nil, "OpenAI integration is not configured")
		return errortypes.NewBusinessError("OpenAI integration is not configured")
	}

	apiKey := runtimeCfg.Config["apiKey"]
	if apiKey == "" {
		s.recordFailedTurn(ctx, req, conversation, req.ConversationID, "", nil, nil, "OpenAI API key is missing")
		return errortypes.NewBusinessError("OpenAI API key is missing")
	}

	client := openai.NewClient(
		option.WithAPIKey(apiKey),
		option.WithMaxRetries(5),
	)

	contextJSON := s.buildConversationContext(ctx, req)

	params := responses.ResponseNewParams{
		Model: openai.ChatModelGPT5_4,
		Input: s.buildConversationInput(req, contextJSON),
		Tools: buildTools(),
	}
	if req.ConversationID != "" {
		params.PreviousResponseID = openai.String(req.ConversationID)
	}

	var (
		allActions        []serviceports.ShipmentImportAction
		latestSuggestions []serviceports.ShipmentImportSuggestion
		toolCallLog       []serviceports.ShipmentImportToolCallRecord
		conversationID    string
		fullText          string
	)

	for round := range 5 {
		// For subsequent rounds, start a new message bubble in the UI
		if round > 0 {
			emit(serviceports.StreamEvent{Event: "new_message", Data: nil})
		}

		stream := client.Responses.NewStreaming(ctx, params)

		type pendingCall struct {
			callID string
			name   string
			args   string
		}
		var pendingCalls []pendingCall

		for stream.Next() {
			event := stream.Current()

			switch event.Type {
			case "response.output_text.delta":
				delta := event.AsResponseOutputTextDelta()
				fullText += delta.Delta
				emit(serviceports.StreamEvent{Event: "text_delta", Data: map[string]string{"delta": delta.Delta}})

			case "response.output_item.done":
				item := event.AsResponseOutputItemDone()
				if item.Item.Type == "function_call" {
					pendingCalls = append(pendingCalls, pendingCall{
						callID: item.Item.CallID,
						name:   item.Item.Name,
						args:   item.Item.Arguments.OfString,
					})
				}

			case "response.completed":
				completed := event.AsResponseCompleted()
				conversationID = completed.Response.ID
			}
		}

		if stream.Err() != nil {
			streamErr := stream.Err()
			s.logger.Error("stream error", zap.Error(streamErr))
			userMsg := friendlyStreamError(streamErr)
			s.recordFailedTurn(ctx, req, conversation, conversationID, fullText, toolCallLog, allActions, userMsg)
			emit(serviceports.StreamEvent{Event: "error", Data: map[string]string{"message": userMsg}})
			return nil
		}

		// No tool calls — we're done
		if len(pendingCalls) == 0 {
			break
		}

		// Execute tool calls and build outputs for next round
		var toolOutputs []responses.ResponseInputItemUnionParam

		for _, pc := range pendingCalls {
			if pc.name == "suggest_quick_actions" {
				var sugArgs struct {
					Suggestions []serviceports.ShipmentImportSuggestion `json:"suggestions"`
				}
				if err := json.Unmarshal([]byte(pc.args), &sugArgs); err == nil {
					latestSuggestions = sugArgs.Suggestions
				}
				toolOutputs = append(toolOutputs, responses.ResponseInputItemUnionParam{
					OfFunctionCallOutput: &responses.ResponseInputItemFunctionCallOutputParam{
						CallID: pc.callID,
						Output: responses.ResponseInputItemFunctionCallOutputOutputUnionParam{OfString: openai.String(`{"ok":true}`)},
					},
				})
				continue
			}

			// Emit tool start
			emit(serviceports.StreamEvent{Event: "tool_call_start", Data: map[string]string{"name": pc.name, "callId": pc.callID}})

			// Execute the tool
			result, toolActions := s.executeToolCall(ctx, req.TenantInfo, pc.name, pc.args)
			allActions = append(allActions, toolActions...)

			status := "completed"
			if len(result) > 0 {
				var check map[string]any
				if json.Unmarshal([]byte(result), &check) == nil {
					if _, hasErr := check["error"]; hasErr {
						status = "error"
					}
				}
			}

			toolCallLog = append(toolCallLog, serviceports.ShipmentImportToolCallRecord{
				Name:   pc.name,
				CallID: pc.callID,
				Status: status,
				Input:  pc.args,
				Output: result,
			})

			emit(serviceports.StreamEvent{Event: "tool_call_done", Data: map[string]any{
				"name":    pc.name,
				"callId":  pc.callID,
				"status":  status,
				"result":  result,
				"actions": toolActions,
			}})

			toolOutputs = append(toolOutputs, responses.ResponseInputItemUnionParam{
				OfFunctionCallOutput: &responses.ResponseInputItemFunctionCallOutputParam{
					CallID: pc.callID,
					Output: responses.ResponseInputItemFunctionCallOutputOutputUnionParam{OfString: openai.String(result)},
				},
			})
		}

		// Start next round with tool outputs
		params = responses.ResponseNewParams{
			Model:              openai.ChatModelGPT5_4,
			PreviousResponseID: openai.String(conversationID),
			Input:              responses.ResponseNewParamsInputUnion{OfInputItemList: toolOutputs},
			Tools:              buildTools(),
		}
	}

	// Emit suggestions only at the very end (so they match the final question, not intermediate steps)
	latestSuggestions = normalizeSuggestions(latestSuggestions)
	if len(latestSuggestions) > 0 {
		emit(serviceports.StreamEvent{Event: "suggestions", Data: map[string]any{"suggestions": latestSuggestions}})
	}

	emit(serviceports.StreamEvent{Event: "done", Data: map[string]any{
		"conversationId": conversationID,
		"actions":        allActions,
	}})

	if err = s.persistConversationTurn(
		ctx,
		req,
		conversation,
		conversationID,
		fullText,
		latestSuggestions,
		toolCallLog,
		allActions,
		shipmentimportchat.TurnResultStatusCompleted,
		"",
	); err != nil {
		return err
	}

	s.logAICall(ctx, req, fullText)
	return nil
}

func (s *Service) GetHistory(
	ctx context.Context,
	documentID string,
	tenantInfo pagination.TenantInfo,
) (*serviceports.ShipmentImportChatHistoryResponse, error) {
	id, err := pulid.Parse(documentID)
	if err != nil {
		return nil, errortypes.NewValidationError("documentId", errortypes.ErrInvalid, "Invalid document ID")
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
	return s.updateConversationStatus(
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
	return s.updateConversationStatus(
		ctx,
		documentID,
		tenantInfo,
		shipmentimportchat.ConversationStatusCompleted,
		shipmentimportchat.ConversationStatusReasonShipmentCreated,
	)
}

var _ serviceports.ShipmentImportAssistantService = (*Service)(nil)
