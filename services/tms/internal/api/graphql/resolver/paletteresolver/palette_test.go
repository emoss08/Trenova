package paletteresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/paletteports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubPalette struct {
	paletteports.PaletteService

	asked   []*paletteports.AskRequest
	stopped []paletteports.AskRef
	kept    []paletteports.AskRef
	actors  []*permissionports.RequestActor
	thread  *conversation.Thread
}

func (s *stubPalette) Ask(
	_ context.Context,
	req *paletteports.AskRequest,
	actor *permissionports.RequestActor,
) (*paletteports.Ask, error) {
	s.asked = append(s.asked, req)
	s.actors = append(s.actors, actor)
	id := pulid.MustNew("pask_")
	return &paletteports.Ask{
		ID:        id,
		StreamURL: "/api/v1/assistant/ask/" + id.String() + "/stream/",
	}, nil
}

func (s *stubPalette) Stop(_ context.Context, ref paletteports.AskRef) error {
	s.stopped = append(s.stopped, ref)
	return nil
}

func (s *stubPalette) Keep(
	_ context.Context,
	ref paletteports.AskRef,
	actor *permissionports.RequestActor,
) (*paletteports.Kept, error) {
	s.kept = append(s.kept, ref)
	s.actors = append(s.actors, actor)
	return &paletteports.Kept{Thread: s.thread, Unanswered: "Where is S1?"}, nil
}

func TestPaletteMutations_CheckTheAssistantPermission(t *testing.T) {
	t.Parallel()

	askID := pulid.MustNew("pask_").String()
	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{"askPalette", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).AskPalette(
				ctx,
				gqlmodel.AskPaletteInput{Content: "?"},
			)
			return err
		}},
		{"stopPaletteAsk", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).StopPaletteAsk(ctx, askID)
			return err
		}},
		{"keepPaletteAsk", permission.OpCreate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).KeepPaletteAsk(ctx, askID)
			return err
		}},
	}

	for _, call := range calls {
		t.Run(call.name, func(t *testing.T) {
			t.Parallel()

			engine := resolvertest.RefusingEngine()
			deps := &Deps{Core: &resolvercore.Core{PermissionEngine: engine}}
			ctx := resolvertest.AuthedContext(t, resolvertest.UserAuthContext())

			err := call.run(ctx, deps)

			resolvertest.RequireRefused(
				t,
				err,
				engine,
				permission.ResourceAssistant,
				call.operation,
			)
		})
	}
}

func TestAskPalette_AsksTheQuestionWithItsPageButNeverItsDraft(t *testing.T) {
	t.Parallel()

	palette := &stubPalette{}
	deps := &Deps{
		Core:           &resolvercore.Core{PermissionEngine: &mocks.AllowAllPermissionEngine{}},
		PaletteService: palette,
	}
	authCtx := resolvertest.UserAuthContext()
	ctx := resolvertest.AuthedContext(t, authCtx)
	shipment := pulid.MustNew("shp_").String()

	ask, err := (&MutationResolver{deps}).AskPalette(ctx, gqlmodel.AskPaletteInput{
		Content: "Where is S1?",
		Context: map[string]any{
			"path":       "/shipments/" + shipment,
			"entityType": "shipment",
			"entityId":   shipment,
			"title":      "S1",
			"draft":      map[string]any{"fields": map[string]any{"weight": "40000"}},
		},
		Mentions: []*gqlmodel.PaletteMentionInput{
			{Type: "shipment", ID: shipment, Label: "S1"},
		},
	})
	require.NoError(t, err)

	require.Len(t, palette.asked, 1)
	req := palette.asked[0]
	assert.Equal(t, "Where is S1?", req.Question.Content)
	require.NotNil(t, req.Question.Page)
	assert.Equal(t, "/shipments/"+shipment, req.Question.Page.Path)
	assert.Equal(t, "shipment", req.Question.Page.EntityType)
	assert.Nil(
		t,
		req.Question.Page.Draft,
		"a form's unsaved values never ride with a palette question",
	)
	require.Len(t, req.Question.Mentions, 1)
	assert.Equal(t, shipment, req.Question.Mentions[0].ID)
	assert.Equal(t, authCtx.OrganizationID, req.TenantInfo.OrgID)
	assert.Equal(t, authCtx.UserID, palette.actors[0].UserID)
	assert.Equal(t, "/api/v1/assistant/ask/"+ask.ID.String()+"/stream/", ask.StreamURL)
}

func TestAskPalette_AskedFromNoPageCarriesNoPage(t *testing.T) {
	t.Parallel()

	question, err := questionFromInput(&gqlmodel.AskPaletteInput{Content: "What changed today?"})
	require.NoError(t, err)

	assert.Nil(t, question.Page)
	assert.Empty(t, question.Mentions)
}

func TestStopAndKeepPaletteAsk_NameTheQuestionByTheCaller(t *testing.T) {
	t.Parallel()

	threadID := pulid.MustNew("athr_")
	palette := &stubPalette{thread: &conversation.Thread{ID: threadID}}
	deps := &Deps{
		Core:           &resolvercore.Core{PermissionEngine: &mocks.AllowAllPermissionEngine{}},
		PaletteService: palette,
	}
	authCtx := resolvertest.UserAuthContext()
	ctx := resolvertest.AuthedContext(t, authCtx)
	askID := pulid.MustNew("pask_")

	stopped, err := (&MutationResolver{deps}).StopPaletteAsk(ctx, askID.String())
	require.NoError(t, err)
	kept, err := (&MutationResolver{deps}).KeepPaletteAsk(ctx, askID.String())
	require.NoError(t, err)

	assert.True(t, stopped)
	for _, ref := range append(palette.stopped, palette.kept...) {
		assert.Equal(t, askID, ref.AskID)
		assert.Equal(t, authCtx.UserID, ref.TenantInfo.UserID)
		assert.Equal(t, authCtx.OrganizationID, ref.TenantInfo.OrgID)
	}
	assert.Equal(t, threadID.String(), kept.ThreadID)
	assert.Equal(t, "Where is S1?", kept.Unanswered)
}

func TestStopPaletteAsk_RefusesAMalformedID(t *testing.T) {
	t.Parallel()

	deps := &Deps{
		Core:           &resolvercore.Core{PermissionEngine: &mocks.AllowAllPermissionEngine{}},
		PaletteService: &stubPalette{},
	}
	ctx := resolvertest.AuthedContext(t, resolvertest.UserAuthContext())

	_, err := (&MutationResolver{deps}).StopPaletteAsk(ctx, "not-an-id")

	var invalid *errortypes.Error
	require.ErrorAs(t, err, &invalid)
	assert.Equal(t, "askId", invalid.Field)
}
