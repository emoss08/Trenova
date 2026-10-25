//go:build integration

package edimessagerepository_test

import (
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/testutil/seededtest"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/edicommunicationprofilerepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/ediinboundfilerepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/edimappingprofilerepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/edimessagerepository"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/edipartnerrepository"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/editest"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/modules/parties/adapters/postgres/carrierrepository"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type partnerCardMessage struct {
	partner  pulid.ID
	set      edi.TransactionSet
	dir      edi.DocumentDirection
	docType  pulid.ID
	delivery edi.MessageDeliveryStatus
	age      int64
}

func TestPartnerCardReadsKeepToOnePartner(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	defer cleanup()

	var err error

	conn := postgres.NewTestConnection(db)

	scope := tenantWithoutPartners(t, ctx, db)
	ctx = scope.Ctx
	org := scope.Info
	documentType := func(set edi.TransactionSet, direction edi.DocumentDirection) pulid.ID {
		return findDocumentType(t, ctx, db, set, direction).ID
	}
	outbound214 := documentType(edi.TransactionSet214, edi.DocumentDirectionOutbound)
	inbound204 := documentType(edi.TransactionSet204, edi.DocumentDirectionInbound)
	tenantInfo := org

	newPartner := func(code string) *edi.EDIPartner {
		partner := &edi.EDIPartner{
			BusinessUnitID: org.BuID,
			OrganizationID: org.OrgID,
			Kind:           edi.PartnerKindExternal,
			Code:           code,
			Name:           code + " Partner",
		}
		require.NoError(t, create(ctx, db, partner))
		return partner
	}
	tyson := newPartner("CARD-TYSON")
	other := newPartner("CARD-OTHER")

	newProfile := func(partner *edi.EDIPartner, name string, status string) *edi.EDICommunicationProfile {
		profile := &edi.EDICommunicationProfile{
			BusinessUnitID: org.BuID,
			OrganizationID: org.OrgID,
			EDIPartnerID:   partner.ID,
			Method:         edi.ConnectionMethodSFTP,
			Name:           name,
		}
		require.NoError(t, create(ctx, db, profile))
		if status != "" {
			editest.Exec(t, ctx, db, "UPDATE edi_communication_profiles SET status = $1 WHERE id = $2",
				status, profile.ID.String())
		}
		return profile
	}
	tysonMailbox := newProfile(tyson, "Tyson mailbox", "")
	tysonRetired := newProfile(tyson, "Tyson old mailbox", "Inactive")
	otherMailbox := newProfile(other, "Other mailbox", "")

	now := timeutils.NowUnix()
	const day = int64(24 * 60 * 60)
	messages := []partnerCardMessage{
		{tyson.ID, edi.TransactionSet214, edi.DocumentDirectionOutbound, outbound214, edi.MessageDeliveryStatusSent, 60},
		{tyson.ID, edi.TransactionSet214, edi.DocumentDirectionOutbound, outbound214, edi.MessageDeliveryStatusSent, 2 * day},
		{tyson.ID, edi.TransactionSet214, edi.DocumentDirectionOutbound, outbound214, edi.MessageDeliveryStatusFailed, 3 * day},
		{tyson.ID, edi.TransactionSet214, edi.DocumentDirectionOutbound, outbound214, edi.MessageDeliveryStatusDeadLettered, day},
		{tyson.ID, edi.TransactionSet214, edi.DocumentDirectionOutbound, outbound214, edi.MessageDeliveryStatusSent, 10 * day},
		{tyson.ID, edi.TransactionSet204, edi.DocumentDirectionInbound, inbound204, "", 120},
		{tyson.ID, edi.TransactionSet204, edi.DocumentDirectionInbound, inbound204, "", 5 * day},
		{other.ID, edi.TransactionSet214, edi.DocumentDirectionOutbound, outbound214, edi.MessageDeliveryStatusSent, 60},
		{other.ID, edi.TransactionSet214, edi.DocumentDirectionOutbound, outbound214, edi.MessageDeliveryStatusDeadLettered, 60},
	}
	for index, entry := range messages {
		message := &edi.EDIMessage{
			BusinessUnitID:           org.BuID,
			OrganizationID:           org.OrgID,
			EDIPartnerID:             entry.partner,
			DocumentTypeID:           entry.docType,
			Direction:                entry.dir,
			Standard:                 edi.EDIStandardX12,
			TransactionSet:           entry.set,
			X12Version:               edi.DefaultX12204Version,
			Status:                   edi.MessageStatusGenerated,
			ValidationMode:           edi.ValidationModeDisabled,
			RawX12:                   "ISA*TEST~",
			InterchangeControlNumber: fmt.Sprintf("%09d", index+1),
			GroupControlNumber:       fmt.Sprintf("%d", index+1),
			TransactionControlNumber: fmt.Sprintf("%04d", index+1),
			DeliveryStatus:           entry.delivery,
			AckStatus:                edi.MessageAcknowledgmentStatusNotExpected,
			GeneratedAt:              now - entry.age,
		}
		require.NoError(t, create(ctx, db, message), "message %d", index)
	}
	editest.Exec(t, ctx, db, "UPDATE edi_messages SET updated_at = generated_at WHERE organization_id = $1",
		org.OrgID.String())

	messageRepo := edimessagerepository.New(edimessagerepository.Params{DB: conn, Logger: zap.NewNop()})
	weekAgo := now - 7*day

	sets, err := messageRepo.GetPartnerSetActivity(ctx, &ediports.GetEDIPartnerSetActivityRequest{
		TenantInfo: tenantInfo,
		PartnerID:  tyson.ID,
		Since:      weekAgo,
	})
	require.NoError(t, err)
	require.Len(t, sets, 2)
	assert.Equal(t, ediports.EDIPartnerSetActivityRow{
		TransactionSet:    edi.TransactionSet214,
		Direction:         edi.DocumentDirectionOutbound,
		Count:             4,
		FailedCount:       1,
		DeadLetteredCount: 1,
		LastAt:            now - 60,
	}, *sets[0], "outbound first; the ten-day-old 214 and the other partner's are not counted")
	assert.Equal(t, ediports.EDIPartnerSetActivityRow{
		TransactionSet: edi.TransactionSet204,
		Direction:      edi.DocumentDirectionInbound,
		Count:          2,
		LastAt:         now - 120,
	}, *sets[1], "an inbound set never counts a failure")

	scorecards, err := messageRepo.GetPartnerScorecards(ctx, &ediports.GetEDIPartnerScorecardsRequest{
		TenantInfo:             tenantInfo,
		Since:                  weekAgo,
		PartnerID:              tyson.ID,
		OverdueAckPendingSince: now - 4*60*60,
		PendingOver4hBefore:    now - 4*60*60,
		PendingOver24hBefore:   now - day,
	})
	require.NoError(t, err)
	require.Len(t, scorecards, 1, "the scorecard keeps to the partner asked for")
	assert.Equal(t, tyson.ID, scorecards[0].PartnerID)
	assert.Equal(t, int64(2), scorecards[0].SentCount)
	assert.Equal(t, int64(2), scorecards[0].ReceivedCount)

	series, err := messageRepo.GetVolumeSeries(ctx, ediports.GetEDIVolumeSeriesRequest{
		TenantInfo:    tenantInfo,
		Since:         now - 14*day,
		BucketSeconds: 3600,
		PartnerID:     tyson.ID,
	})
	require.NoError(t, err)
	var outboundTotal, receivedTotal int64
	for _, point := range series {
		outboundTotal += point.OutboundCount
		receivedTotal += point.ReceivedCount
	}
	assert.Equal(t, int64(5), outboundTotal, "every 214 of the partner's in fourteen days")
	assert.Equal(t, int64(2), receivedTotal)

	dead, err := messageRepo.ListRecentDeadLettered(ctx, &ediports.ListRecentEDIMessageFailuresRequest{
		TenantInfo: tenantInfo,
		PartnerID:  tyson.ID,
		Since:      now - 2*60*60,
		Limit:      5,
	})
	require.NoError(t, err)
	assert.Empty(t, dead, "a dead letter older than the window is not drawn")
	dead, err = messageRepo.ListRecentDeadLettered(ctx, &ediports.ListRecentEDIMessageFailuresRequest{
		TenantInfo: tenantInfo,
		PartnerID:  tyson.ID,
		Since:      weekAgo,
		Limit:      5,
	})
	require.NoError(t, err)
	require.Len(t, dead, 1)
	assert.Equal(t, tyson.ID, dead[0].EDIPartnerID)

	for index, entry := range []struct {
		partner *edi.EDIPartner
		profile *edi.EDICommunicationProfile
		age     int64
	}{
		{tyson, tysonMailbox, 60},
		{tyson, tysonMailbox, 9 * day},
		{other, otherMailbox, 60},
	} {
		file := &edi.EDIInboundFile{
			BusinessUnitID:         org.BuID,
			OrganizationID:         org.OrgID,
			CommunicationProfileID: entry.profile.ID,
			EDIPartnerID:           entry.partner.ID,
			Method:                 edi.ConnectionMethodSFTP,
			RemotePath:             "/outbound/tenders/drop.csv",
			FileName:               "drop.csv",
			Checksum:               pulid.MustNew("chk_").String(),
			RawContent:             "not x12",
			Status:                 edi.InboundFileStatusQuarantined,
			FailureReason:          "not an interchange",
			ReceivedAt:             now - entry.age,
		}
		require.NoError(t, create(ctx, db, file), "file %d", index)
	}
	fileRepo := ediinboundfilerepository.New(ediinboundfilerepository.Params{DB: conn, Logger: zap.NewNop()})
	quarantined, err := fileRepo.ListRecentQuarantined(ctx, ediports.ListRecentQuarantinedEDIInboundFilesRequest{
		TenantInfo: tenantInfo,
		PartnerID:  tyson.ID,
		Since:      weekAgo,
		Limit:      5,
	})
	require.NoError(t, err)
	require.Len(t, quarantined, 1, "only this partner's files inside the window")
	assert.Equal(t, tyson.ID, quarantined[0].EDIPartnerID)

	profileRepo := edicommunicationprofilerepository.New(edicommunicationprofilerepository.Params{
		DB:     conn,
		Logger: zap.NewNop(),
	})
	profiles, err := profileRepo.ListProfilesByPartner(ctx, ediports.ListEDICommunicationProfilesByPartnerRequest{
		TenantInfo: tenantInfo,
		PartnerID:  tyson.ID,
	})
	require.NoError(t, err)
	require.Len(t, profiles, 2, "inactive transports are listed too")
	assert.Equal(t, tysonMailbox.ID, profiles[0].ID, "the active transport first")
	assert.Equal(t, tysonRetired.ID, profiles[1].ID)

	mappingRepo := edimappingprofilerepository.New(edimappingprofilerepository.Params{
		DB:     conn,
		Logger: zap.NewNop(),
	})
	empty, err := mappingRepo.GetMappingSummary(ctx, ediports.GetEDIMappingSummaryRequest{
		TenantInfo: tenantInfo,
		PartnerID:  tyson.ID,
	})
	require.NoError(t, err)
	assert.True(t, empty.ProfileID.IsNil(), "a partner never mapped has no profile")
	assert.Empty(t, empty.Counts)

	_, err = mappingRepo.SaveMappingItems(ctx, &ediports.SaveMappingItemsRequest{
		PartnerID:  tyson.ID,
		TenantInfo: tenantInfo,
		Items: []*edi.EDIMappingProfileItem{
			{EntityType: edi.MappingEntityTypeLocation, SourceID: pulid.MustNew("loc_"), TargetID: pulid.MustNew("loc_")},
			{EntityType: edi.MappingEntityTypeLocation, SourceID: pulid.MustNew("loc_"), TargetID: pulid.MustNew("loc_")},
			{EntityType: edi.MappingEntityTypeAccessorialCharge, SourceID: pulid.MustNew("acc_"), TargetID: pulid.MustNew("acc_")},
		},
	})
	require.NoError(t, err)
	summary, err := mappingRepo.GetMappingSummary(ctx, ediports.GetEDIMappingSummaryRequest{
		TenantInfo: tenantInfo,
		PartnerID:  tyson.ID,
	})
	require.NoError(t, err)
	assert.False(t, summary.ProfileID.IsNil())
	assert.Positive(t, summary.UpdatedAt)
	counts := map[edi.MappingEntityType]int64{}
	for _, count := range summary.Counts {
		counts[count.EntityType] = count.Count
	}
	assert.Equal(t, map[edi.MappingEntityType]int64{
		edi.MappingEntityTypeLocation:          2,
		edi.MappingEntityTypeAccessorialCharge: 1,
	}, counts)

	partnerRepo := edipartnerrepository.New(edipartnerrepository.Params{DB: conn, Logger: zap.NewNop()})
	plain, err := partnerRepo.GetByID(ctx, ediports.GetEDIPartnerByIDRequest{ID: tyson.ID, TenantInfo: tenantInfo})
	require.NoError(t, err)
	assert.Nil(t, plain.Connection)
	linked, err := partnerRepo.GetByID(ctx, ediports.GetEDIPartnerByIDRequest{
		ID:           tyson.ID,
		TenantInfo:   tenantInfo,
		IncludeLinks: true,
	})
	require.NoError(t, err)
	assert.Equal(t, "CARD-TYSON", linked.Code)
	assert.Nil(t, linked.Customer, "no customer is named")
}

type seededCarrier struct {
	ID             pulid.ID
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	Name           string
}

func TestCarrierEDIChannelsByPartner(t *testing.T) {
	ctx, db, cleanup := seededtest.SetupTestDB(t)
	defer cleanup()

	conn := postgres.NewTestConnection(db)

	rows, err := db.Pool().Query(ctx, `SELECT id, organization_id, business_unit_id, name FROM carriers
		WHERE organization_id = (SELECT organization_id FROM carriers GROUP BY organization_id HAVING COUNT(*) >= 2 ORDER BY organization_id LIMIT 1)
		ORDER BY name ASC LIMIT 2`)
	require.NoError(t, err)
	carriers, err := pgx.CollectRows(rows, pgx.RowToStructByPos[seededCarrier])
	require.NoError(t, err)
	require.Len(t, carriers, 2, "the seeds give one organization two carriers")
	org := carriers[0]
	tenantInfo := pagination.TenantInfo{OrgID: org.OrganizationID, BuID: org.BusinessUnitID}
	ctx = dbscope.WithTenant(ctx, tenantInfo.DBTenant())

	newPartner := func(code string) *edi.EDIPartner {
		partner := &edi.EDIPartner{
			BusinessUnitID: org.BusinessUnitID,
			OrganizationID: org.OrganizationID,
			Kind:           edi.PartnerKindExternal,
			Code:           code,
			Name:           code + " Partner",
		}
		require.NoError(t, create(ctx, db, partner))
		return partner
	}
	van := newPartner("CARD-VAN")
	elsewhere := newPartner("CARD-ELSEWHERE")

	for _, entry := range []struct {
		carrier   seededCarrier
		partner   *edi.EDIPartner
		isDefault bool
	}{
		{carriers[0], van, false},
		{carriers[1], van, true},
		{carriers[0], elsewhere, true},
	} {
		channel := &carrier.CarrierEDIChannel{
			BusinessUnitID: org.BusinessUnitID,
			OrganizationID: org.OrganizationID,
			CarrierID:      entry.carrier.ID,
			EDIPartnerID:   entry.partner.ID,
			IsDefault:      entry.isDefault,
		}
		require.NoError(t, create(ctx, db, channel))
	}

	repo := carrierrepository.New(carrierrepository.Params{DB: conn, Logger: zap.NewNop()})
	channels, err := repo.ListEDIChannelsByPartner(ctx, carrierports.ListCarrierEDIChannelsByPartnerRequest{
		TenantInfo:   tenantInfo,
		EDIPartnerID: van.ID,
	})
	require.NoError(t, err)
	require.Len(t, channels, 2, "only the channels through this partner")
	assert.Equal(t, carriers[1].ID, channels[0].CarrierID, "the default channel first")
	assert.True(t, channels[0].IsDefault)
	require.NotNil(t, channels[0].Carrier, "each channel brings its carrier")
	assert.Equal(t, carriers[1].Name, channels[0].Carrier.Name)
	assert.Equal(t, carriers[0].ID, channels[1].CarrierID)
}
