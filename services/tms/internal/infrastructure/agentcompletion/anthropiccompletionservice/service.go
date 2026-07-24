package anthropiccompletionservice

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/integrationservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/errortypes"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger      *zap.Logger
	Config      *config.Config
	Integration *integrationservice.Service
}

type Service struct {
	logger      *zap.Logger
	cfg         *config.DocumentIntelligenceConfig
	integration *integrationservice.Service
	httpClient  *http.Client
}

func New(p Params) serviceports.CompletionService {
	cfg := p.Config.GetDocumentIntelligenceConfig()
	return &Service{
		logger:      p.Logger.Named("service.anthropic-completion"),
		cfg:         cfg,
		integration: p.Integration,
		httpClient: &http.Client{
			Timeout: cfg.GetAITimeout(),
		},
	}
}

func (s *Service) Diagnose(
	ctx context.Context,
	req *serviceports.DiagnoseRequest,
) (*serviceports.DiagnoseResult, error) {
	if !s.cfg.AIEnabled() {
		return nil, errortypes.NewBusinessError("AI billing exception agent is disabled")
	}

	runtimeCfg, err := s.integration.GetRuntimeConfig(ctx, req.TenantInfo, integration.TypeAnthropic)
	if err != nil {
		return nil, err
	}

	apiKey := strings.TrimSpace(runtimeCfg.Config["apiKey"])
	if apiKey == "" {
		return nil, errortypes.NewBusinessError("Anthropic API key is not configured")
	}

	resp, err := s.executeMessages(ctx, apiKey, s.buildMessagesRequest(req))
	if err != nil {
		return nil, err
	}

	if resp.StopReason == "refusal" {
		return nil, errortypes.NewBusinessError("model declined to produce a diagnosis")
	}

	text := firstTextBlock(resp)
	if text == "" {
		return nil, fmt.Errorf("model returned no structured content")
	}

	var payload diagnosisPayload
	if err = sonic.Unmarshal([]byte(text), &payload); err != nil {
		return nil, fmt.Errorf("%w: %w", serviceports.ErrModelSchemaValidation, err)
	}

	return mapDiagnosis(payload, resp.Model), nil
}

func (s *Service) buildMessagesRequest(req *serviceports.DiagnoseRequest) messagesRequest {
	return messagesRequest{
		Model:     defaultModel,
		MaxTokens: defaultMaxTokens,
		System:    buildSystemPrompt(req),
		Messages: []message{
			{Role: "user", Content: BuildContextText(req.Context)},
		},
		OutputConfig: &outputConfig{
			Format: outputFormat{
				Type:   "json_schema",
				Schema: buildDiagnosisSchema(),
			},
		},
	}
}

func firstTextBlock(resp *messagesResponse) string {
	for _, block := range resp.Content {
		if block.Type == "text" && strings.TrimSpace(block.Text) != "" {
			return block.Text
		}
	}

	return ""
}

func mapDiagnosis(payload diagnosisPayload, model string) *serviceports.DiagnoseResult {
	result := &serviceports.DiagnoseResult{
		ModelIdentifier: model,
		Proposals:       make([]serviceports.ProposedAction, 0, len(payload.Proposals)),
		Exceptions:      make([]serviceports.RaisedException, 0, len(payload.Exceptions)),
	}

	for _, p := range payload.Proposals {
		result.Proposals = append(result.Proposals, serviceports.ProposedAction{
			ToolName:   p.ToolName,
			ToolParams: p.ToolParams,
			Confidence: p.Confidence,
			Rationale:  p.Rationale,
			Evidence:   p.Evidence,
		})
	}

	for _, e := range payload.Exceptions {
		result.Exceptions = append(result.Exceptions, serviceports.RaisedException{
			Category:       e.Category,
			Severity:       e.Severity,
			AttemptSummary: e.AttemptSummary,
			Evidence:       e.Evidence,
			BlastRadius:    e.BlastRadius,
		})
	}

	return result
}
