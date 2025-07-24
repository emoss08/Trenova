/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package external

import (
	"github.com/emoss08/trenova/internal/infrastructure/external/ai/claude"
	"github.com/emoss08/trenova/internal/infrastructure/external/maps/googlemaps"
	"github.com/emoss08/trenova/internal/infrastructure/external/maps/pcmiler"
	"github.com/emoss08/trenova/internal/pkg/config"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"go.uber.org/fx"
)

// ClaudeClientParams contains dependencies for Claude client
type ClaudeClientParams struct {
	fx.In

	Config *config.Config
	Logger *logger.Logger
}

// NewClaudeClient creates a new Claude API client
func NewClaudeClient(p ClaudeClientParams) *claude.Client {
	cfg := claude.Config{
		APIKey:      p.Config.AI.ClaudeAPIKey,
		MaxTokens:   p.Config.AI.MaxTokens,
		Temperature: p.Config.AI.Temperature,
	}

	return claude.NewClient(claude.ClientParams{
		Logger: p.Logger,
		Config: cfg,
	})
}

var Module = fx.Module("external", fx.Provide(
	pcmiler.NewClient,
	googlemaps.NewClient,
	NewClaudeClient,
))
