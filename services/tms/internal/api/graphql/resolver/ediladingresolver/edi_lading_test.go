package ediladingresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEDILadingOperations_CheckTheEDIPermission(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("ldv_").String()
	x12 := "ISA*00*~"

	calls := []struct {
		name      string
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{"ediLadingContext", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).EDILadingContext(ctx)
			return err
		}},
		{"ediLadingPartnerProject", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).EDILadingPartnerProject(ctx, id)
			return err
		}},
		{"ediLadingVersions", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).EDILadingVersions(ctx, id, nil)
			return err
		}},
		{"ediLadingVersion", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).EDILadingVersion(ctx, id)
			return err
		}},
		{"ediLadingReplays", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).EDILadingReplays(ctx, id, nil)
			return err
		}},
		{"ediLadingShadowResults", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).EDILadingShadowResults(ctx, id, nil, nil)
			return err
		}},
		{"runEdiLadingVersion", permission.OpRead, func(ctx context.Context, deps *Deps) error {
			_, err := (&QueryResolver{deps}).RunEDILadingVersion(
				ctx,
				id,
				"tender_in",
				gqlmodel.EDILadingRunInput{X12: &x12},
				nil,
			)
			return err
		}},
		{
			"createEdiLadingProject",
			permission.OpCreate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).CreateEDILadingProject(
					ctx,
					gqlmodel.CreateEDILadingProjectInput{PartnerID: id, Role: "carrier"},
				)
				return err
			},
		},
		{"createEdiLadingDraft", permission.OpCreate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).CreateEDILadingDraft(ctx, id, id)
			return err
		}},
		{
			"updateEdiLadingSettings",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).UpdateEDILadingSettings(
					ctx,
					id,
					map[string]any{},
					1,
				)
				return err
			},
		},
		{
			"certifyEdiLadingVersion",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).CertifyEDILadingVersion(ctx, id)
				return err
			},
		},
		{
			"activateEdiLadingVersion",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).ActivateEDILadingVersion(ctx, id)
				return err
			},
		},
		{
			"archiveEdiLadingVersion",
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).ArchiveEDILadingVersion(ctx, id)
				return err
			},
		},
		{"startEdiLadingShadow", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).StartEDILadingShadow(ctx, id, 7)
			return err
		}},
		{"stopEdiLadingShadow", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).StopEDILadingShadow(ctx, id)
			return err
		}},
		{"startEdiLadingReplay", permission.OpUpdate, func(ctx context.Context, deps *Deps) error {
			_, err := (&MutationResolver{deps}).StartEDILadingReplay(
				ctx,
				id,
				gqlmodel.StartEDILadingReplayInput{Map: "tender_in", From: 1, To: 2},
			)
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

			resolvertest.RequireRefused(t, err, engine, permission.ResourceEDI, call.operation)
		})
	}
}

func TestCreateProjectRequest_MapsTheEnvelopePairs(t *testing.T) {
	t.Parallel()

	partnerID := pulid.MustNew("edip_")
	usage := "test"
	ackWithin := 3600

	req, err := createProjectRequest(&gqlmodel.CreateEDILadingProjectInput{
		PartnerID: partnerID.String(),
		Role:      "carrier",
		Sets:      []string{"204", "990"},
		Them:      []string{"02", "ACME"},
		Us:        []string{"ZZ", "TRENOVA"},
		Usage:     &usage,
		AckWithin: &ackWithin,
	})
	require.NoError(t, err)

	assert.Equal(t, partnerID, req.PartnerID)
	assert.Equal(t, []string{"204", "990"}, req.Sets)
	require.NotNil(t, req.Them)
	assert.Equal(t, [2]string{"02", "ACME"}, *req.Them)
	require.NotNil(t, req.Us)
	assert.Equal(t, [2]string{"ZZ", "TRENOVA"}, *req.Us)
	assert.Nil(t, req.Group)
	assert.Equal(t, "test", req.Usage)
	assert.Equal(t, 3600, req.AckWithin)
}

func TestCreateProjectRequest_RefusesAPairThatIsNotTwoValues(t *testing.T) {
	t.Parallel()

	_, err := createProjectRequest(&gqlmodel.CreateEDILadingProjectInput{
		PartnerID: pulid.MustNew("edip_").String(),
		Role:      "carrier",
		Them:      []string{"02"},
	})

	require.True(t, errortypes.IsError(err), "%v", err)
}

func TestRunInputFromGraphQL_EncodesJSONInputs(t *testing.T) {
	t.Parallel()

	csv := "a,b\n1,2"
	input, err := runInputFromGraphQL(&gqlmodel.EDILadingRunInput{
		CSV:      &csv,
		JSON:     map[string]any{"load": "B-1"},
		Document: nil,
	})
	require.NoError(t, err)

	assert.Equal(t, &csv, input.CSV)
	assert.Nil(t, input.X12)
	assert.JSONEq(t, `{"load":"B-1"}`, string(input.JSON))
	assert.Nil(t, input.Document)
}

func TestListLimit(t *testing.T) {
	t.Parallel()

	few, tooMany, none := 10, 1000, 0
	assert.Equal(t, maxListLimit, listLimit(nil))
	assert.Equal(t, maxListLimit, listLimit(&none))
	assert.Equal(t, 10, listLimit(&few))
	assert.Equal(t, maxListLimit, listLimit(&tooMany))
}
