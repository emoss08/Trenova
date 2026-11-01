package deskcaseresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/deskcase"
	"github.com/emoss08/trenova/internal/modules/agent/ports/deskcaseports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type cases struct {
	deskcaseports.AssistantCaseService

	bound *deskcaseports.BindCaseRequest
}

func (c *cases) Bind(
	_ context.Context,
	req *deskcaseports.BindCaseRequest,
) (*conversation.Thread, error) {
	c.bound = req

	return &conversation.Thread{
		ID:           req.ThreadID,
		SubjectType:  req.SubjectType,
		SubjectID:    req.SubjectID,
		SnoozeAnchor: deskcase.SnoozeTime,
	}, nil
}

func TestCaseOperationsNeedOnlyTheAssistant(t *testing.T) {
	t.Parallel()

	engine := resolvertest.RefusingEngine()
	resolver := &MutationResolver{&Deps{Core: &resolvercore.Core{
		L:                zap.NewNop(),
		PermissionEngine: engine,
	}}}

	_, err := resolver.WakeAssistantCase(
		resolvertest.AuthedContext(t, resolvertest.UserAuthContext()),
		pulid.MustNew("athr_").String(),
	)

	resolvertest.RequireRefused(t, err, engine, permission.ResourceAssistant, permission.OpRead)
}

func TestBindAssistantCaseBindsAsTheCallerAndAnswersTheBinding(t *testing.T) {
	t.Parallel()

	auth := resolvertest.UserAuthContext()
	service := &cases{}
	resolver := &MutationResolver{&Deps{
		Core: &resolvercore.Core{
			L: zap.NewNop(),
			PermissionEngine: &resolvertest.GrantingPermissionEngine{Granted: map[string]bool{
				permission.ResourceAssistant.String() + "|" + string(permission.OpRead): true,
			}},
		},
		AssistantCaseService: service,
	}}
	threadID, shipmentID := pulid.MustNew("athr_"), pulid.MustNew("shp_")

	got, err := resolver.BindAssistantCase(
		resolvertest.AuthedContext(t, auth),
		threadID.String(),
		"Shipment",
		shipmentID.String(),
	)

	require.NoError(t, err)
	require.NotNil(t, service.bound)
	assert.Equal(t, threadID, service.bound.ThreadID)
	assert.Equal(t, agent.SubjectType("Shipment"), service.bound.SubjectType)
	assert.Equal(t, shipmentID, service.bound.SubjectID)
	assert.Equal(t, auth.UserID, service.bound.Actor.UserID)
	assert.Equal(t, auth.UserID, service.bound.TenantInfo.UserID)
	assert.Equal(t, caseBinding{
		ThreadID:     threadID,
		SubjectType:  "Shipment",
		SubjectID:    shipmentID,
		SnoozeAnchor: deskcase.SnoozeTime,
	}, got)
}
