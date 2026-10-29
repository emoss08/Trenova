//go:build integration

package customertrackingrepository_test

import (
	"context"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document/documentdb"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/customertrackingrepository"
	"github.com/emoss08/trenova/internal/modules/operations/domain/customertracking"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmenthold"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmenthold/shipmentholddb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/customertrackingports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype/documenttypedb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/holdreason"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/emoss08/trenova/shared/tokenutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type harness struct {
	ctx           context.Context
	db            *keel.DB
	dsn           string
	tenant        pagination.TenantInfo
	data          *seedtest.TestData
	fixture       *testutil.ShipmentIntegrationFixture
	profiles      customertrackingports.ProfileRepository
	links         customertrackingports.LinkRepository
	notifications customertrackingports.NotificationRepository
	facts         customertrackingports.FactRepository
	now           int64
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	ctx, db, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	tenant := pagination.TenantInfo{OrgID: data.Organization.ID, BuID: data.BusinessUnit.ID}
	params := customertrackingrepository.Params{
		DB:     postgres.NewTestConnection(db),
		Logger: zap.NewNop(),
	}
	return &harness{
		ctx:           dbscope.WithTenant(ctx, tenant.DBTenant()),
		db:            db,
		dsn:           dsn,
		tenant:        tenant,
		data:          data,
		fixture:       testutil.SeedShipmentIntegrationFixture(t, ctx, db, data, tenant),
		profiles:      customertrackingrepository.NewProfiles(params),
		links:         customertrackingrepository.NewLinks(params),
		notifications: customertrackingrepository.NewNotifications(params),
		facts:         customertrackingrepository.NewFacts(params),
		now:           timeutils.NowUnix(),
	}
}

func (h *harness) shipment(
	t *testing.T,
	code string,
	status shipmentvocab.Status,
) *testutil.ShipmentGraph {
	t.Helper()
	graph := testutil.CreateShipmentGraph(
		t,
		h.ctx,
		h.db,
		h.fixture,
		h.tenant,
		testutil.ShipmentGraphParams{
			BOL:          "BOL-" + code,
			ProNumber:    "PRO-" + code,
			ShipmentID:   pulid.MustNew("shp_"),
			MoveStatuses: []shipmentvocab.MoveStatus{shipmentvocab.MoveStatusNew},
		},
	)
	sp := shipmentdb.Shipments
	_, err := sp.Where(sp.ID.Eq(graph.Shipment.ID)).Update(h.ctx, h.db, sp.Status.To(status))
	require.NoError(t, err)
	return graph
}

func (h *harness) profile(
	t *testing.T,
	customerID pulid.ID,
	notify bool,
) *customertracking.Profile {
	t.Helper()
	profile := customertracking.DefaultProfile(h.tenant.OrgID, h.tenant.BuID, customerID)
	profile.ID = customertracking.NewProfileID()
	profile.CreatedAt, profile.UpdatedAt = h.now, h.now
	if notify {
		profile.NotifyDelivered = true
		enabled := h.now - 3600
		profile.NotificationsEnabledAt = &enabled
	}
	require.NoError(t, h.profiles.Create(h.ctx, profile))
	return profile
}

func TestProfilesRoundTripTheirFieldsAndGuardTheVersion(t *testing.T) {
	h := newHarness(t)
	graph := h.shipment(t, "PR-1", shipmentvocab.StatusInTransit)
	customerID := graph.Shipment.CustomerID

	missing, err := h.profiles.GetByCustomer(h.ctx, h.tenant, customerID)
	require.NoError(t, err)
	assert.Nil(t, missing)

	created := h.profile(t, customerID, false)
	created.VisibleFields = []customertracking.Field{
		customertracking.FieldStatus,
		customertracking.FieldProofOfDelivery,
	}
	created.NotifyPickedUp = true
	enabled := h.now
	created.NotificationsEnabledAt = &enabled
	created.Recipients = "ops@example.com"
	require.NoError(t, h.profiles.Update(h.ctx, created))
	assert.Equal(t, int64(1), created.Version)

	read, err := h.profiles.GetByCustomer(h.ctx, h.tenant, customerID)
	require.NoError(t, err)
	assert.Equal(t, created.VisibleFields, read.VisibleFields)
	assert.True(t, read.NotifyPickedUp)
	assert.Equal(t, "ops@example.com", read.Recipients)

	created.Version = 0
	err = h.profiles.Update(h.ctx, created)
	require.Error(t, err)

	listed, err := h.profiles.ListByCustomers(h.ctx, h.tenant, []pulid.ID{customerID})
	require.NoError(t, err)
	assert.Len(t, listed, 1)
}

func TestAProfileCreatedWithLinksOffStoresFalse(t *testing.T) {
	h := newHarness(t)
	graph := h.shipment(t, "PR-OFF", shipmentvocab.StatusInTransit)

	profile := customertracking.DefaultProfile(
		h.tenant.OrgID,
		h.tenant.BuID,
		graph.Shipment.CustomerID,
	)
	profile.ID = customertracking.NewProfileID()
	profile.CreatedAt, profile.UpdatedAt = h.now, h.now
	profile.LinkEnabled = false
	profile.IncludeLink = false
	require.NoError(t, h.profiles.Create(h.ctx, profile))

	read, err := h.profiles.GetByCustomer(h.ctx, h.tenant, graph.Shipment.CustomerID)
	require.NoError(t, err)
	require.NotNil(t, read)
	assert.False(t, read.LinkEnabled)
	assert.False(t, read.IncludeLink)
}

func TestTheDatabaseRefusesAProfileWithRulesButNoStartTime(t *testing.T) {
	h := newHarness(t)
	graph := h.shipment(t, "PR-2", shipmentvocab.StatusInTransit)
	profile := customertracking.DefaultProfile(
		h.tenant.OrgID,
		h.tenant.BuID,
		graph.Shipment.CustomerID,
	)
	profile.ID = customertracking.NewProfileID()
	profile.CreatedAt, profile.UpdatedAt = h.now, h.now
	profile.NotifyDelivered = true

	require.Error(t, h.profiles.Create(h.ctx, profile))
}

func TestLinksResolveByHashAndRevoke(t *testing.T) {
	h := newHarness(t)
	graph := h.shipment(t, "LK-1", shipmentvocab.StatusInTransit)
	newLink := func() (*customertracking.Link, string) {
		token, hash, err := tokenutils.New()
		require.NoError(t, err)
		link := &customertracking.Link{
			ID:             customertracking.NewLinkID(),
			OrganizationID: h.tenant.OrgID,
			BusinessUnitID: h.tenant.BuID,
			ShipmentID:     graph.Shipment.ID,
			CustomerID:     graph.Shipment.CustomerID,
			TokenHash:      hash,
			Purpose:        customertracking.LinkPurposeShared,
			ExpiresAt:      h.now + 3600,
			CreatedAt:      h.now,
			UpdatedAt:      h.now,
		}
		require.NoError(t, h.links.Create(h.ctx, link))
		return link, token
	}
	first, token := newLink()
	second, _ := newLink()

	found, err := h.links.GetByHash(context.Background(), tokenutils.Hash(token))
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, first.ID, found.ID)

	require.NoError(t, h.links.RecordView(h.ctx, h.tenant, first.ID, h.now))
	require.NoError(t, h.links.RecordView(h.ctx, h.tenant, first.ID, h.now+5))
	viewed, err := h.links.GetByID(h.ctx, h.tenant, first.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(2), viewed.ViewCount)
	assert.Equal(t, h.now+5, *viewed.LastViewedAt)

	revoked, err := h.links.Revoke(h.ctx, &customertrackingports.RevokeLinksRequest{
		TenantInfo: h.tenant,
		ShipmentID: graph.Shipment.ID,
		LinkID:     second.ID,
		At:         h.now,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, revoked)
	revoked, err = h.links.Revoke(h.ctx, &customertrackingports.RevokeLinksRequest{
		TenantInfo: h.tenant,
		ShipmentID: graph.Shipment.ID,
		At:         h.now,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, revoked)

	all, err := h.links.ListForShipment(h.ctx, h.tenant, graph.Shipment.ID)
	require.NoError(t, err)
	assert.Len(t, all, 2)

	purged, err := h.links.PurgeDead(context.Background(), h.now+1, 10)
	require.NoError(t, err)
	assert.Equal(t, 2, purged)
}

func TestNotificationsAreOwedOnceAndSettledOnce(t *testing.T) {
	h := newHarness(t)
	graph := h.shipment(t, "NT-1", shipmentvocab.StatusInTransit)
	row := func() *customertracking.Notification {
		return &customertracking.Notification{
			ID:             customertracking.NewNotificationID(),
			OrganizationID: h.tenant.OrgID,
			BusinessUnitID: h.tenant.BuID,
			ShipmentID:     graph.Shipment.ID,
			CustomerID:     graph.Shipment.CustomerID,
			Event:          customertracking.EventDelivered,
			OccurredAt:     h.now - 60,
			Status:         customertracking.NotificationStatusPending,
			Recipients:     []string{},
			NextAttemptAt:  h.now,
			CreatedAt:      h.now,
			UpdatedAt:      h.now,
		}
	}
	first := row()
	inserted, err := h.notifications.InsertOnce(h.ctx, first)
	require.NoError(t, err)
	assert.True(t, inserted)
	inserted, err = h.notifications.InsertOnce(h.ctx, row())
	require.NoError(t, err)
	assert.False(t, inserted)

	due, err := h.notifications.ListDue(h.ctx, h.tenant, h.now, 10)
	require.NoError(t, err)
	require.Len(t, due, 1)
	tenants, err := h.notifications.ListTenantsWithDue(context.Background(), h.now)
	require.NoError(t, err)
	assert.Contains(t, tenants, h.tenant)

	sentAt := h.now
	outcome := &customertrackingports.DeliveryOutcome{
		TenantInfo:     h.tenant,
		ID:             first.ID,
		Status:         customertracking.NotificationStatusSent,
		Recipients:     []string{"ops@example.com"},
		EmailMessageID: pulid.MustNew("emsg_"),
		Attempts:       1,
		NextAttemptAt:  h.now,
		SentAt:         &sentAt,
		At:             h.now,
	}
	settled, err := h.notifications.Record(h.ctx, outcome)
	require.NoError(t, err)
	assert.True(t, settled)
	settled, err = h.notifications.Record(h.ctx, outcome)
	require.NoError(t, err)
	assert.False(t, settled)

	stored, err := h.notifications.GetByID(h.ctx, h.tenant, first.ID)
	require.NoError(t, err)
	assert.Equal(t, customertracking.NotificationStatusSent, stored.Status)
	assert.Equal(t, []string{"ops@example.com"}, stored.Recipients)
	assert.Equal(t, outcome.EmailMessageID, stored.EmailMessageID)

	due, err = h.notifications.ListDue(h.ctx, h.tenant, h.now, 10)
	require.NoError(t, err)
	assert.Empty(t, due)
}

func (h *harness) touch(t *testing.T, id pulid.ID, sets ...keel.Assignment) {
	t.Helper()
	sp := shipmentdb.Shipments
	_, err := sp.Where(sp.ID.Eq(id)).Update(h.ctx, h.db, sets...)
	require.NoError(t, err)
}

func (h *harness) podType(t *testing.T) pulid.ID {
	t.Helper()
	docType := &documenttype.DocumentType{
		ID:             pulid.MustNew("dt_"),
		OrganizationID: h.tenant.OrgID,
		BusinessUnitID: h.tenant.BuID,
		Code:           documenttype.CodePOD,
		Name:           "Proof of delivery",
	}
	require.NoError(t, documenttypedb.DocumentTypes.Create(h.ctx, h.db, docType))
	return docType.ID
}

func (h *harness) attach(
	t *testing.T,
	typeID, shipmentID pulid.ID,
	name string,
	createdAt int64,
	status document.Status,
) pulid.ID {
	t.Helper()
	entity := &document.Document{
		ID:               pulid.MustNew("doc_"),
		OrganizationID:   h.tenant.OrgID,
		BusinessUnitID:   h.tenant.BuID,
		LineageID:        pulid.MustNew("dln_"),
		IsCurrentVersion: true,
		FileName:         name,
		OriginalName:     name,
		FileSize:         128,
		FileType:         "application/pdf",
		StoragePath:      "test/" + name,
		Status:           status,
		ResourceID:       shipmentID.String(),
		ResourceType:     "shipment",
		UploadedByID:     h.data.User.ID,
		DocumentTypeID:   &typeID,
	}
	if status == document.StatusRejected {
		entity.RejectedByID = h.data.User.ID
		entity.RejectedAt = &createdAt
		entity.RejectionReason = "Illegible"
	}
	docs := documentdb.Documents
	require.NoError(t, docs.Create(h.ctx, h.db, entity))
	_, err := docs.Where(docs.ID.Eq(entity.ID)).
		Update(h.ctx, h.db, docs.CreatedAt.To(createdAt))
	require.NoError(t, err)
	return entity.ID
}

func TestCandidatesAreLiveShipmentsOfNotifyingCustomers(t *testing.T) {
	h := newHarness(t)
	since := h.now - 3600
	old := since - 86400
	sp := shipmentdb.Shipments

	recent := h.shipment(t, "CD-1", shipmentvocab.StatusCompleted)
	moving := h.shipment(t, "CD-2", shipmentvocab.StatusInTransit)
	delivered := h.shipment(t, "CD-3", shipmentvocab.StatusCompleted)
	podded := h.shipment(t, "CD-4", shipmentvocab.StatusCompleted)
	stale := h.shipment(t, "CD-5", shipmentvocab.StatusCompleted)
	canceled := h.shipment(t, "CD-6", shipmentvocab.StatusCanceled)
	for _, graph := range []*testutil.ShipmentGraph{moving, delivered, podded, stale, canceled} {
		require.Equal(t, recent.Shipment.CustomerID, graph.Shipment.CustomerID)
	}
	h.profile(t, recent.Shipment.CustomerID, true)

	h.touch(t, recent.Shipment.ID, sp.UpdatedAt.To(h.now-60))
	h.touch(t, moving.Shipment.ID, sp.UpdatedAt.To(old-1))
	h.touch(t, delivered.Shipment.ID,
		sp.UpdatedAt.To(old-2), sp.ActualDeliveryDate.To(h.now-600))
	h.touch(t, podded.Shipment.ID, sp.UpdatedAt.To(old-3), sp.ActualDeliveryDate.To(old))
	h.touch(t, stale.Shipment.ID, sp.UpdatedAt.To(old-4), sp.ActualDeliveryDate.To(old))
	h.touch(t, canceled.Shipment.ID, sp.UpdatedAt.To(h.now-30))
	podType := h.podType(t)
	h.attach(t, podType, podded.Shipment.ID, "pod.pdf", h.now-300, document.StatusActive)
	h.attach(t, podType, stale.Shipment.ID, "old.pdf", old, document.StatusActive)
	h.attach(t, podType, stale.Shipment.ID, "rejected.pdf", h.now-300, document.StatusRejected)

	tenants, err := h.facts.ListNotifyingTenants(context.Background())
	require.NoError(t, err)
	assert.Contains(t, tenants, h.tenant)

	ids, err := h.facts.ListCandidateShipments(
		h.ctx,
		&customertrackingports.CandidateShipmentsRequest{
			TenantInfo: h.tenant,
			Since:      since,
		},
	)
	require.NoError(t, err)
	assert.NotContains(t, ids, stale.Shipment.ID, "nothing happened to it since the window opened")
	assert.NotContains(t, ids, canceled.Shipment.ID)

	mine := map[pulid.ID]bool{
		recent.Shipment.ID:    true,
		moving.Shipment.ID:    true,
		delivered.Shipment.ID: true,
		podded.Shipment.ID:    true,
	}
	ordered := make([]pulid.ID, 0, len(mine))
	for _, id := range ids {
		if mine[id] {
			ordered = append(ordered, id)
		}
	}
	assert.Equal(t, []pulid.ID{
		recent.Shipment.ID,
		moving.Shipment.ID,
		delivered.Shipment.ID,
		podded.Shipment.ID,
	}, ordered, "every way to owe an email, most recently touched first, each once")
}

func TestLatestPodsIsTheFirstCurrentPodOnFile(t *testing.T) {
	h := newHarness(t)
	graph := h.shipment(t, "PD-1", shipmentvocab.StatusCompleted)
	podType := h.podType(t)
	attach := func(name string, createdAt int64, status document.Status) pulid.ID {
		return h.attach(t, podType, graph.Shipment.ID, name, createdAt, status)
	}
	attach("rejected.pdf", h.now-900, document.StatusRejected)
	first := attach("first.pdf", h.now-600, document.StatusActive)
	attach("second.pdf", h.now-300, document.StatusActive)

	pods, err := h.facts.LatestPods(h.ctx, h.tenant, []pulid.ID{graph.Shipment.ID})
	require.NoError(t, err)
	pod := pods[graph.Shipment.ID]
	require.NotNil(t, pod)
	assert.Equal(t, first, pod.DocumentID)
	assert.Equal(t, "first.pdf", pod.FileName)
	assert.Equal(t, h.now-600, pod.FiledAt)
}

func TestOnlyCustomerVisibleHoldsAndNotesAreShown(t *testing.T) {
	h := newHarness(t)
	graph := h.shipment(t, "VS-1", shipmentvocab.StatusInTransit)
	hold := func(holdType holdreason.HoldType, visible, released bool) {
		entity := &shipmenthold.ShipmentHold{
			ID:                pulid.MustNew("shh_"),
			ShipmentID:        graph.Shipment.ID,
			OrganizationID:    h.tenant.OrgID,
			BusinessUnitID:    h.tenant.BuID,
			Type:              holdType,
			Severity:          holdreason.HoldSeverityAdvisory,
			ReasonCode:        "APPT",
			Source:            shipmenthold.HoldSourceUser,
			VisibleToCustomer: visible,
			StartedAt:         h.now - 60,
		}
		if released {
			at := h.now
			entity.ReleasedAt = &at
		}
		require.NoError(t, shipmentholddb.ShipmentHolds.Create(h.ctx, h.db, entity))
	}
	hold(holdreason.HoldTypeCustomer, true, false)
	hold(holdreason.HoldTypeOperational, false, false)
	hold(holdreason.HoldTypeFinance, true, true)

	comment := func(visibility shipmentvocab.CommentVisibility, text string) {
		entity := &shipment.ShipmentComment{
			ID:             pulid.MustNew("shc_"),
			OrganizationID: h.tenant.OrgID,
			BusinessUnitID: h.tenant.BuID,
			ShipmentID:     graph.Shipment.ID,
			UserID:         h.data.User.ID,
			Comment:        text,
			Visibility:     visibility,
		}
		require.NoError(t, shipmentdb.ShipmentComments.Create(h.ctx, h.db, entity))
	}
	comment(shipmentvocab.CommentVisibilityCustomer, "Appointment moved to 14:00")
	comment(shipmentvocab.CommentVisibilityInternal, "Driver is grumpy")

	holds, err := h.facts.CustomerVisibleHolds(h.ctx, h.tenant, graph.Shipment.ID)
	require.NoError(t, err)
	require.Len(t, holds, 1)
	assert.Equal(t, "CustomerHold", holds[0].Type)
	assert.Equal(t, "APPT", holds[0].Label)

	notes, err := h.facts.CustomerVisibleNotes(h.ctx, h.tenant, graph.Shipment.ID, 10)
	require.NoError(t, err)
	require.Len(t, notes, 1)
	assert.Equal(t, "Appointment moved to 14:00", notes[0].Text)

	descriptors, err := h.facts.Descriptors(h.ctx, h.tenant, graph.Shipment.ID)
	require.NoError(t, err)
	assert.NotNil(t, descriptors)

	location, err := h.facts.LastReportedLocation(h.ctx, h.tenant, graph.Shipment.ID)
	require.NoError(t, err)
	assert.Nil(t, location)
}
