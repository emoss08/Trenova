//go:build integration

package edimessagerepository_test

import (
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/edimessagerepository"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const overdueAge = int64(4 * 60 * 60)

type ackHealthMessage struct {
	name     string
	partner  int
	ack      edi.MessageAcknowledgmentStatus
	delivery edi.MessageDeliveryStatus
	age      int64
}

func TestOverdueAcks_ListAndCountAgreeOnWhatIsOverdue(t *testing.T) {
	ctx, db := seededDevelopmentDB(t)

	var org seededSummaryOrg
	require.NoError(t, db.NewSelect().
		Table("organizations").
		Column("id", "business_unit_id").
		Where("NOT EXISTS (SELECT 1 FROM edi_partners AS ep WHERE ep.organization_id = organizations.id)").
		Limit(1).
		Scan(ctx, &org))
	var documentType seededSummaryDocumentType
	require.NoError(t, db.NewSelect().
		Table("edi_document_types").
		Column("id").
		Where("transaction_set = ?", edi.TransactionSet204).
		Where("direction = ?", edi.DocumentDirectionOutbound).
		Limit(1).
		Scan(ctx, &documentType))
	tenant := pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID}

	partners := make([]*edi.EDIPartner, 0, 2)
	for index, name := range []string{"Acme Retail", "Bolt Logistics"} {
		partner := &edi.EDIPartner{
			BusinessUnitID: org.BusinessUnitID,
			OrganizationID: org.ID,
			Kind:           edi.PartnerKindExternal,
			Code:           fmt.Sprintf("ACK-HEALTH-%d", index),
			Name:           name,
		}
		_, err := db.NewInsert().Model(partner).Exec(ctx)
		require.NoError(t, err)
		partners = append(partners, partner)
	}

	now := timeutils.NowUnix()
	sent := edi.MessageDeliveryStatusSent
	pending := edi.MessageAcknowledgmentStatusPending
	fixtures := []ackHealthMessage{
		{"acmeOldest", 0, pending, sent, 30 * 60 * 60},
		{"boltOld", 1, pending, sent, 20 * 60 * 60},
		{"acmeOld", 0, pending, sent, 6 * 60 * 60},
		{"acmeJustPending", 0, pending, sent, 60 * 60},
		{"boltAcceptedOld", 1, edi.MessageAcknowledgmentStatusAccepted, sent, 40 * 60 * 60},
		{"boltErrorsOld", 1, edi.MessageAcknowledgmentStatusAcceptedWithErrors, sent, 40 * 60 * 60},
		{"acmeDead", 0, edi.MessageAcknowledgmentStatusNotExpected,
			edi.MessageDeliveryStatusDeadLettered, 60},
		{"boltDead", 1, edi.MessageAcknowledgmentStatusNotExpected,
			edi.MessageDeliveryStatusDeadLettered, 120},
	}
	ids := make(map[string]pulid.ID, len(fixtures))
	for index, entry := range fixtures {
		message := &edi.EDIMessage{
			BusinessUnitID:           org.BusinessUnitID,
			OrganizationID:           org.ID,
			EDIPartnerID:             partners[entry.partner].ID,
			DocumentTypeID:           documentType.ID,
			Direction:                edi.DocumentDirectionOutbound,
			Standard:                 edi.EDIStandardX12,
			TransactionSet:           edi.TransactionSet214,
			X12Version:               edi.DefaultX12204Version,
			Status:                   edi.MessageStatusGenerated,
			ValidationMode:           edi.ValidationModeDisabled,
			RawX12:                   "ISA*TEST~",
			InterchangeControlNumber: fmt.Sprintf("%09d", index+1),
			GroupControlNumber:       fmt.Sprintf("%d", index+1),
			TransactionControlNumber: fmt.Sprintf("%04d", index+1),
			DeliveryStatus:           entry.delivery,
			AckStatus:                entry.ack,
			GeneratedAt:              now - entry.age,
		}
		_, err := db.NewInsert().Model(message).Exec(ctx)
		require.NoError(t, err, entry.name)
		ids[entry.name] = message.ID
	}

	repo := edimessagerepository.New(edimessagerepository.Params{
		DB:     postgres.NewTestConnection(db),
		Logger: zap.NewNop(),
	})
	pendingSince := now - overdueAge

	t.Run("every partner", func(t *testing.T) {
		count, err := repo.GetOverdueAckCount(ctx, ediports.GetEDIOverdueAckCountRequest{
			TenantInfo:   tenant,
			PendingSince: pendingSince,
		})
		require.NoError(t, err)
		require.Equal(t, 3, count)

		overdue, err := repo.ListOverdueAcks(ctx, &ediports.ListOverdueEDIAcksRequest{
			TenantInfo:   tenant,
			PendingSince: pendingSince,
			Limit:        10,
		})
		require.NoError(t, err)
		require.Equal(t,
			[]pulid.ID{ids["acmeOldest"], ids["boltOld"], ids["acmeOld"]},
			messageIDs(overdue.Messages),
			"oldest first, and nothing that is not overdue")
		require.Equal(t, []*ediports.OverdueEDIAckPartner{
			{
				PartnerID:         partners[0].ID,
				PartnerName:       "Acme Retail",
				PartnerCode:       "ACK-HEALTH-0",
				Count:             2,
				OldestGeneratedAt: now - 30*60*60,
			},
			{
				PartnerID:         partners[1].ID,
				PartnerName:       "Bolt Logistics",
				PartnerCode:       "ACK-HEALTH-1",
				Count:             1,
				OldestGeneratedAt: now - 20*60*60,
			},
		}, overdue.Partners, "partners in the order of their oldest overdue message")
		total := 0
		for _, group := range overdue.Partners {
			total += group.Count
		}
		require.Equal(t, count, total, "the list and the count read one predicate")
		require.NotNil(t, overdue.Messages[0].Partner)
		require.Equal(t, "Acme Retail", overdue.Messages[0].Partner.Name)
	})

	t.Run("a limit cuts the rows but not the counts", func(t *testing.T) {
		overdue, err := repo.ListOverdueAcks(ctx, &ediports.ListOverdueEDIAcksRequest{
			TenantInfo:   tenant,
			PendingSince: pendingSince,
			Limit:        1,
		})
		require.NoError(t, err)
		require.Equal(t, []pulid.ID{ids["acmeOldest"]}, messageIDs(overdue.Messages))
		require.Len(t, overdue.Partners, 2)
		require.Equal(t, 2, overdue.Partners[0].Count)
		require.Equal(t, 1, overdue.Partners[1].Count)
	})

	t.Run("one partner", func(t *testing.T) {
		count, err := repo.GetOverdueAckCount(ctx, ediports.GetEDIOverdueAckCountRequest{
			TenantInfo:   tenant,
			PendingSince: pendingSince,
			PartnerID:    partners[1].ID,
		})
		require.NoError(t, err)
		require.Equal(t, 1, count)

		overdue, err := repo.ListOverdueAcks(ctx, &ediports.ListOverdueEDIAcksRequest{
			TenantInfo:   tenant,
			PendingSince: pendingSince,
			PartnerID:    partners[1].ID,
			Limit:        10,
		})
		require.NoError(t, err)
		require.Equal(t, []pulid.ID{ids["boltOld"]}, messageIDs(overdue.Messages))
		require.Len(t, overdue.Partners, 1)
		require.Equal(t, partners[1].ID, overdue.Partners[0].PartnerID)
		require.Equal(t, 1, overdue.Partners[0].Count)

		counts, err := repo.GetAckStatusCounts(ctx, ediports.GetEDIMessageStatusCountsRequest{
			TenantInfo: tenant,
			PartnerID:  partners[1].ID,
		})
		require.NoError(t, err)
		require.Equal(t, map[edi.MessageAcknowledgmentStatus]int{
			edi.MessageAcknowledgmentStatusPending:            1,
			edi.MessageAcknowledgmentStatusAccepted:           1,
			edi.MessageAcknowledgmentStatusAcceptedWithErrors: 1,
			edi.MessageAcknowledgmentStatusNotExpected:        1,
		}, counts)

		dead, err := repo.ListRecentDeadLettered(ctx, &ediports.ListRecentEDIMessageFailuresRequest{
			TenantInfo: tenant,
			PartnerID:  partners[1].ID,
			Limit:      5,
		})
		require.NoError(t, err)
		require.Equal(t, []pulid.ID{ids["boltDead"]}, messageIDs(dead))
	})

	t.Run("counts by partner", func(t *testing.T) {
		rows, err := repo.GetAckStatusCountsByPartner(ctx, ediports.GetEDIMessageStatusCountsRequest{
			TenantInfo: tenant,
			Since:      now - 35*60*60,
		})
		require.NoError(t, err)
		require.Equal(t, []*ediports.EDIPartnerAckStatusCount{
			{
				PartnerID: partners[0].ID, PartnerName: "Acme Retail", PartnerCode: "ACK-HEALTH-0",
				AckStatus: edi.MessageAcknowledgmentStatusNotExpected, Count: 1,
			},
			{
				PartnerID: partners[0].ID, PartnerName: "Acme Retail", PartnerCode: "ACK-HEALTH-0",
				AckStatus: edi.MessageAcknowledgmentStatusPending, Count: 3,
			},
			{
				PartnerID: partners[1].ID, PartnerName: "Bolt Logistics", PartnerCode: "ACK-HEALTH-1",
				AckStatus: edi.MessageAcknowledgmentStatusNotExpected, Count: 1,
			},
			{
				PartnerID: partners[1].ID, PartnerName: "Bolt Logistics", PartnerCode: "ACK-HEALTH-1",
				AckStatus: edi.MessageAcknowledgmentStatusPending, Count: 1,
			},
		}, rows, "by partner name, then status, and only what was sent in the window")

		one, err := repo.GetAckStatusCountsByPartner(ctx, ediports.GetEDIMessageStatusCountsRequest{
			TenantInfo: tenant,
			PartnerID:  partners[1].ID,
		})
		require.NoError(t, err)
		counts := make(map[edi.MessageAcknowledgmentStatus]int, len(one))
		for _, row := range one {
			require.Equal(t, partners[1].ID, row.PartnerID)
			counts[row.AckStatus] = row.Count
		}
		require.Equal(t, map[edi.MessageAcknowledgmentStatus]int{
			edi.MessageAcknowledgmentStatusPending:            1,
			edi.MessageAcknowledgmentStatusAccepted:           1,
			edi.MessageAcknowledgmentStatusAcceptedWithErrors: 1,
			edi.MessageAcknowledgmentStatusNotExpected:        1,
		}, counts, "the same counts GetAckStatusCounts gives for one partner")
	})

	t.Run("nothing overdue", func(t *testing.T) {
		overdue, err := repo.ListOverdueAcks(ctx, &ediports.ListOverdueEDIAcksRequest{
			TenantInfo:   tenant,
			PendingSince: now - 48*60*60,
			Limit:        10,
		})
		require.NoError(t, err)
		require.Empty(t, overdue.Messages)
		require.Empty(t, overdue.Partners)
	})
}

func messageIDs(messages []*edi.EDIMessage) []pulid.ID {
	ids := make([]pulid.ID, 0, len(messages))
	for _, message := range messages {
		ids = append(ids, message.ID)
	}

	return ids
}

func TestDevelopmentSeed_TheEDIPartnerHasAcknowledgmentsInEveryState(t *testing.T) {
	ctx, db := seededDevelopmentDB(t)

	var partner struct {
		ID             pulid.ID `bun:"id"`
		OrganizationID pulid.ID `bun:"organization_id"`
		BusinessUnitID pulid.ID `bun:"business_unit_id"`
	}
	require.NoError(t, db.NewSelect().
		Table("edi_partners").
		Column("id", "organization_id", "business_unit_id").
		Where("code = ?", "ACMEMFG").
		Scan(ctx, &partner))
	tenant := pagination.TenantInfo{OrgID: partner.OrganizationID, BuID: partner.BusinessUnitID}
	repo := edimessagerepository.New(edimessagerepository.Params{
		DB:     postgres.NewTestConnection(db),
		Logger: zap.NewNop(),
	})

	overdue, err := repo.ListOverdueAcks(ctx, &ediports.ListOverdueEDIAcksRequest{
		TenantInfo:   tenant,
		PendingSince: timeutils.NowUnix() - overdueAge,
		PartnerID:    partner.ID,
		Limit:        10,
	})
	require.NoError(t, err)
	require.Len(t, overdue.Messages, 3)
	require.Len(t, overdue.Partners, 1)
	require.Equal(t, "Acme Manufacturing EDI", overdue.Partners[0].PartnerName)

	counts, err := repo.GetAckStatusCounts(ctx, ediports.GetEDIMessageStatusCountsRequest{
		TenantInfo: tenant,
		PartnerID:  partner.ID,
	})
	require.NoError(t, err)
	require.Equal(t, 4, counts[edi.MessageAcknowledgmentStatusPending])
	require.Equal(t, 1, counts[edi.MessageAcknowledgmentStatusAcceptedWithErrors])
	require.Equal(t, 1, counts[edi.MessageAcknowledgmentStatusRejected])

	dead, err := repo.ListRecentDeadLettered(ctx, &ediports.ListRecentEDIMessageFailuresRequest{
		TenantInfo: tenant,
		PartnerID:  partner.ID,
	})
	require.NoError(t, err)
	require.Len(t, dead, 1)
}
