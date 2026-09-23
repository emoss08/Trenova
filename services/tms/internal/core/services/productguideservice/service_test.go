package productguideservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/productguide"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const fixture = `{
  "version": "sha256:test",
  "modules": [],
  "pages": [
    {
      "path": "/billing/configuration-files/rate-matrices",
      "name": "Rate matrices",
      "module": "billing",
      "breadcrumb": ["Billing", "Configuration files", "Rate matrices"],
      "description": "Published tariffs as grids.",
      "summary": "Rates by lane, zone and weight break.",
      "requires": [{"resource": "rate_matrix", "operation": "read"}],
      "capabilities": [],
      "aliases": ["tariff"],
      "tasks": [
        {"title": "Add a rate matrix", "keywords": ["new tariff"], "steps": ["Open it.", "Select New rate matrix."]},
        {"title": "Retire a rate matrix", "keywords": [], "steps": ["Set Status to Inactive."]}
      ],
      "notes": "",
      "related": [],
      "covers": [],
      "createAction": {"label": "New rate matrix", "query": {"panelType": "create"}},
      "inNavigation": true
    },
    {
      "path": "/shipment-management/shipments",
      "name": "Shipments",
      "module": "shipment",
      "breadcrumb": ["Shipment management", "Shipments"],
      "description": "Every load.",
      "summary": "The operations command center.",
      "requires": [{"resource": "shipment", "operation": "read"}],
      "capabilities": [],
      "aliases": ["loads", "freight"],
      "tasks": [{"title": "Create a shipment", "keywords": ["new load"], "steps": ["Select New shipment."]}],
      "notes": "",
      "related": [],
      "covers": [],
      "createAction": null,
      "inNavigation": true
    },
    {
      "path": "/dispatch/carrier-sourcing",
      "name": "Carrier sourcing",
      "module": "dispatch",
      "breadcrumb": ["Dispatch", "Carrier sourcing"],
      "description": "Find a carrier for a load.",
      "summary": "Shop carriers for brokered loads.",
      "requires": [],
      "capabilities": ["brokerage"],
      "aliases": [],
      "tasks": [],
      "notes": "",
      "related": [],
      "covers": [],
      "createAction": null,
      "inNavigation": true
    }
  ],
  "records": [
    {"entity": "shipment", "label": "Shipment", "path": "/shipment-management/shipments", "params": {"panelType": "edit", "panelEntityId": "{id}"}}
  ]
}`

type stubPermissions struct {
	serviceports.PermissionEngine

	allowed map[string]bool
	batches int
	checked []string
}

func (s *stubPermissions) CheckBatch(
	_ context.Context,
	req *serviceports.BatchPermissionCheckRequest,
) (*serviceports.BatchPermissionCheckResult, error) {
	s.batches++
	results := make([]serviceports.PermissionCheckResult, 0, len(req.Checks))
	for _, check := range req.Checks {
		key := check.Resource + ":" + string(check.Operation)
		s.checked = append(s.checked, key)
		results = append(results, serviceports.PermissionCheckResult{Allowed: s.allowed[key]})
	}

	return &serviceports.BatchPermissionCheckResult{Results: results}, nil
}

type stubOrganizations struct {
	repositories.OrganizationRepository

	capabilities repositories.OrganizationCapabilities
	reads        int
}

func (s *stubOrganizations) GetCapabilities(
	context.Context,
	repositories.GetOrganizationCapabilitiesRequest,
) (*repositories.OrganizationCapabilities, error) {
	s.reads++
	capabilities := s.capabilities

	return &capabilities, nil
}

type harness struct {
	service       *Service
	permissions   *stubPermissions
	organizations *stubOrganizations
}

func newHarness(t *testing.T, allowed ...string) *harness {
	t.Helper()

	catalog, err := productguide.Load([]byte(fixture))
	require.NoError(t, err)

	permissions := &stubPermissions{allowed: map[string]bool{}}
	for _, key := range allowed {
		permissions.allowed[key] = true
	}
	organizations := &stubOrganizations{}

	return &harness{
		service: newService(Params{
			Logger:        zap.NewNop(),
			Permissions:   permissions,
			Organizations: organizations,
		}, catalog),
		permissions:   permissions,
		organizations: organizations,
	}
}

func actor() *serviceports.RequestActor {
	return &serviceports.RequestActor{
		PrincipalType:  serviceports.PrincipalTypeUser,
		PrincipalID:    pulid.MustNew("usr_"),
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}
}

func (h *harness) search(t *testing.T, query string) []serviceports.ProductGuideMatch {
	t.Helper()

	matches, err := h.service.Search(t.Context(), &serviceports.ProductGuideSearchRequest{
		Actor:      actor(),
		TenantInfo: pagination.TenantInfo{},
		Query:      query,
	})
	require.NoError(t, err)

	return matches
}

func TestSearch_FindsThePageAndTheTaskAQuestionIsAbout(t *testing.T) {
	t.Parallel()

	h := newHarness(t, "rate_matrix:read", "shipment:read")
	matches := h.search(t, "how do I add a rate matrix")

	require.NotEmpty(t, matches)
	assert.Equal(t, "/billing/configuration-files/rate-matrices", matches[0].Page.Path)
	require.NotNil(t, matches[0].Task)
	assert.Equal(t, "Add a rate matrix", matches[0].Task.Title)
	assert.True(t, matches[0].CanOpen)
}

// People say "load", not "shipment"; the guide's aliases carry the words the
// page's name does not.
func TestSearch_UnderstandsTheWordsPeopleUse(t *testing.T) {
	t.Parallel()

	h := newHarness(t, "rate_matrix:read", "shipment:read")
	matches := h.search(t, "where do I enter a new load")

	require.NotEmpty(t, matches)
	assert.Equal(t, "/shipment-management/shipments", matches[0].Page.Path)
}

// A page the person cannot open is still the answer, named with what they
// would need, so the agent can say who to ask rather than that nothing exists.
func TestSearch_NamesWhatIsMissingRatherThanHidingThePage(t *testing.T) {
	t.Parallel()

	h := newHarness(t, "shipment:read")
	matches := h.search(t, "add a rate matrix")

	require.NotEmpty(t, matches)
	assert.Equal(t, "/billing/configuration-files/rate-matrices", matches[0].Page.Path)
	assert.False(t, matches[0].CanOpen)
	require.Len(t, matches[0].Missing, 1)
	assert.Contains(t, matches[0].Missing[0], "read permission")
}

func TestSearch_ChecksOnlyThePagesItAnswersWith(t *testing.T) {
	t.Parallel()

	h := newHarness(t, "rate_matrix:read", "shipment:read")
	h.search(t, "rate matrix")

	assert.Equal(t, 1, h.permissions.batches, "one batch per answer")
	assert.NotContains(t, h.permissions.checked, "shipment:read",
		"a page not in the answer is not checked")
	assert.Zero(t, h.organizations.reads,
		"no page in the answer depends on a capability, so none is read")
}

func TestSearch_AnOrganizationWithoutBrokerageCannotOpenBrokeragePages(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	matches := h.search(t, "carrier sourcing")

	require.NotEmpty(t, matches)
	assert.Equal(t, "/dispatch/carrier-sourcing", matches[0].Page.Path)
	assert.False(t, matches[0].CanOpen)
	assert.Equal(t, []string{"brokerage to be turned on for the organization"}, matches[0].Missing)

	h.organizations.capabilities.BrokerageEnabled = true
	matches = h.search(t, "carrier sourcing")
	require.NotEmpty(t, matches)
	assert.True(t, matches[0].CanOpen)
}

func TestSearch_AnswersNothingForAQuestionTheGuideDoesNotCover(t *testing.T) {
	t.Parallel()

	h := newHarness(t, "rate_matrix:read", "shipment:read")

	assert.Empty(t, h.search(t, "photosynthesis"))
	assert.Zero(t, h.permissions.batches, "nothing to answer with, nothing to check")
}

// Asked about one page, the answer comes from that page alone.
func TestSearch_AnswersFromOnePageWhenAskedAboutIt(t *testing.T) {
	t.Parallel()

	h := newHarness(t, "rate_matrix:read", "shipment:read")
	matches, err := h.service.Search(t.Context(), &serviceports.ProductGuideSearchRequest{
		Actor: actor(),
		Query: "retire",
		Page:  "/billing/configuration-files/rate-matrices",
	})
	require.NoError(t, err)

	require.Len(t, matches, 1)
	require.NotNil(t, matches[0].Task)
	assert.Equal(t, "Retire a rate matrix", matches[0].Task.Title)
}

func TestSearch_RefusesToAnswerForNobody(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	_, err := h.service.Search(t.Context(), &serviceports.ProductGuideSearchRequest{Query: "shipments"})

	require.Error(t, err)
}

func TestDestination_OpensAPageTheCreateFormOrARecord(t *testing.T) {
	t.Parallel()

	h := newHarness(t, "rate_matrix:read", "shipment:read")

	page, err := h.service.Destination(t.Context(), &serviceports.ProductGuideDestinationRequest{
		Actor: actor(),
		Page:  "/billing/configuration-files/rate-matrices",
	})
	require.NoError(t, err)
	assert.Equal(t, "/billing/configuration-files/rate-matrices", page.Path)
	assert.Equal(t, "Rate matrices", page.Label)

	create, err := h.service.Destination(t.Context(), &serviceports.ProductGuideDestinationRequest{
		Actor:  actor(),
		Page:   "/billing/configuration-files/rate-matrices",
		Create: true,
	})
	require.NoError(t, err)
	assert.Equal(t, "/billing/configuration-files/rate-matrices?panelType=create", create.Path)

	record, err := h.service.Destination(t.Context(), &serviceports.ProductGuideDestinationRequest{
		Actor:    actor(),
		Entity:   "shipment",
		RecordID: "shp_1",
	})
	require.NoError(t, err)
	assert.Equal(t, "/shipment-management/shipments?panelEntityId=shp_1&panelType=edit", record.Path)
	assert.Equal(t, "/shipment-management/shipments", record.Page.Path)
}

func TestDestination_RefusesWhatItCannotOrMayNotOpen(t *testing.T) {
	t.Parallel()

	h := newHarness(t, "shipment:read")

	tests := []struct {
		name string
		req  serviceports.ProductGuideDestinationRequest
	}{
		{name: "nowhere", req: serviceports.ProductGuideDestinationRequest{}},
		{name: "a page that does not exist", req: serviceports.ProductGuideDestinationRequest{
			Page: "/billing/made-up",
		}},
		{name: "a create form the page does not have", req: serviceports.ProductGuideDestinationRequest{
			Page:   "/shipment-management/shipments",
			Create: true,
		}},
		{name: "a record kind with no page", req: serviceports.ProductGuideDestinationRequest{
			Entity:   "unicorn",
			RecordID: "uni_1",
		}},
		{name: "a record with no id", req: serviceports.ProductGuideDestinationRequest{
			Entity: "shipment",
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := tt.req
			req.Actor = actor()
			_, err := newHarness(t, "shipment:read").service.Destination(t.Context(), &req)

			var validation *errortypes.Error
			require.ErrorAs(t, err, &validation)
		})
	}

	_, err := h.service.Destination(t.Context(), &serviceports.ProductGuideDestinationRequest{
		Actor: actor(),
		Page:  "/billing/configuration-files/rate-matrices",
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err), "a page the person may not open is refused")
}
