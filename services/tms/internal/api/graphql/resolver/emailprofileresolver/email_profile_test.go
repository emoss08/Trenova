package emailprofileresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmailOperations_CheckTheirPermission(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("emlprof_").String()
	input := gqlmodel.EmailProfileInput{
		Name:        "Billing",
		SenderName:  "Acme Billing",
		SenderEmail: "billing@acme.test",
	}

	calls := []struct {
		name      string
		resource  permission.Resource
		operation permission.Operation
		run       func(ctx context.Context, deps *Deps) error
	}{
		{
			"emailLogs",
			permission.ResourceEmailLog,
			permission.OpRead,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&QueryResolver{deps}).EmailLogs(ctx, gqlmodel.DataTableConnectionInput{})
				return err
			},
		},
		{
			"emailProfileAssignments",
			permission.ResourceEmailProfile,
			permission.OpRead,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&QueryResolver{deps}).EmailProfileAssignments(ctx)
				return err
			},
		},
		{
			"createEmailProfile",
			permission.ResourceEmailProfile,
			permission.OpCreate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).CreateEmailProfile(ctx, input)
				return err
			},
		},
		{
			"updateEmailProfile",
			permission.ResourceEmailProfile,
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).UpdateEmailProfile(ctx, id, input)
				return err
			},
		},
		{
			"deleteEmailProfile",
			permission.ResourceEmailProfile,
			permission.OpDelete,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).DeleteEmailProfile(ctx, id)
				return err
			},
		},
		{
			"sendEmailProfileTest",
			permission.ResourceEmailProfile,
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).SendEmailProfileTest(
					ctx,
					id,
					gqlmodel.TestEmailProfileInput{To: "ops@acme.test", Subject: "Test"},
				)
				return err
			},
		},
		{
			"updateEmailProfileAssignments",
			permission.ResourceEmailProfile,
			permission.OpUpdate,
			func(ctx context.Context, deps *Deps) error {
				_, err := (&MutationResolver{deps}).UpdateEmailProfileAssignments(ctx, nil)
				return err
			},
		},
	}

	for _, call := range calls {
		t.Run(call.name, func(t *testing.T) {
			t.Parallel()

			engine := resolvertest.RefusingEngine()
			deps := &Deps{Core: &resolvercore.Core{PermissionEngine: engine}}
			ctx := resolvertest.AuthedContext(t, resolvertest.UserAuthContext())

			err := call.run(ctx, deps)

			resolvertest.RequireRefused(t, err, engine, call.resource, call.operation)
		})
	}
}

func TestApplyEmailProfileInput_DefaultsProviderAndStatus(t *testing.T) {
	t.Parallel()

	profile := &email.Profile{AuthType: email.AuthTypeAPIKey, Version: 4}
	replyTo := "replies@acme.test"
	applyEmailProfileInput(profile, &gqlmodel.EmailProfileInput{
		Name:         "Billing",
		SenderName:   "Acme Billing",
		SenderEmail:  "billing@acme.test",
		ReplyToEmail: &replyTo,
	})

	assert.Equal(t, "Billing", profile.Name)
	assert.Equal(t, "billing@acme.test", profile.SenderEmail)
	assert.Equal(t, replyTo, profile.ReplyToEmail)
	assert.Equal(t, email.ProviderResend, profile.Provider)
	assert.Equal(t, email.ProfileStatusActive, profile.Status)
	assert.Equal(t, email.AuthTypeAPIKey, profile.AuthType)
	assert.Equal(t, int64(4), profile.Version)
}

func TestApplyEmailProfileInput_KeepsAChosenProviderAndStatus(t *testing.T) {
	t.Parallel()

	provider := email.ProviderPostmark
	status := email.ProfileStatusInactive
	profile := &email.Profile{}
	applyEmailProfileInput(profile, &gqlmodel.EmailProfileInput{
		Name:        "Ops",
		SenderName:  "Ops",
		SenderEmail: "ops@acme.test",
		Provider:    &provider,
		Status:      &status,
	})

	assert.Equal(t, email.ProviderPostmark, profile.Provider)
	assert.Equal(t, email.ProfileStatusInactive, profile.Status)
}

func TestAssignmentsFromInput(t *testing.T) {
	t.Parallel()

	profileID := pulid.MustNew("emlprof_")
	assignments, err := assignmentsFromInput([]*gqlmodel.EmailProfileAssignmentInput{
		{Purpose: email.PurposeBilling, ProfileID: profileID.String()},
		nil,
	})
	require.NoError(t, err)
	require.Len(t, assignments, 1)
	assert.Equal(t, email.PurposeBilling, assignments[0].Purpose)
	assert.Equal(t, profileID, assignments[0].ProfileID)

	_, err = assignmentsFromInput([]*gqlmodel.EmailProfileAssignmentInput{
		{Purpose: email.PurposeBilling, ProfileID: "nope"},
	})
	require.True(t, errortypes.IsError(err), "%v", err)
}
