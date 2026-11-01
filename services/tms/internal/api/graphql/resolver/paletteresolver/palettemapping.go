package paletteresolver

import (
	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/internal/modules/agent/ports/paletteports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
)

func questionFromInput(input *gqlmodel.AskPaletteInput) (paletteports.Question, error) {
	page, err := pageContextFromInput(input.Context)
	if err != nil {
		return paletteports.Question{}, err
	}

	mentions := make([]agentpagecontext.EntityRef, 0, len(input.Mentions))
	for _, mention := range input.Mentions {
		mentions = append(mentions, agentpagecontext.EntityRef{
			Type:  mention.Type,
			ID:    mention.ID,
			Label: mention.Label,
		})
	}

	return paletteports.Question{Content: input.Content, Page: page, Mentions: mentions}, nil
}

func pageContextFromInput(raw map[string]any) (*agentpagecontext.PageContext, error) {
	if raw == nil {
		return nil, nil //nolint:nilnil // a question asked from no page carries no page context
	}

	encoded, err := sonic.Marshal(raw)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"context", errortypes.ErrInvalid, "The page context is not valid",
		)
	}
	page := new(agentpagecontext.PageContext)
	if err = sonic.Unmarshal(encoded, page); err != nil {
		return nil, errortypes.NewValidationError(
			"context", errortypes.ErrInvalid, "The page context is not valid",
		)
	}
	page.Draft = nil

	return page, nil
}

func askRef(authCtx *authctx.AuthContext, askID string) (paletteports.AskRef, error) {
	id, err := gqlid.ParseIDField("askId", askID)
	if err != nil {
		return paletteports.AskRef{}, err
	}

	return paletteports.AskRef{AskID: id, TenantInfo: resolvercore.TenantInfo(authCtx)}, nil
}

func keptToModel(kept *paletteports.Kept) *gqlmodel.KeptPaletteAsk {
	out := &gqlmodel.KeptPaletteAsk{Unanswered: kept.Unanswered}
	if kept.Thread != nil {
		out.ThreadID = kept.Thread.ID.String()
	}

	return out
}
