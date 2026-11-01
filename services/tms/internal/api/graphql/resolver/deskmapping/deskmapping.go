package deskmapping

import (
	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationturn"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/pagedraft"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

func ThreadRequest(
	authCtx *authctx.AuthContext,
	field, rawID string,
) (conversationports.GetThreadRequest, error) {
	threadID, err := gqlid.ParseIDField(field, rawID)
	if err != nil {
		return conversationports.GetThreadRequest{}, err
	}

	return conversationports.GetThreadRequest{
		ID:         threadID,
		UserID:     authCtx.UserID,
		TenantInfo: resolvercore.TenantInfo(authCtx),
	}, nil
}

func StartedTurn(turn *conversationturn.AssistantTurn) *gqlmodel.AssistantStartedTurn {
	return &gqlmodel.AssistantStartedTurn{
		TurnID:    turn.ID.String(),
		ThreadID:  turn.ThreadID.String(),
		StreamURL: "/api/v1/assistant/turns/" + turn.ID.String() + "/stream/",
		Status:    string(turn.Status),
	}
}

func PageContext(
	input *gqlmodel.AssistantPageContextInput,
) (*agentpagecontext.PageContext, error) {
	if input == nil {
		return nil, nil
	}

	page := &agentpagecontext.PageContext{
		Path:       input.Path,
		EntityType: input.EntityType,
		EntityID:   input.EntityID,
		Title:      input.Title,
	}
	if input.View != nil {
		page.View = new(agentpagecontext.PageView)
		if err := decodeJSON(input.View, page.View); err != nil {
			return nil, contextError("context.view", "The page's table could not be read")
		}
	}
	if input.Draft != nil {
		page.Draft = new(pagedraft.Draft)
		if err := decodeJSON(input.Draft, page.Draft); err != nil {
			return nil, contextError("context.draft", "The page's unsaved work could not be read")
		}
	}

	return page, nil
}

func decodeJSON(value any, dest any) error {
	raw, err := sonic.Marshal(value)
	if err != nil {
		return err
	}

	return sonic.Unmarshal(raw, dest)
}

func contextError(field, message string) error {
	return errortypes.NewValidationError(field, errortypes.ErrInvalid, message)
}

func Mentions(input []*gqlmodel.AssistantEntityRefInput) []agentpagecontext.EntityRef {
	refs := make([]agentpagecontext.EntityRef, 0, len(input))
	for _, ref := range input {
		if ref == nil {
			continue
		}
		refs = append(refs, agentpagecontext.EntityRef{
			Type:  ref.Type,
			ID:    ref.ID,
			Label: ref.Label,
		})
	}

	return refs
}

func IDs(field string, raw []string) ([]pulid.ID, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	return gqlid.ParseIDListField(field, raw)
}

func Provider(field string, raw *string) (pulid.ID, bool, error) {
	if raw == nil {
		return pulid.Nil, false, nil
	}
	if *raw == "" {
		return pulid.Nil, true, nil
	}
	id, err := gqlid.ParseIDField(field, *raw)
	if err != nil {
		return pulid.Nil, false, err
	}

	return id, true, nil
}
