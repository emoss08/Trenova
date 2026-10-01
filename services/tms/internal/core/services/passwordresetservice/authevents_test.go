package passwordresetservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/iam"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingAuthEvents struct {
	records []serviceports.AuthEventRecord
}

func (r *recordingAuthEvents) Record(_ context.Context, rec serviceports.AuthEventRecord) {
	r.records = append(r.records, rec)
}

func TestRequestResetRecordsEachOutcome(t *testing.T) {
	t.Parallel()

	inactive := activeUser()
	inactive.Status = domaintypes.StatusInactive

	tests := []struct {
		name     string
		users    *stubUserRepo
		tokens   *stubTokenRepo
		outcome  iam.AuthEventOutcome
		code     string
		withUser bool
	}{
		{"issued", &stubUserRepo{user: activeUser()}, &stubTokenRepo{}, iam.AuthEventOutcomeSuccess, "", true},
		{
			"unknown address",
			&stubUserRepo{findErr: errors.New("not found")},
			&stubTokenRepo{},
			iam.AuthEventOutcomeFailed,
			resetErrorUnknownAccount,
			false,
		},
		{
			"inactive account",
			&stubUserRepo{user: inactive},
			&stubTokenRepo{},
			iam.AuthEventOutcomeDenied,
			resetErrorAccountUnavailable,
			true,
		},
		{
			"rate limited",
			&stubUserRepo{user: activeUser()},
			&stubTokenRepo{countSince: 5},
			iam.AuthEventOutcomeDenied,
			resetErrorRateLimited,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			events := &recordingAuthEvents{}
			svc := newService(t, tt.users, tt.tokens, &stubSessionRepo{})
			svc.authEvents = events

			require.NoError(t, svc.RequestReset(t.Context(), "dana@example.com"))

			require.Len(t, events.records, 1)
			rec := events.records[0]
			assert.Equal(t, serviceports.AuthEventProviderPasswordResetRequest, rec.Provider)
			assert.Equal(t, tt.outcome, rec.Outcome)
			assert.Equal(t, tt.code, rec.ErrorCode)
			if tt.withUser {
				assert.Equal(t, tt.users.user.ID, rec.UserID)
				assert.Equal(t, tt.users.user.CurrentOrganizationID, rec.OrganizationID)
			} else {
				assert.True(t, rec.UserID.IsNil())
				assert.True(t, rec.OrganizationID.IsNil())
			}
		})
	}
}

func TestResetPasswordRecordsTheRedemption(t *testing.T) {
	t.Parallel()

	user := activeUser()
	token := redeemableToken(user)
	events := &recordingAuthEvents{}
	svc := newService(t, &stubUserRepo{user: user}, &stubTokenRepo{redeemable: token, markUsedOK: true}, &stubSessionRepo{})
	svc.authEvents = events

	require.NoError(t, svc.ResetPassword(t.Context(), "raw-token", "a-good-password"))

	require.Len(t, events.records, 1)
	rec := events.records[0]
	assert.Equal(t, serviceports.AuthEventProviderPasswordResetConfirm, rec.Provider)
	assert.Equal(t, iam.AuthEventOutcomeSuccess, rec.Outcome)
	assert.Equal(t, user.ID, rec.UserID)
	assert.Equal(t, token.OrganizationID, rec.OrganizationID)
}

func TestResetPasswordRecordsARefusedToken(t *testing.T) {
	t.Parallel()

	events := &recordingAuthEvents{}
	svc := newService(
		t,
		&stubUserRepo{user: activeUser()},
		&stubTokenRepo{findErr: errors.New("not found")},
		&stubSessionRepo{},
	)
	svc.authEvents = events

	require.Error(t, svc.ResetPassword(t.Context(), "raw-token", "a-good-password"))

	require.Len(t, events.records, 1)
	assert.Equal(t, iam.AuthEventOutcomeFailed, events.records[0].Outcome)
	assert.Equal(t, resetErrorInvalidToken, events.records[0].ErrorCode)
	assert.True(t, events.records[0].OrganizationID.IsNil())
}

func TestResetPasswordDoesNotRecordAWeakPassword(t *testing.T) {
	t.Parallel()

	events := &recordingAuthEvents{}
	svc := newService(t, &stubUserRepo{}, &stubTokenRepo{}, &stubSessionRepo{})
	svc.authEvents = events

	require.Error(t, svc.ResetPassword(t.Context(), "raw-token", "short"))
	assert.Empty(t, events.records)
}
