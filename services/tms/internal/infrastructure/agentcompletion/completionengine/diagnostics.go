package completionengine

import (
	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"go.uber.org/zap"
)

// noteCacheDiagnosis logs what the provider said about why the prompt cache
// missed against the call before this one, for an operator who turned cache
// diagnostics on. A changed prompt is the harness's to fix and is logged as
// such; a call with nothing to compare against is logged only for debugging.
func (s *Engine) NoteCacheDiagnosis(
	provider *aiprovider.Provider,
	req *llmports.ChatCompletionRequest,
	resp *modeladapter.Response,
) {
	diagnosis := resp.CacheDiagnosis
	if diagnosis == nil {
		return
	}

	fields := []zap.Field{
		zap.String("provider", provider.ID.String()),
		zap.String("model", resp.ModelIdentifier),
		zap.String("reason", diagnosis.Reason),
		zap.Int("missedTokens", diagnosis.MissedTokens),
		zap.Int("inputTokens", resp.InputTokens),
		zap.Int("cacheReadTokens", resp.CacheReadTokens),
		zap.Int("cacheWriteTokens", resp.CacheWriteTokens),
		zap.String("organization", req.TenantInfo.OrgID.String()),
		zap.String("agent", req.Attribution.AgentDefinitionID.String()),
		zap.String("thread", req.Attribution.ThreadID.String()),
		zap.String("run", req.Attribution.RunID.String()),
	}
	if diagnosis.Changed() {
		s.Logger.Info("the prompt stopped matching the call before it", fields...)
		return
	}
	s.Logger.Debug("the prompt was not compared with the call before it", fields...)
}
