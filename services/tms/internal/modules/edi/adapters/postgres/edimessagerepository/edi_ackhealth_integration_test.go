//go:build integration

package edimessagerepository_test

import (
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/edimessagerepository"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edimessage"
	"github.com/emoss08/trenova/internal/modules/edi/editest"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edimessageports"
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
	ack      edimessage.MessageAcknowledgmentStatus
	delivery edimessage.MessageDeliveryStatus
	age      int64
}

func TestOverdueAcks_ListAndCountAgreeOnWhatIsOverdue(t *testing.T) {
	ctx, db := seededDevelopmentDB(t)

	scope := tenantWithoutPartners(t, ctx, db)
	ctx = scope.Ctx
	org := scope.Info
	documentType := findDocumentType(
		t,
		ctx,
		db,
		edidocument.TransactionSet204,
		edidocument.DocumentDirectionOutbound,
	)
	tenant := org

	partners := make([]*edi.EDIPartner, 0, 2)
	for index, name := range []string{"Acme Retail", "Bolt Logistics"} {
		partner := &edi.EDIPartner{
			BusinessUnitID: org.BuID,
			OrganizationID: org.OrgID,
			Kind:           edi.PartnerKindExternal,
			Code:           fmt.Sprintf("ACK-HEALTH-%d", index),
			Name:           name,
		}
		require.NoError(t, create(ctx, db, partner))
		partners = append(partners, partner)
	}

	now := timeutils.NowUnix()
	sent := edimessage.MessageDeliveryStatusSent
	pending := edimessage.MessageAcknowledgmentStatusPending
	fixtures := []ackHealthMessage{
		{"acmeOldest", 0, pending, sent, 30 * 60 * 60},
		{"boltOld", 1, pending, sent, 20 * 60 * 60},
		{"acmeOld", 0, pending, sent, 6 * 60 * 60},
		{"acmeJustPending", 0, pending, sent, 60 * 60},
		{"boltAcceptedOld", 1, edimessage.MessageAcknowledgmentStatusAccepted, sent, 40 * 60 * 60},
		{
			"boltErrorsOld",
			1,
			edimessage.MessageAcknowledgmentStatusAcceptedWithErrors,
			sent,
			40 * 60 * 60,
		},
		{"acmeDead", 0, edimessage.MessageAcknowledgmentStatusNotExpected,
			edimessage.MessageDeliveryStatusDeadLettered, 60},
		{"boltDead", 1, edimessage.MessageAcknowledgmentStatusNotExpected,
			edimessage.MessageDeliveryStatusDeadLettered, 120},
	}
	ids := make(map[string]pulid.ID, len(fixtures))
	for index, entry := range fixtures {
		message := &edimessage.EDIMessage{
			BusinessUnitID:           org.BuID,
			OrganizationID:           org.OrgID,
			EDIPartnerID:             partners[entry.partner].ID,
			DocumentTypeID:           documentType.ID,
			Direction:                edidocument.DocumentDirectionOutbound,
			Standard:                 edidocument.EDIStandardX12,
			TransactionSet:           edidocument.TransactionSet214,
			X12Version:               edidocument.DefaultX12204Version,
			Status:                   edimessage.MessageStatusGenerated,
			RawX12:                   "ISA*TEST~",
			InterchangeControlNumber: fmt.Sprintf("%09d", index+1),
			GroupControlNumber:       fmt.Sprintf("%d", index+1),
			TransactionControlNumber: fmt.Sprintf("%04d", index+1),
			DeliveryStatus:           entry.delivery,
			AckStatus:                entry.ack,
			GeneratedAt:              now - entry.age,
		}
		require.NoError(t, create(ctx, db, message), entry.name)
		ids[entry.name] = message.ID
	}

	repo := edimessagerepository.New(edimessagerepository.Params{
		DB:     postgres.NewTestConnection(db),
		Logger: zap.NewNop(),
	})
	pendingSince := now - overdueAge

	t.Run("every partner", func(t *testing.T) {
		count, err := repo.GetOverdueAckCount(ctx, &edimessageports.GetEDIOverdueAckCountRequest{
			TenantInfo:   tenant,
			PendingSince: pendingSince,
		})
		require.NoError(t, err)
		require.Equal(t, 3, count)

		overdue, err := repo.ListOverdueAcks(ctx, &edimessageports.ListOverdueEDIAcksRequest{
			TenantInfo:   tenant,
			PendingSince: pendingSince,
			Limit:        10,
		})
		require.NoError(t, err)
		require.Equal(t,
			[]pulid.ID{ids["acmeOldest"], ids["boltOld"], ids["acmeOld"]},
			messageIDs(overdue.Messages),
			"oldest first, and nothing that is not overdue")
		require.Equal(t, []*edimessageports.OverdueEDIAckPartner{
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
		overdue, err := repo.ListOverdueAcks(ctx, &edimessageports.ListOverdueEDIAcksRequest{
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
		count, err := repo.GetOverdueAckCount(ctx, &edimessageports.GetEDIOverdueAckCountRequest{
			TenantInfo:   tenant,
			PendingSince: pendingSince,
			PartnerID:    partners[1].ID,
		})
		require.NoError(t, err)
		require.Equal(t, 1, count)

		overdue, err := repo.ListOverdueAcks(ctx, &edimessageports.ListOverdueEDIAcksRequest{
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

		counts, err := repo.GetAckStatusCounts(
			ctx,
			&edimessageports.GetEDIMessageStatusCountsRequest{
				TenantInfo: tenant,
				PartnerID:  partners[1].ID,
			},
		)
		require.NoError(t, err)
		require.Equal(t, map[edimessage.MessageAcknowledgmentStatus]int{
			edimessage.MessageAcknowledgmentStatusPending:            1,
			edimessage.MessageAcknowledgmentStatusAccepted:           1,
			edimessage.MessageAcknowledgmentStatusAcceptedWithErrors: 1,
			edimessage.MessageAcknowledgmentStatusNotExpected:        1,
		}, counts)

		dead, err := repo.ListRecentDeadLettered(
			ctx,
			&edimessageports.ListRecentEDIMessageFailuresRequest{
				TenantInfo: tenant,
				PartnerID:  partners[1].ID,
				Limit:      5,
			},
		)
		require.NoError(t, err)
		require.Equal(t, []pulid.ID{ids["boltDead"]}, messageIDs(dead))
	})

	t.Run("one transaction set", func(t *testing.T) {
		count, err := repo.GetOverdueAckCount(ctx, &edimessageports.GetEDIOverdueAckCountRequest{
			TenantInfo:     tenant,
			PendingSince:   pendingSince,
			TransactionSet: edidocument.TransactionSet214,
		})
		require.NoError(t, err)
		require.Equal(t, 3, count)

		none, err := repo.GetOverdueAckCount(ctx, &edimessageports.GetEDIOverdueAckCountRequest{
			TenantInfo:     tenant,
			PendingSince:   pendingSince,
			TransactionSet: edidocument.TransactionSet210,
		})
		require.NoError(t, err)
		require.Zero(t, none, "every fixture is a 214")

		overdue, err := repo.ListOverdueAcks(ctx, &edimessageports.ListOverdueEDIAcksRequest{
			TenantInfo:     tenant,
			PendingSince:   pendingSince,
			TransactionSet: edidocument.TransactionSet210,
			Limit:          10,
		})
		require.NoError(t, err)
		require.Empty(t, overdue.Messages)
		require.Empty(t, overdue.Partners)

		counts, err := repo.GetAckStatusCounts(
			ctx,
			&edimessageports.GetEDIMessageStatusCountsRequest{
				TenantInfo:     tenant,
				TransactionSet: edidocument.TransactionSet210,
			},
		)
		require.NoError(t, err)
		require.Empty(t, counts)

		rows, err := repo.GetAckStatusCountsByPartner(
			ctx,
			&edimessageports.GetEDIMessageStatusCountsRequest{
				TenantInfo:     tenant,
				TransactionSet: edidocument.TransactionSet214,
				PartnerID:      partners[1].ID,
			},
		)
		require.NoError(t, err)
		require.Len(t, rows, 4, "Bolt's pending, accepted, accepted with errors and dead letter")

		dead, err := repo.ListRecentDeadLettered(
			ctx,
			&edimessageports.ListRecentEDIMessageFailuresRequest{
				TenantInfo:     tenant,
				TransactionSet: edidocument.TransactionSet210,
				Limit:          5,
			},
		)
		require.NoError(t, err)
		require.Empty(t, dead)
	})

	t.Run("counts by partner", func(t *testing.T) {
		rows, err := repo.GetAckStatusCountsByPartner(
			ctx,
			&edimessageports.GetEDIMessageStatusCountsRequest{
				TenantInfo: tenant,
				Since:      now - 35*60*60,
			},
		)
		require.NoError(t, err)
		require.Equal(t, []*edimessageports.EDIPartnerAckStatusCount{
			{
				PartnerID: partners[0].ID, PartnerName: "Acme Retail", PartnerCode: "ACK-HEALTH-0",
				AckStatus: edimessage.MessageAcknowledgmentStatusNotExpected, Count: 1,
			},
			{
				PartnerID: partners[0].ID, PartnerName: "Acme Retail", PartnerCode: "ACK-HEALTH-0",
				AckStatus: edimessage.MessageAcknowledgmentStatusPending, Count: 3,
			},
			{
				PartnerID: partners[1].ID, PartnerName: "Bolt Logistics", PartnerCode: "ACK-HEALTH-1",
				AckStatus: edimessage.MessageAcknowledgmentStatusNotExpected, Count: 1,
			},
			{
				PartnerID: partners[1].ID, PartnerName: "Bolt Logistics", PartnerCode: "ACK-HEALTH-1",
				AckStatus: edimessage.MessageAcknowledgmentStatusPending, Count: 1,
			},
		}, rows, "by partner name, then status, and only what was sent in the window")

		one, err := repo.GetAckStatusCountsByPartner(
			ctx,
			&edimessageports.GetEDIMessageStatusCountsRequest{
				TenantInfo: tenant,
				PartnerID:  partners[1].ID,
			},
		)
		require.NoError(t, err)
		counts := make(map[edimessage.MessageAcknowledgmentStatus]int, len(one))
		for _, row := range one {
			require.Equal(t, partners[1].ID, row.PartnerID)
			counts[row.AckStatus] = row.Count
		}
		require.Equal(t, map[edimessage.MessageAcknowledgmentStatus]int{
			edimessage.MessageAcknowledgmentStatusPending:            1,
			edimessage.MessageAcknowledgmentStatusAccepted:           1,
			edimessage.MessageAcknowledgmentStatusAcceptedWithErrors: 1,
			edimessage.MessageAcknowledgmentStatusNotExpected:        1,
		}, counts, "the same counts GetAckStatusCounts gives for one partner")
	})

	t.Run("nothing overdue", func(t *testing.T) {
		overdue, err := repo.ListOverdueAcks(ctx, &edimessageports.ListOverdueEDIAcksRequest{
			TenantInfo:   tenant,
			PendingSince: now - 48*60*60,
			Limit:        10,
		})
		require.NoError(t, err)
		require.Empty(t, overdue.Messages)
		require.Empty(t, overdue.Partners)
	})
}

func messageIDs(messages []*edimessage.EDIMessage) []pulid.ID {
	ids := make([]pulid.ID, 0, len(messages))
	for _, message := range messages {
		ids = append(ids, message.ID)
	}

	return ids
}

func TestDevelopmentSeed_TheEDIPartnerHasAcknowledgmentsInEveryState(t *testing.T) {
	ctx, db := seededDevelopmentDB(t)

	var partnerID, orgID, buID string
	require.NoError(t, db.Pool().QueryRow(ctx,
		"SELECT id, organization_id, business_unit_id FROM edi_partners WHERE code = $1",
		"ACMEMFG",
	).Scan(&partnerID, &orgID, &buID))
	partner := struct{ ID pulid.ID }{ID: pulid.ID(partnerID)}
	scope := editest.TenantOf(
		ctx,
		pagination.TenantInfo{OrgID: pulid.ID(orgID), BuID: pulid.ID(buID)},
	)
	ctx = scope.Ctx
	tenant := scope.Info
	repo := edimessagerepository.New(edimessagerepository.Params{
		DB:     postgres.NewTestConnection(db),
		Logger: zap.NewNop(),
	})

	overdue, err := repo.ListOverdueAcks(ctx, &edimessageports.ListOverdueEDIAcksRequest{
		TenantInfo:   tenant,
		PendingSince: timeutils.NowUnix() - overdueAge,
		PartnerID:    partner.ID,
		Limit:        10,
	})
	require.NoError(t, err)
	require.Len(t, overdue.Messages, 3)
	require.Len(t, overdue.Partners, 1)
	require.Equal(t, "Acme Manufacturing EDI", overdue.Partners[0].PartnerName)

	counts, err := repo.GetAckStatusCounts(ctx, &edimessageports.GetEDIMessageStatusCountsRequest{
		TenantInfo: tenant,
		PartnerID:  partner.ID,
	})
	require.NoError(t, err)
	require.Equal(t, 4, counts[edimessage.MessageAcknowledgmentStatusPending])
	require.Equal(t, 1, counts[edimessage.MessageAcknowledgmentStatusAcceptedWithErrors])
	require.Equal(t, 1, counts[edimessage.MessageAcknowledgmentStatusRejected])

	dead, err := repo.ListRecentDeadLettered(
		ctx,
		&edimessageports.ListRecentEDIMessageFailuresRequest{
			TenantInfo: tenant,
			PartnerID:  partner.ID,
		},
	)
	require.NoError(t, err)
	require.Len(t, dead, 1)
}
