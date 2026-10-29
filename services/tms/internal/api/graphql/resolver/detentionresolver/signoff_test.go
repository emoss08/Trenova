package detentionresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/loaders/identityloaders"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type usersByID struct {
	tenantports.UserRepository

	users    map[pulid.ID]*tenant.User
	requests [][]pulid.ID
}

func (u *usersByID) GetByIDs(
	_ context.Context,
	req tenantports.GetUsersByIDsRequest,
) ([]*tenant.User, error) {
	u.requests = append(u.requests, req.UserIDs)
	found := make([]*tenant.User, 0, len(req.UserIDs))
	for _, id := range req.UserIDs {
		if user, ok := u.users[id]; ok {
			found = append(found, user)
		}
	}
	return found, nil
}

func withUserLoader(ctx context.Context, users *usersByID) context.Context {
	factory := identityloaders.NewUserByIDLoaderFactory(
		identityloaders.UserByIDLoaderFactoryParams{Repo: users},
	)
	return factory.Attach(ctx, pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	})
}

func TestDetentionOccurrenceToModel_CarriesWhoApprovedAndWaivedIt(t *testing.T) {
	t.Parallel()

	approverID := pulid.MustNew("usr_")
	waiverID := pulid.MustNew("usr_")
	approvedAt := int64(1_791_849_600)

	model := detentionOccurrenceToModel(&detention.DetentionOccurrence{
		ID:           pulid.MustNew("dto_"),
		Status:       detention.OccurrenceStatusWaived,
		ApprovedByID: &approverID,
		ApprovedAt:   &approvedAt,
		WaivedByID:   &waiverID,
	})

	require.NotNil(t, model)
	require.NotNil(t, model.ApprovedByID)
	assert.Equal(t, approverID.String(), *model.ApprovedByID)
	require.NotNil(t, model.ApprovedAt)
	assert.Equal(t, int(approvedAt), *model.ApprovedAt)
	require.NotNil(t, model.WaivedByID)
	assert.Equal(t, waiverID.String(), *model.WaivedByID)
}

func TestDetentionOccurrenceToModel_LeavesSignoffsNullWhenNobodySigned(t *testing.T) {
	t.Parallel()

	model := detentionOccurrenceToModel(&detention.DetentionOccurrence{
		ID:     pulid.MustNew("dto_"),
		Status: detention.OccurrenceStatusAccruing,
	})

	require.NotNil(t, model)
	assert.Nil(t, model.ApprovedByID)
	assert.Nil(t, model.ApprovedAt)
	assert.Nil(t, model.WaivedByID)
}

func TestDetentionOccurrenceSignoffs_ReadWhoApprovedAndWaivedThroughTheLoader(t *testing.T) {
	t.Parallel()

	approverID := pulid.MustNew("usr_")
	waiverID := pulid.MustNew("usr_")
	users := &usersByID{users: map[pulid.ID]*tenant.User{
		approverID: {ID: approverID, Name: "Dana Ruiz"},
		waiverID:   {ID: waiverID, Name: "Mia Chen"},
	}}
	ctx := withUserLoader(t.Context(), users)
	approver, waiver := approverID.String(), waiverID.String()
	occurrence := &gqlmodel.DetentionOccurrence{ApprovedByID: &approver, WaivedByID: &waiver}
	resolver := &DetentionOccurrenceResolver{&Deps{Core: &resolvercore.Core{}}}

	approved, err := resolver.ApprovedBy(ctx, occurrence)
	require.NoError(t, err)
	require.NotNil(t, approved)
	assert.Equal(t, "Dana Ruiz", approved.Name)

	waived, err := resolver.WaivedBy(ctx, occurrence)
	require.NoError(t, err)
	require.NotNil(t, waived)
	assert.Equal(t, "Mia Chen", waived.Name)
}

func TestDetentionOccurrenceSignoffs_AreNullWithoutAnyoneOrAnyLoader(t *testing.T) {
	t.Parallel()

	resolver := &DetentionOccurrenceResolver{&Deps{Core: &resolvercore.Core{}}}
	unsigned := &gqlmodel.DetentionOccurrence{}

	approved, err := resolver.ApprovedBy(withUserLoader(t.Context(), &usersByID{}), unsigned)
	require.NoError(t, err)
	assert.Nil(t, approved)

	approver := pulid.MustNew("usr_").String()
	waived, err := resolver.WaivedBy(
		t.Context(),
		&gqlmodel.DetentionOccurrence{WaivedByID: &approver},
	)
	require.NoError(t, err)
	assert.Nil(t, waived)
}

func TestDetentionOccurrenceSignoffs_RefuseAnIDThatIsNotOne(t *testing.T) {
	t.Parallel()

	resolver := &DetentionOccurrenceResolver{&Deps{Core: &resolvercore.Core{}}}
	bad := "not-an-id"

	_, err := resolver.ApprovedBy(t.Context(), &gqlmodel.DetentionOccurrence{ApprovedByID: &bad})
	require.Error(t, err)
}

func TestDetentionOccurrenceDetailToModel_PricesARunningClockAsOfNow(t *testing.T) {
	t.Parallel()

	detail := detentionOccurrenceDetailToModel(&detentionservice.OccurrenceDetail{
		Occurrence: &detention.DetentionOccurrence{ID: pulid.MustNew("dto_"), IsOpen: true},
		CurrentlyBillable: &detentionservice.ClockPrice{
			BillableMinutes: 45,
			BillableAmount:  decimal.RequireFromString("56.25"),
			AsOf:            1_791_849_600,
		},
	})

	require.NotNil(t, detail.CurrentlyBillable)
	assert.Equal(t, &gqlmodel.DetentionClockPrice{
		BillableMinutes: 45,
		BillableAmount:  "56.25",
		AsOf:            1_791_849_600,
	}, detail.CurrentlyBillable)
}

func TestDetentionOccurrenceDetailToModel_HasNoLivePriceOnceTheClockStops(t *testing.T) {
	t.Parallel()

	detail := detentionOccurrenceDetailToModel(&detentionservice.OccurrenceDetail{
		Occurrence: &detention.DetentionOccurrence{ID: pulid.MustNew("dto_")},
	})

	assert.Nil(t, detail.CurrentlyBillable)
}
