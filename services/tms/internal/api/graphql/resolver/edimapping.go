package resolver

import (
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/services/ediservice"
	"github.com/emoss08/trenova/pkg/pagination"
)

func ediPartnerConnectionToModel(
	result *pagination.CursorListResult[*edi.EDIPartner],
) (*gqlmodel.EdiPartnerConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *edi.EDIPartner, cursor string) *gqlmodel.EdiPartnerEdge {
			return &gqlmodel.EdiPartnerEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.EdiPartnerEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.EdiPartnerConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

func ediCommunicationProfileConnectionToModel(
	result *pagination.CursorListResult[*edi.EDICommunicationProfile],
) (*gqlmodel.EdiCommunicationProfileConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *edi.EDICommunicationProfile, cursor string) *gqlmodel.EdiCommunicationProfileEdge {
			return &gqlmodel.EdiCommunicationProfileEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.EdiCommunicationProfileEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.EdiCommunicationProfileConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

func ediTransferConnectionToModel(
	result *pagination.CursorListResult[*edi.EDITransfer],
) (*gqlmodel.EdiTransferConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *edi.EDITransfer, cursor string) *gqlmodel.EdiTransferEdge {
			return &gqlmodel.EdiTransferEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.EdiTransferEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.EdiTransferConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

func ediMessageConnectionToModel(
	result *pagination.CursorListResult[*edi.EDIMessage],
) (*gqlmodel.EdiMessageConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *edi.EDIMessage, cursor string) *gqlmodel.EdiMessageEdge {
			return &gqlmodel.EdiMessageEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.EdiMessageEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.EdiMessageConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

func ediSummaryToModel(summary *ediservice.EDISummary) *gqlmodel.EdiSummary {
	attention := make(
		[]*gqlmodel.EdiSummaryAttentionItem,
		0,
		len(summary.RecentDeadLettered)+len(summary.RecentQuarantined),
	)
	for _, message := range summary.RecentDeadLettered {
		item := &gqlmodel.EdiSummaryAttentionItem{
			Kind:       gqlmodel.EdiSummaryAttentionKindMessage,
			ID:         message.ID.String(),
			Reference:  strPtr(string(message.TransactionSet) + " " + message.TransactionControlNumber),
			Error:      strPtr(message.DeliveryLastError),
			OccurredAt: int(message.UpdatedAt),
		}
		if message.EDIPartnerID.IsNotNil() {
			partnerID := message.EDIPartnerID.String()
			item.PartnerID = &partnerID
		}
		if message.Partner != nil {
			item.PartnerName = strPtr(message.Partner.Name)
			item.PartnerCode = strPtr(message.Partner.Code)
		}
		attention = append(attention, item)
	}
	for _, file := range summary.RecentQuarantined {
		item := &gqlmodel.EdiSummaryAttentionItem{
			Kind:       gqlmodel.EdiSummaryAttentionKindInboundFile,
			ID:         file.ID.String(),
			Reference:  strPtr(file.FileName),
			Error:      strPtr(file.FailureReason),
			OccurredAt: int(file.ReceivedAt),
		}
		if file.EDIPartnerID.IsNotNil() {
			partnerID := file.EDIPartnerID.String()
			item.PartnerID = &partnerID
		}
		if file.Partner != nil {
			item.PartnerName = strPtr(file.Partner.Name)
			item.PartnerCode = strPtr(file.Partner.Code)
		}
		attention = append(attention, item)
	}
	slices.SortFunc(attention, func(a, b *gqlmodel.EdiSummaryAttentionItem) int {
		return b.OccurredAt - a.OccurredAt
	})

	return &gqlmodel.EdiSummary{
		DeliveryStatusCounts:        summaryStatusCounts(summary.DeliveryStatusCounts),
		AckStatusCounts:             summaryStatusCounts(summary.AckStatusCounts),
		InboundFileStatusCounts:     summaryStatusCounts(summary.InboundFileStatusCounts),
		InboundTransferStatusCounts: summaryStatusCounts(summary.InboundTransferStatusCounts),
		OverdueAckCount:             summary.OverdueAckCount,
		AttentionItems:              attention,
	}
}

func summaryStatusCounts[T ~string](counts map[T]int) []*gqlmodel.EdiSummaryStatusCount {
	result := make([]*gqlmodel.EdiSummaryStatusCount, 0, len(counts))
	for status, count := range counts {
		result = append(result, &gqlmodel.EdiSummaryStatusCount{
			Status: string(status),
			Count:  count,
		})
	}
	slices.SortFunc(result, func(a, b *gqlmodel.EdiSummaryStatusCount) int {
		return strings.Compare(a.Status, b.Status)
	})
	return result
}

func strPtr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func ediTestCaseConnectionToModel(
	result *pagination.CursorListResult[*edi.EDITestCase],
) (*gqlmodel.EdiTestCaseConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *edi.EDITestCase, cursor string) *gqlmodel.EdiTestCaseEdge {
			return &gqlmodel.EdiTestCaseEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.EdiTestCaseEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.EdiTestCaseConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

func ediTemplateConnectionToModel(
	result *pagination.CursorListResult[*edi.EDITemplate],
) (*gqlmodel.EdiTemplateConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *edi.EDITemplate, cursor string) *gqlmodel.EdiTemplateEdge {
			return &gqlmodel.EdiTemplateEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.EdiTemplateEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.EdiTemplateConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

func ediMappingProfileConnectionToModel(
	result *pagination.CursorListResult[*edi.EDIMappingProfile],
) (*gqlmodel.EdiMappingProfileConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *edi.EDIMappingProfile, cursor string) *gqlmodel.EdiMappingProfileEdge {
			return &gqlmodel.EdiMappingProfileEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.EdiMappingProfileEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.EdiMappingProfileConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

func ediInboundFileConnectionToModel(
	result *pagination.CursorListResult[*edi.EDIInboundFile],
) (*gqlmodel.EdiInboundFileConnection, error) {
	page, err := entityCursorConnection(
		result,
		func(node *edi.EDIInboundFile, cursor string) *gqlmodel.EdiInboundFileEdge {
			return &gqlmodel.EdiInboundFileEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.EdiInboundFileEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}

	return &gqlmodel.EdiInboundFileConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}
