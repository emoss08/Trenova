//go:build integration

package emailrepository_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/testutil/seededtest"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/workspace/adapters/postgres/emailrepository"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestCreateMessageOnce_Integration(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	defer cleanup()

	var org struct {
		ID             pulid.ID `bun:"id"`
		BusinessUnitID pulid.ID `bun:"business_unit_id"`
	}
	require.NoError(t, db.NewSelect().
		Table("organizations").
		Column("id", "business_unit_id").
		Limit(1).
		Scan(ctx, &org))

	repo := emailrepository.New(emailrepository.Params{
		DB:     postgres.NewTestConnection(db),
		Logger: zap.NewNop(),
	})

	draft := &email.Profile{
		BusinessUnitID: org.BusinessUnitID,
		OrganizationID: org.ID,
		Name:           "Idempotency test",
		Provider:       email.ProviderResend,
		SenderName:     "Dispatch",
		SenderEmail:    "dispatch@example.com",
		Status:         email.ProfileStatusActive,
	}
	multiErr := errortypes.NewMultiError()
	draft.Validate(multiErr)
	require.False(t, multiErr.HasErrors(), multiErr.Error())
	profile, err := repo.CreateProfile(ctx, draft)
	require.NoError(t, err)

	message := func(subject string) *email.Message {
		return &email.Message{
			BusinessUnitID: org.BusinessUnitID,
			OrganizationID: org.ID,
			ProfileID:      profile.ID,
			Purpose:        email.PurposeGeneral,
			Provider:       email.ProviderResend,
			IdempotencyKey: "tender-offer-tof_integration",
			Status:         email.MessageStatusQueued,
			FromEmail:      "dispatch@example.com",
			FromName:       "Dispatch",
			ToRecipients:   []string{"carrier@example.com"},
			Subject:        subject,
		}
	}

	first, created, err := repo.CreateMessageOnce(ctx, message("First"))
	require.NoError(t, err)
	require.True(t, created)
	require.True(t, first.ID.IsNotNil())

	second, created, err := repo.CreateMessageOnce(ctx, message("Second"))
	require.NoError(t, err)
	assert.False(t, created)
	assert.Equal(t, first.ID, second.ID)
	assert.Equal(t, "First", second.Subject)
	assert.Equal(t, []string{"carrier@example.com"}, second.ToRecipients)
}
