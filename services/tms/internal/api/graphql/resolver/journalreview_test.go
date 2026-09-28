package resolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/journalentry"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJournalEntryStatusAndTypeResolveFromTheEntry(t *testing.T) {
	t.Parallel()

	entry := &journalentry.JournalEntry{
		Status:    journalentry.StatusApproved,
		EntryType: journalentry.EntryType("Standard"),
	}
	resolver := &journalEntryResolver{}

	status, err := resolver.Status(t.Context(), entry)
	require.NoError(t, err)
	assert.Equal(t, "Approved", status)

	entryType, err := resolver.EntryType(t.Context(), entry)
	require.NoError(t, err)
	assert.Equal(t, "Standard", entryType)
}

func TestJournalReviewRequestParsesEveryEntryID(t *testing.T) {
	t.Parallel()

	authCtx := &authctx.AuthContext{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}
	first, second := pulid.MustNew("je_"), pulid.MustNew("je_")

	req, err := journalReviewRequest(authCtx, gqlmodel.JournalReviewInput{
		EntryIds: []string{first.String(), second.String()},
	})
	require.NoError(t, err)
	assert.Equal(t, []pulid.ID{first, second}, req.EntryIDs)
	assert.Equal(t, authCtx.OrganizationID, req.TenantInfo.OrgID)
	assert.Equal(t, authCtx.BusinessUnitID, req.TenantInfo.BuID)

	_, err = journalReviewRequest(authCtx, gqlmodel.JournalReviewInput{EntryIds: []string{"not-an-id"}})
	var validation *errortypes.Error
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, "entryIds", validation.Field)
}
