package customertrackingservice

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/customertracking"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/stopeta"
	"github.com/emoss08/trenova/internal/modules/operations/ports/customertrackingports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const linkPrefix = "https://app.example.com/shipment-tracking/"

func (h *harness) issue(t *testing.T, sh *shipment.Shipment) string {
	t.Helper()
	issued, err := h.service.IssueLink(t.Context(), &customertrackingports.IssueLinkRequest{
		TenantInfo: testTenant,
		ShipmentID: sh.ID,
		IssuedByID: testTenant.UserID,
	})
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(issued.URL, linkPrefix))
	return strings.TrimPrefix(issued.URL, linkPrefix)
}

func TestPreview_ShowsOnlyTheFieldsTheCustomerAllows(t *testing.T) {
	h := newHarness()
	sh := pickedUpShipment(h)
	h.facts.descriptors = customertrackingports.ShipmentDescriptors{
		CarrierName: "Redline Freight",
		Equipment:   "53' Dry Van",
	}
	h.facts.holds = []customertrackingports.PublicHold{{Type: "CustomerHold", StartedAt: testNow}}
	token := h.issue(t, sh)

	view, err := h.service.PreviewByToken(t.Context(), token)
	require.NoError(t, err)

	assert.Equal(t, customertracking.DefaultFields(), view.Fields)
	assert.Equal(t, "PRO-77", view.ShipmentProNumber)
	assert.Equal(t, "BOL-1", view.BOL)
	assert.Equal(t, customertrackingports.PublicStatusInTransit, view.Status)
	assert.Empty(t, view.CarrierName)
	assert.Empty(t, view.Equipment)
	assert.Nil(t, view.Holds)
	require.Len(t, view.Stops, 2)
	assert.Equal(t, "Dallas", view.Stops[0].City)
	assert.Empty(t, view.Stops[0].LocationName)
	assert.Equal(t, at(-3600), view.Stops[0].DepartedAt)
	assert.Equal(t, 1, h.links.views)
}

func TestPreview_AddsWhatTheProfileTurnsOn(t *testing.T) {
	h := newHarness()
	sh := pickedUpShipment(h)
	h.notify(sh.CustomerID, func(p *customertracking.Profile) {
		p.VisibleFields = []customertracking.Field{
			customertracking.FieldCarrier,
			customertracking.FieldHolds,
			customertracking.FieldLocationNames,
			customertracking.FieldEta,
		}
	})
	h.facts.descriptors = customertrackingports.ShipmentDescriptors{CarrierName: "Redline Freight"}
	h.facts.holds = []customertrackingports.PublicHold{{Type: "CustomerHold", StartedAt: testNow}}
	estimated := testNow + 7200
	h.etas.rows = []*stopeta.StopETA{{
		StopID:           sh.Moves[0].Stops[1].ID,
		Verdict:          stopeta.VerdictAtRisk,
		EstimatedArrival: &estimated,
	}}
	token := h.issue(t, sh)

	view, err := h.service.PreviewByToken(t.Context(), token)
	require.NoError(t, err)

	assert.Equal(t, "Redline Freight", view.CarrierName)
	assert.Len(t, view.Holds, 1)
	assert.Empty(t, view.Status)
	assert.Empty(t, view.BOL)
	assert.Equal(t, "Memphis DC", view.Stops[1].LocationName)
	assert.Nil(t, view.Stops[0].WindowStart)
	assert.Nil(t, view.Stops[0].DepartedAt)
	assert.Equal(t, &estimated, view.Stops[1].EstimatedArrival)
	assert.Equal(t, "AtRisk", view.Stops[1].Running)
}

func TestPreview_StatusReadsDelayedWhenAStopIsAtRisk(t *testing.T) {
	h := newHarness()
	sh := pickedUpShipment(h)
	h.etas.rows = []*stopeta.StopETA{
		{StopID: sh.Moves[0].Stops[1].ID, Verdict: stopeta.VerdictLate},
	}
	token := h.issue(t, sh)

	view, err := h.service.PreviewByToken(t.Context(), token)
	require.NoError(t, err)

	assert.Equal(t, customertrackingports.PublicStatusDelayed, view.Status)
}

func TestPreview_EveryDeadLinkAnswersTheSame(t *testing.T) {
	cases := map[string]func(h *harness, sh *shipment.Shipment){
		"revoked": func(h *harness, sh *shipment.Shipment) {
			_, err := h.service.RevokeLink(t.Context(), &customertrackingports.RevokeLinkRequest{
				TenantInfo: testTenant,
				ShipmentID: sh.ID,
			})
			require.NoError(t, err)
		},
		"expired": func(h *harness, _ *shipment.Shipment) {
			h.now = testNow + int64(
				customertracking.DefaultLinkTTLDays,
			)*customertracking.SecondsPerDay
		},
		"moved to another customer": func(_ *harness, sh *shipment.Shipment) {
			sh.CustomerID = pulid.MustNew("cus_")
		},
		"links turned off": func(h *harness, sh *shipment.Shipment) {
			h.notify(sh.CustomerID, func(p *customertracking.Profile) { p.LinkEnabled = false })
		},
		"shipment gone": func(h *harness, sh *shipment.Shipment) {
			delete(h.shipments.rows, sh.ID)
		},
	}
	for name, kill := range cases {
		t.Run(name, func(t *testing.T) {
			h := newHarness()
			sh := pickedUpShipment(h)
			token := h.issue(t, sh)
			kill(h, sh)

			_, err := h.service.PreviewByToken(t.Context(), token)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "This tracking link is no longer valid")
		})
	}

	h := newHarness()
	_, err := h.service.PreviewByToken(t.Context(), "never-issued")
	assert.Contains(t, err.Error(), "This tracking link is no longer valid")
}

func TestDownloadPod_OnlyWhenTheCustomerSeesProofOfDelivery(t *testing.T) {
	h := newHarness()
	sh := pickedUpShipment(h)
	docID := pulid.MustNew("doc_")
	h.facts.pods = map[pulid.ID]*customertrackingports.PodDocument{
		sh.ID: {ShipmentID: sh.ID, DocumentID: docID, FiledAt: testNow},
	}
	token := h.issue(t, sh)

	_, err := h.service.DownloadPodByToken(t.Context(), token)
	require.Error(t, err)
	assert.Empty(t, h.documents.read)

	h.notify(sh.CustomerID, func(p *customertracking.Profile) {
		p.VisibleFields = []customertracking.Field{customertracking.FieldProofOfDelivery}
	})
	view, err := h.service.PreviewByToken(t.Context(), token)
	require.NoError(t, err)
	require.NotNil(t, view.ProofOfDelivery)
	assert.Equal(t, docID, view.ProofOfDelivery.DocumentID)

	result, err := h.service.DownloadPodByToken(t.Context(), token)
	require.NoError(t, err)
	assert.Equal(t, "pod.pdf", result.FileName)
	assert.Equal(t, []pulid.ID{docID}, h.documents.read)
}

func TestIssueLink_RefusesWhatItCannotShare(t *testing.T) {
	h := newHarness()
	canceled := h.addShipment(shipment.StatusCanceled,
		stopSpec{stopType: shipment.StopTypePickup, city: "Dallas", state: "TX"})
	_, err := h.service.IssueLink(t.Context(), &customertrackingports.IssueLinkRequest{
		TenantInfo: testTenant,
		ShipmentID: canceled.ID,
	})
	assert.Contains(t, err.Error(), "canceled")

	off := pickedUpShipment(h)
	h.notify(off.CustomerID, func(p *customertracking.Profile) { p.LinkEnabled = false })
	_, err = h.service.IssueLink(t.Context(), &customertrackingports.IssueLinkRequest{
		TenantInfo: testTenant,
		ShipmentID: off.ID,
	})
	assert.Contains(t, err.Error(), "turned off")

	h.service.appConfig.WebBaseURL = ""
	open := pickedUpShipment(h)
	_, err = h.service.IssueLink(t.Context(), &customertrackingports.IssueLinkRequest{
		TenantInfo: testTenant,
		ShipmentID: open.ID,
	})
	assert.Contains(t, err.Error(), "public base URL")

	_, err = h.service.IssueLink(t.Context(), &customertrackingports.IssueLinkRequest{
		TenantInfo:     testTenant,
		ShipmentID:     open.ID,
		RecipientEmail: "not-an-email",
	})
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
}

func TestIssueLink_EmailsTheLinkWhenAskedTo(t *testing.T) {
	h := newHarness()
	sh := pickedUpShipment(h)

	issued, err := h.service.IssueLink(t.Context(), &customertrackingports.IssueLinkRequest{
		TenantInfo:     testTenant,
		ShipmentID:     sh.ID,
		RecipientEmail: " lane@halstead.example ",
	})
	require.NoError(t, err)

	assert.True(t, issued.Emailed)
	assert.Equal(t, "lane@halstead.example", issued.Link.RecipientEmail)
	require.Len(t, h.email.sent, 1)
	assert.Equal(t, []string{"lane@halstead.example"}, h.email.sent[0].To)
	assert.Equal(
		t,
		"customer-tracking-link-"+issued.Link.ID.String(),
		h.email.sent[0].IdempotencyKey,
	)
}

func TestRevokeLink_OneOrAll(t *testing.T) {
	h := newHarness()
	sh := pickedUpShipment(h)
	h.issue(t, sh)
	h.issue(t, sh)

	revoked, err := h.service.RevokeLink(t.Context(), &customertrackingports.RevokeLinkRequest{
		TenantInfo: testTenant,
		ShipmentID: sh.ID,
		LinkID:     h.links.rows[0].ID,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, revoked)

	revoked, err = h.service.RevokeLink(t.Context(), &customertrackingports.RevokeLinkRequest{
		TenantInfo: testTenant,
		ShipmentID: sh.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, revoked)

	_, err = h.service.RevokeLink(t.Context(), &customertrackingports.RevokeLinkRequest{
		TenantInfo: testTenant,
		ShipmentID: sh.ID,
	})
	assert.True(t, errortypes.IsNotFoundError(err))
}

func TestUpdateProfile_CreatesThenGuardsTheVersion(t *testing.T) {
	h := newHarness()
	sh := pickedUpShipment(h)

	created, err := h.service.UpdateProfile(
		t.Context(),
		&customertrackingports.UpdateProfileRequest{
			TenantInfo: testTenant,
			Profile: &customertracking.Profile{
				CustomerID:    sh.CustomerID,
				LinkEnabled:   true,
				VisibleFields: []customertracking.Field{customertracking.FieldEta},
				NotifyDelayed: true,
				IncludeLink:   true,
				Recipients:    "Ops@Halstead.example",
			},
		},
	)
	require.NoError(t, err)
	assert.False(t, created.ID.IsNil())
	assert.Equal(t, "ops@halstead.example", created.Recipients)
	assert.Equal(t, testNow, *created.NotificationsEnabledAt)
	assert.Equal(t, customertracking.DefaultLinkTTLDays, created.LinkTTLDays)

	stale := int64(7)
	_, err = h.service.UpdateProfile(t.Context(), &customertrackingports.UpdateProfileRequest{
		TenantInfo:      testTenant,
		ExpectedVersion: &stale,
		Profile:         &customertracking.Profile{CustomerID: sh.CustomerID},
	})
	assert.Contains(t, err.Error(), "changed since you opened them")

	_, err = h.service.UpdateProfile(t.Context(), &customertrackingports.UpdateProfileRequest{
		TenantInfo: testTenant,
		Profile:    &customertracking.Profile{CustomerID: pulid.MustNew("cus_")},
	})
	assert.True(t, errortypes.IsNotFoundError(err))
}

func TestGetProfile_DefaultsUntilChanged(t *testing.T) {
	h := newHarness()
	sh := pickedUpShipment(h)

	profile, err := h.service.GetProfile(t.Context(), testTenant, sh.CustomerID)
	require.NoError(t, err)

	assert.True(t, profile.ID.IsNil())
	assert.True(t, profile.LinkEnabled)
	assert.Equal(t, customertracking.DefaultFields(), profile.VisibleFields)
	assert.False(t, profile.NotifiesAny())
}

func TestPreviewForShipment_ShowsWhatALinkWouldWithoutMintingOne(t *testing.T) {
	h := newHarness()
	sh := pickedUpShipment(h)
	h.notify(sh.CustomerID, func(p *customertracking.Profile) {
		p.VisibleFields = []customertracking.Field{
			customertracking.FieldStatus,
			customertracking.FieldCarrier,
		}
		p.LinkTTLDays = 10
	})
	h.facts.descriptors = customertrackingports.ShipmentDescriptors{
		CarrierName: "Redline Freight",
		Equipment:   "53' Dry Van",
	}

	view, err := h.service.PreviewForShipment(t.Context(), testTenant, sh.ID)
	require.NoError(t, err)

	token := h.issue(t, sh)
	viaLink, err := h.service.PreviewByToken(t.Context(), token)
	require.NoError(t, err)

	assert.Equal(t, viaLink.Fields, view.Fields)
	assert.Equal(t, viaLink.Stops, view.Stops)
	assert.Equal(t, viaLink.Status, view.Status)
	assert.Equal(t, "Redline Freight", view.CarrierName)
	assert.Empty(t, view.Equipment)
	assert.Equal(t, testNow+10*86400, view.ExpiresAt)
	assert.Len(t, h.links.rows, 1, "only the link issued above exists")
	assert.Equal(t, 1, h.links.views, "only the link's own open is counted")
}

func TestPreviewForShipment_ShowsTheDefaultsForACustomerWithoutSettings(t *testing.T) {
	h := newHarness()
	sh := pickedUpShipment(h)

	view, err := h.service.PreviewForShipment(t.Context(), testTenant, sh.ID)
	require.NoError(t, err)

	assert.Equal(t, customertracking.DefaultFields(), view.Fields)
	assert.Equal(t, "PRO-77", view.ShipmentProNumber)
	assert.Empty(t, h.links.rows)
}

func TestPreviewForShipment_UnknownShipmentIsNotFound(t *testing.T) {
	h := newHarness()

	_, err := h.service.PreviewForShipment(t.Context(), testTenant, pulid.MustNew("shp_"))
	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err))
}
