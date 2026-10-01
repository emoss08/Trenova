package completionrouter

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
)

func (s *Service) ChatContextWindow(
	ctx context.Context,
	req serviceports.ChatContextWindowRequest,
) (serviceports.ChatContextWindow, error) {
	usable, err := s.usableFor(ctx, aiprovider.TaskAssistantChat, req.TenantInfo)
	if err != nil {
		return serviceports.ChatContextWindow{}, err
	}

	usable = preferFirst(usable, req.PreferredProviderID)
	if req.Pinned {
		usable = pinPreferred(usable, req.PreferredProviderID)
	}

	return windowAcross(usable), nil
}

func windowAcross(providers []*aiprovider.Provider) serviceports.ChatContextWindow {
	var window serviceports.ChatContextWindow
	for _, provider := range providers {
		tokens := provider.ResolvedContextWindow()
		if window.Tokens == 0 || tokens < window.Tokens {
			window.Tokens = tokens
		}
		window.ReplyTokens = max(window.ReplyTokens, provider.ResolvedMaxTokens())
	}
	if window.Tokens > 0 {
		window.ReplyTokens = min(window.ReplyTokens, window.Tokens/4)
	}

	return window
}
