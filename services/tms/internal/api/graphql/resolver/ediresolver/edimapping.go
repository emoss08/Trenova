package ediresolver

import (
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlconnection"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediservice"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ediinboundfile"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edimessage"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ladingproject"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edimessageports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
)

func ediPartnerConnectionToModel(
	result *pagination.CursorListResult[*edi.EDIPartner],
) (*gqlmodel.EDIPartnerConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
		result,
		func(node *edi.EDIPartner, cursor string) *gqlmodel.EDIPartnerEdge {
			return &gqlmodel.EDIPartnerEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.EDIPartnerEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.EDIPartnerConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func ediCommunicationProfileConnectionToModel(
	result *pagination.CursorListResult[*edi.EDICommunicationProfile],
) (*gqlmodel.EDICommunicationProfileConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
		result,
		func(node *edi.EDICommunicationProfile, cursor string) *gqlmodel.EDICommunicationProfileEdge {
			return &gqlmodel.EDICommunicationProfileEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.EDICommunicationProfileEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.EDICommunicationProfileConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func ediTransferConnectionToModel(
	result *pagination.CursorListResult[*editransfer.EDITransfer],
) (*gqlmodel.EDITransferConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
		result,
		func(node *editransfer.EDITransfer, cursor string) *gqlmodel.EDITransferEdge {
			return &gqlmodel.EDITransferEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.EDITransferEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.EDITransferConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func ediMessageConnectionToModel(
	result *pagination.CursorListResult[*edimessage.EDIMessage],
) (*gqlmodel.EDIMessageConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
		result,
		func(node *edimessage.EDIMessage, cursor string) *gqlmodel.EDIMessageEdge {
			return &gqlmodel.EDIMessageEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.EDIMessageEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.EDIMessageConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func ediPartnerScorecardsToModel(
	rows []*edimessageports.EDIPartnerScorecardRow,
) []*gqlmodel.EDIPartnerScorecard {
	now := timeutils.NowUnix()
	cards := make([]*gqlmodel.EDIPartnerScorecard, 0, len(rows))
	for _, row := range rows {
		card := &gqlmodel.EDIPartnerScorecard{
			PartnerID:           row.PartnerID.String(),
			PartnerName:         row.PartnerName,
			PartnerCode:         row.PartnerCode,
			OutboundTotal:       int(row.OutboundTotal),
			SentCount:           int(row.SentCount),
			FailedCount:         int(row.FailedCount),
			DeadLetteredCount:   int(row.DeadLetteredCount),
			ReceivedCount:       int(row.ReceivedCount),
			AvgAckSeconds:       row.AvgAckSeconds,
			P95AckSeconds:       row.P95AckSeconds,
			OverdueAckCount:     int(row.OverdueAckCount),
			PendingOver4hCount:  int(row.PendingOver4hCount),
			PendingOver24hCount: int(row.PendingOver24hCount),
		}
		if attempted := row.SentCount + row.FailedCount + row.DeadLetteredCount; attempted > 0 {
			rate := float64(row.SentCount) / float64(attempted)
			card.DeliverySuccessRate = &rate
		}
		if row.OldestPendingAt != nil && now >= *row.OldestPendingAt {
			age := int(now - *row.OldestPendingAt)
			card.OldestPendingAgeSeconds = &age
		}
		cards = append(cards, card)
	}
	return cards
}

func ediVolumeSeriesToModel(series *ediservice.EDIVolumeSeries) []*gqlmodel.EDIVolumePoint {
	points := make([]*gqlmodel.EDIVolumePoint, 0, len(series.Points))
	for _, point := range series.Points {
		points = append(points, &gqlmodel.EDIVolumePoint{
			BucketStart:   int(point.BucketStart),
			BucketSeconds: int(series.BucketSeconds),
			OutboundCount: int(point.OutboundCount),
			SentCount:     int(point.SentCount),
			FailedCount:   int(point.FailedCount),
			ReceivedCount: int(point.ReceivedCount),
		})
	}
	return points
}

func ediSummaryToModel(summary *ediservice.EDISummary) *gqlmodel.EDISummary {
	attention := make(
		[]*gqlmodel.EDISummaryAttentionItem,
		0,
		len(summary.RecentDeadLettered)+len(summary.RecentQuarantined),
	)
	for _, message := range summary.RecentDeadLettered {
		item := &gqlmodel.EDISummaryAttentionItem{
			Kind: gqlmodel.EDISummaryAttentionKindMessage,
			ID:   message.ID.String(),
			Reference: gqlscalar.StrPtr(
				string(message.TransactionSet) + " " + message.TransactionControlNumber,
			),
			Error:      gqlscalar.StrPtr(message.DeliveryLastError),
			OccurredAt: int(message.UpdatedAt),
		}
		if message.EDIPartnerID.IsNotNil() {
			partnerID := message.EDIPartnerID.String()
			item.PartnerID = &partnerID
		}
		if message.Partner != nil {
			item.PartnerName = gqlscalar.StrPtr(message.Partner.Name)
			item.PartnerCode = gqlscalar.StrPtr(message.Partner.Code)
		}
		attention = append(attention, item)
	}
	for _, file := range summary.RecentQuarantined {
		item := &gqlmodel.EDISummaryAttentionItem{
			Kind:       gqlmodel.EDISummaryAttentionKindInboundFile,
			ID:         file.ID.String(),
			Reference:  gqlscalar.StrPtr(file.FileName),
			Error:      gqlscalar.StrPtr(file.FailureReason),
			OccurredAt: int(file.ReceivedAt),
		}
		if file.EDIPartnerID.IsNotNil() {
			partnerID := file.EDIPartnerID.String()
			item.PartnerID = &partnerID
		}
		if file.Partner != nil {
			item.PartnerName = gqlscalar.StrPtr(file.Partner.Name)
			item.PartnerCode = gqlscalar.StrPtr(file.Partner.Code)
		}
		attention = append(attention, item)
	}
	slices.SortFunc(attention, func(a, b *gqlmodel.EDISummaryAttentionItem) int {
		return b.OccurredAt - a.OccurredAt
	})

	return &gqlmodel.EDISummary{
		DeliveryStatusCounts:        summaryStatusCounts(summary.DeliveryStatusCounts),
		AckStatusCounts:             summaryStatusCounts(summary.AckStatusCounts),
		InboundFileStatusCounts:     summaryStatusCounts(summary.InboundFileStatusCounts),
		InboundTransferStatusCounts: summaryStatusCounts(summary.InboundTransferStatusCounts),
		OverdueAckCount:             summary.OverdueAckCount,
		AttentionItems:              attention,
	}
}

func summaryStatusCounts[T ~string](counts map[T]int) []*gqlmodel.EDISummaryStatusCount {
	result := make([]*gqlmodel.EDISummaryStatusCount, 0, len(counts))
	for status, count := range counts {
		result = append(result, &gqlmodel.EDISummaryStatusCount{
			Status: string(status),
			Count:  count,
		})
	}
	slices.SortFunc(result, func(a, b *gqlmodel.EDISummaryStatusCount) int {
		return strings.Compare(a.Status, b.Status)
	})
	return result
}

func ediFlowInstanceConnectionToModel(
	result *pagination.CursorListResult[*ladingproject.FlowInstance],
) (*gqlmodel.EDIFlowInstanceConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
		result,
		func(node *ladingproject.FlowInstance, cursor string) *gqlmodel.EDIFlowInstanceEdge {
			return &gqlmodel.EDIFlowInstanceEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.EDIFlowInstanceEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.EDIFlowInstanceConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func ediMappingProfileConnectionToModel(
	result *pagination.CursorListResult[*edi.EDIMappingProfile],
) (*gqlmodel.EDIMappingProfileConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
		result,
		func(node *edi.EDIMappingProfile, cursor string) *gqlmodel.EDIMappingProfileEdge {
			return &gqlmodel.EDIMappingProfileEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.EDIMappingProfileEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.EDIMappingProfileConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}

func ediInboundFileConnectionToModel(
	result *pagination.CursorListResult[*ediinboundfile.EDIInboundFile],
) (*gqlmodel.EDIInboundFileConnection, error) {
	page, err := gqlconnection.EntityCursorConnection(
		result,
		func(node *ediinboundfile.EDIInboundFile, cursor string) *gqlmodel.EDIInboundFileEdge {
			return &gqlmodel.EDIInboundFileEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.EDIInboundFileEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.EDIInboundFileConnection{
		Edges:           page.Edges,
		PageInfo:        page.PageInfo,
		TotalCount:      page.TotalCount,
		TotalCountExact: page.TotalCountExact,
	}, nil
}
