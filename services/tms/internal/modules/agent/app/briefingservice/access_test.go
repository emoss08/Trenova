package briefingservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/briefing"
	"github.com/emoss08/trenova/internal/modules/agent/ports/briefingports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type failingBatch struct {
	mocks.AllowAllPermissionEngine
}

func (*failingBatch) CheckBatch(
	context.Context,
	*permissionports.BatchPermissionCheckRequest,
) (*permissionports.BatchPermissionCheckResult, error) {
	return nil, errors.New("permission store unavailable")
}

type fakePages struct {
	briefingports.BriefingRepository

	page  *briefing.Briefing
	pages []*briefing.Briefing
}

func (f *fakePages) GetForDay(
	context.Context,
	briefingports.GetBriefingForDayRequest,
) (*briefing.Briefing, error) {
	return f.page, nil
}

func (f *fakePages) GetByID(
	context.Context,
	briefingports.GetBriefingByIDRequest,
) (*briefing.Briefing, error) {
	return f.page, nil
}

func (f *fakePages) List(
	context.Context,
	briefingports.ListBriefingsQuery,
) ([]*briefing.Briefing, error) {
	return f.pages, nil
}

func (f *fakePages) MarkRead(
	context.Context,
	briefingports.GetBriefingByIDRequest,
	int64,
) (*briefing.Briefing, error) {
	return f.page, nil
}

func everySectionPage() *briefing.Briefing {
	section := func(key briefing.SectionKey, title string, items ...briefing.Item) briefing.Section {
		return briefing.Section{
			Key:     key,
			Title:   title,
			Summary: title + " in figures.",
			Body:    title + " in a model's words.",
			Items:   items,
		}
	}

	return &briefing.Briefing{
		ID:           pulid.MustNew("brf_"),
		RoleKey:      briefing.RoleGeneral,
		BriefingDate: "2026-10-23",
		Status:       briefing.StatusReady,
		Headline:     "Twelve items open and $4,210 received yesterday.",
		Narrated:     true,
		Sections: []briefing.Section{
			section(briefing.SectionAttention, "Needs attention",
				briefing.Item{Label: "Service failure", Value: "4",
					Path: "/desk/watchtower?kinds=ServiceFailure"},
				briefing.Item{Label: "EDI quarantined", Value: "8",
					Path: "/desk/watchtower?kinds=EDIInboundQuarantined"},
			),
			section(briefing.SectionToday, "Today"),
			section(briefing.SectionCoverage, "The next day"),
			section(briefing.SectionDecisions, "Waiting on you"),
			section(briefing.SectionCompliance, "Credentials"),
			section(briefing.SectionBilling, "Billing queue"),
			section(briefing.SectionCash, "Cash received yesterday"),
			section(briefing.SectionExceptions, "Critical",
				briefing.Item{Label: "Tender exhausted", Value: "1",
					Path: "/desk/watchtower?kinds=TenderExhausted"},
			),
		},
	}
}

func gatedService(page *briefing.Briefing, denied ...permission.Resource) *Service {
	names := make([]string, 0, len(denied))
	for _, resource := range denied {
		names = append(names, resource.String())
	}

	return &Service{
		l:           zap.NewNop(),
		repo:        &fakePages{page: page, pages: []*briefing.Briefing{page}},
		permissions: &mocks.DenyResourcesPermissionEngine{Denied: names},
		now:         func() int64 { return 1_761_200_000 },
	}
}

func reader() *permissionports.RequestActor {
	return &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeUser,
		PrincipalID:    pulid.MustNew("usr_"),
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}
}

func today(t *testing.T, svc *Service) *briefing.Briefing {
	t.Helper()

	page, err := svc.Today(t.Context(), briefingports.GetBriefingRequest{
		TenantInfo:   pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		BriefingDate: "2026-10-23",
	}, reader())
	require.NoError(t, err)

	return page
}

func keys(page *briefing.Briefing) []briefing.SectionKey {
	out := make([]briefing.SectionKey, 0, len(page.Sections))
	for _, section := range page.Sections {
		out = append(out, section.Key)
	}

	return out
}

func TestToday_AReaderOfEverySectionReadsTheWholeNarratedPage(t *testing.T) {
	t.Parallel()

	page := today(t, gatedService(everySectionPage()))

	assert.Len(t, page.Sections, 8)
	assert.Empty(t, page.WithheldSections)
	assert.NotNil(t, page.WithheldSections, "the list is never null")
	assert.Equal(t, "Twelve items open and $4,210 received yesterday.", page.Headline)
	assert.True(t, page.Narrated)
	assert.Equal(t, "Today in a model's words.", page.Sections[1].Read())
}

/*
Each section is shown only to a reader of the records it counts.

The page was written once for everybody with briefing read, so without this a
dispatcher opened the Desk to the morning's cash and billing figures. A reader
missing a section's permission loses that section, the headline and every
model's sentence (written about the whole page, they may cite it), and keeps
the other sections in their computed wording.
*/
func TestToday_AReaderMissingASectionsPermissionDoesNotSeeIt(t *testing.T) {
	t.Parallel()

	cases := []struct {
		resource permission.Resource
		left     []briefing.SectionKey
	}{
		{permission.ResourceWatchtower, []briefing.SectionKey{
			briefing.SectionAttention, briefing.SectionExceptions,
		}},
		{permission.ResourceShipment, []briefing.SectionKey{briefing.SectionToday}},
		{permission.ResourceShipmentMove, []briefing.SectionKey{briefing.SectionCoverage}},
		{permission.ResourceAgentProposal, []briefing.SectionKey{briefing.SectionDecisions}},
		{permission.ResourceWorkerCredential, []briefing.SectionKey{briefing.SectionCompliance}},
		{permission.ResourceBillingQueue, []briefing.SectionKey{briefing.SectionBilling}},
		{permission.ResourceCustomerPayment, []briefing.SectionKey{briefing.SectionCash}},
	}
	for _, tc := range cases {
		t.Run(tc.resource.String(), func(t *testing.T) {
			t.Parallel()

			page := today(t, gatedService(everySectionPage(), tc.resource))

			for _, key := range tc.left {
				assert.NotContains(t, keys(page), key)
			}
			assert.Len(t, page.Sections, 8-len(tc.left))
			withheld := make([]briefing.SectionKey, 0, len(page.WithheldSections))
			for _, section := range page.WithheldSections {
				assert.False(t, section.Partial)
				assert.NotEmpty(t, section.Title)
				withheld = append(withheld, section.Key)
			}
			assert.ElementsMatch(t, tc.left, withheld)
			assert.Empty(t, page.Headline, "the headline may cite a withheld section")
			assert.False(t, page.Narrated)
			for _, section := range page.Sections {
				assert.Empty(t, section.Body, "%s keeps only its computed wording", section.Key)
				assert.Equal(t, section.Title+" in figures.", section.Read())
			}
		})
	}
}

func TestToday_AWatchtowerSectionKeepsOnlyTheKindsTheReaderMayRead(t *testing.T) {
	t.Parallel()

	page := today(t, gatedService(everySectionPage(), permission.ResourceEDI))

	attention := page.Sections[0]
	require.Equal(t, briefing.SectionAttention, attention.Key)
	require.Len(t, attention.Items, 1)
	assert.Equal(t, "Service failure", attention.Items[0].Label)
	assert.Empty(t, attention.Summary, "its wording counted the EDI files too")
	assert.Empty(t, attention.Body)
	assert.Equal(t, []briefing.WithheldSection{{
		Key: briefing.SectionAttention, Title: "Needs attention", Partial: true,
	}}, page.WithheldSections)
	assert.Empty(t, page.Headline)
	assert.Len(t, page.Sections, 8, "the critical section's one kind is still readable")
}

func TestToday_AWatchtowerLineNamingNoKindIsWithheld(t *testing.T) {
	t.Parallel()

	source := everySectionPage()
	source.Sections[7].Items = append(source.Sections[7].Items, briefing.Item{
		Label: "Something", Value: "2", Path: "/desk/watchtower",
	})

	page := today(t, gatedService(source))

	assert.Len(t, page.Sections[7].Items, 1)
	assert.Equal(t, []briefing.WithheldSection{{
		Key: briefing.SectionExceptions, Title: "Critical", Partial: true,
	}}, page.WithheldSections)
}

func TestToday_ASectionNobodyMappedIsWithheld(t *testing.T) {
	t.Parallel()

	source := everySectionPage()
	source.Sections = append(source.Sections, briefing.Section{
		Key: briefing.SectionKey("Payroll"), Title: "Payroll", Summary: "Payroll runs Friday.",
	})

	page := today(t, gatedService(source))

	assert.NotContains(t, keys(page), briefing.SectionKey("Payroll"))
	assert.Equal(t, []briefing.WithheldSection{{
		Key: briefing.SectionKey("Payroll"), Title: "Payroll",
	}}, page.WithheldSections)
}

func TestToday_NoPageWrittenYetIsNoPage(t *testing.T) {
	t.Parallel()

	assert.Nil(t, today(t, gatedService(nil)))
}

// Every other read of a page is cut the same way, so no path shows what Today
// withholds.
func TestEveryRead_CutsThePageToTheReader(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	cut := func(t *testing.T, page *briefing.Briefing) {
		t.Helper()
		assert.NotContains(t, keys(page), briefing.SectionCash)
		assert.Equal(t, briefing.SectionCash, page.WithheldSections[0].Key)
	}

	t.Run("get", func(t *testing.T) {
		t.Parallel()
		page, err := gatedService(everySectionPage(), permission.ResourceCustomerPayment).
			Get(t.Context(), tenant, pulid.MustNew("brf_"), reader())
		require.NoError(t, err)
		cut(t, page)
	})
	t.Run("list", func(t *testing.T) {
		t.Parallel()
		pages, err := gatedService(everySectionPage(), permission.ResourceCustomerPayment).
			List(t.Context(), briefingports.ListBriefingsRequest{TenantInfo: tenant}, reader())
		require.NoError(t, err)
		require.Len(t, pages, 1)
		cut(t, pages[0])
	})
	t.Run("mark read", func(t *testing.T) {
		t.Parallel()
		page, err := gatedService(everySectionPage(), permission.ResourceCustomerPayment).
			MarkRead(t.Context(), tenant, pulid.MustNew("brf_"), reader())
		require.NoError(t, err)
		cut(t, page)
	})
	t.Run("regenerate", func(t *testing.T) {
		t.Parallel()
		svc := gatedService(nil, permission.ResourceCustomerPayment)
		svc.days = &fakeDays{result: &briefingports.WriteBriefingResult{
			Briefings: []*briefing.Briefing{everySectionPage()},
		}}
		page, err := svc.Regenerate(t.Context(), briefingports.GetBriefingRequest{
			TenantInfo: tenant,
		}, reader())
		require.NoError(t, err)
		cut(t, page)
	})
}

func TestToday_AFailedPermissionCheckIsAnError(t *testing.T) {
	t.Parallel()

	svc := gatedService(everySectionPage())
	svc.permissions = &failingBatch{}

	_, err := svc.Today(t.Context(), briefingports.GetBriefingRequest{
		BriefingDate: "2026-10-23",
	}, reader())
	require.Error(t, err)
}
