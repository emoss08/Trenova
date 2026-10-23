package formulatemplatehandler_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type formulaPageAssistant struct {
	opened []*agentports.OpenPageThreadRequest
}

func (p *formulaPageAssistant) OpenPageThread(
	_ context.Context,
	req *agentports.OpenPageThreadRequest,
	_ *permissionports.RequestActor,
) (*agentports.PageThread, error) {
	p.opened = append(p.opened, req)
	thread := &conversation.Thread{ID: pulid.MustNew("athr_"), Origin: req.Origin}
	if req.SubjectID.IsNotNil() {
		thread.SubjectType = req.SubjectType
		thread.SubjectID = req.SubjectID
	}

	return &agentports.PageThread{
		Thread: thread,
		Agent:  agentports.PageAgent{Name: "Formula assistant", SystemKey: "formula_assistant"},
	}, nil
}

func openFormulaThread(
	t *testing.T,
	granted map[string]bool,
	assistant *formulaPageAssistant,
	prime func(f *fixture),
	body map[string]any,
) *testutil.GinTestContext {
	t.Helper()

	f := newFixture(t, fixtureOptions{granted: granted, assistant: assistant})
	if prime != nil {
		prime(f)
	}
	return f.serve(http.MethodPost, "/api/v1/formula-templates/ai/thread/", body)
}

func allowedFormulaAssistant() map[string]bool {
	return map[string]bool{"formula_template:read": true, "assistant:create": true}
}

func TestOpenAssistantThread_OpensTheTemplatesConversation(t *testing.T) {
	t.Parallel()

	templateID := pulid.MustNew("ft_")
	assistant := &formulaPageAssistant{}
	ginCtx := openFormulaThread(t, allowedFormulaAssistant(), assistant, func(f *fixture) {
		f.repo.On("GetByID", mock.Anything, mock.Anything).
			Return(&formulatemplate.FormulaTemplate{ID: templateID, Name: "Per Mile"}, nil)
	}, map[string]any{"templateId": templateID.String()})

	require.Equal(t, http.StatusOK, ginCtx.ResponseCode(), string(ginCtx.ResponseBody()))
	require.Len(t, assistant.opened, 1)
	assert.Equal(t, conversation.ThreadOriginFormula, assistant.opened[0].Origin)
	assert.Equal(t, agent.SubjectFormulaTemplate, assistant.opened[0].SubjectType)
	assert.Equal(t, templateID, assistant.opened[0].SubjectID)
}

func TestOpenAssistantThread_ANewTemplateHasNoSubject(t *testing.T) {
	t.Parallel()

	assistant := &formulaPageAssistant{}
	ginCtx := openFormulaThread(t, allowedFormulaAssistant(), assistant, nil, map[string]any{})

	require.Equal(t, http.StatusOK, ginCtx.ResponseCode(), string(ginCtx.ResponseBody()))
	require.Len(t, assistant.opened, 1)
	assert.True(t, assistant.opened[0].SubjectID.IsNil())
}

func TestOpenAssistantThread_RefusesATemplateItCannotFind(t *testing.T) {
	t.Parallel()

	assistant := &formulaPageAssistant{}
	ginCtx := openFormulaThread(t, allowedFormulaAssistant(), assistant, func(f *fixture) {
		f.repo.On("GetByID", mock.Anything, mock.Anything).
			Return(nil, errortypes.NewNotFoundError("Formula template not found"))
	}, map[string]any{"templateId": pulid.MustNew("ft_").String()})

	assert.Equal(t, http.StatusNotFound, ginCtx.ResponseCode())
	assert.Empty(t, assistant.opened)

	bad := openFormulaThread(t, allowedFormulaAssistant(), assistant, nil,
		map[string]any{"templateId": "not-an-id"})
	assert.Equal(t, http.StatusBadRequest, bad.ResponseCode())
	assert.Empty(t, assistant.opened)
}

func TestOpenAssistantThread_NeedsTheFormulaAndTheAssistant(t *testing.T) {
	t.Parallel()

	for name, granted := range map[string]map[string]bool{
		"no formula":   {"assistant:create": true},
		"no assistant": {"formula_template:read": true, "assistant:read": true},
	} {
		assistant := &formulaPageAssistant{}
		ginCtx := openFormulaThread(t, granted, assistant, nil, map[string]any{})

		assert.Equal(t, http.StatusForbidden, ginCtx.ResponseCode(), name)
		assert.Empty(t, assistant.opened, name)
	}
}
