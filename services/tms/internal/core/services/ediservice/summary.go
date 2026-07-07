package ediservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	defaultOverdueAckAgeSeconds = int64(4 * 60 * 60)
	defaultSummaryFeedLimit     = 10
)

type GetEDISummaryRequest struct {
	TenantInfo    pagination.TenantInfo
	Since         int64
	OverdueAckAge int64
	FeedLimit     int
}

type EDISummary struct {
	DeliveryStatusCounts        map[edi.MessageDeliveryStatus]int
	AckStatusCounts             map[edi.MessageAcknowledgmentStatus]int
	InboundFileStatusCounts     map[edi.InboundFileStatus]int
	InboundTransferStatusCounts map[edi.TransferStatus]int
	OverdueAckCount             int
	RecentDeadLettered          []*edi.EDIMessage
	RecentQuarantined           []*edi.EDIInboundFile
}

func (s *Service) GetEDISummary(
	ctx context.Context,
	req *GetEDISummaryRequest,
) (*EDISummary, error) {
	overdueAge := req.OverdueAckAge
	if overdueAge <= 0 {
		overdueAge = defaultOverdueAckAgeSeconds
	}
	feedLimit := req.FeedLimit
	if feedLimit <= 0 {
		feedLimit = defaultSummaryFeedLimit
	}
	countsReq := repositories.GetEDIMessageStatusCountsRequest{
		TenantInfo: req.TenantInfo,
		Since:      req.Since,
	}

	deliveryCounts, err := s.messageRepo.GetDeliveryStatusCounts(ctx, countsReq)
	if err != nil {
		return nil, err
	}
	ackCounts, err := s.messageRepo.GetAckStatusCounts(ctx, countsReq)
	if err != nil {
		return nil, err
	}
	overdueAcks, err := s.messageRepo.GetOverdueAckCount(
		ctx,
		repositories.GetEDIOverdueAckCountRequest{
			TenantInfo:   req.TenantInfo,
			PendingSince: timeutils.NowUnix() - overdueAge,
		},
	)
	if err != nil {
		return nil, err
	}
	fileCounts, err := s.inboundFileRepo.GetInboundFileStatusCounts(
		ctx,
		repositories.GetEDIInboundFileStatusCountsRequest{
			TenantInfo: req.TenantInfo,
			Since:      req.Since,
		},
	)
	if err != nil {
		return nil, err
	}
	transferCounts, err := s.transferRepo.GetInboundStatusCounts(
		ctx,
		repositories.GetEDITransferStatusCountsRequest{
			TenantInfo: req.TenantInfo,
			Since:      req.Since,
		},
	)
	if err != nil {
		return nil, err
	}
	deadLettered, err := s.messageRepo.ListRecentDeadLettered(
		ctx,
		&repositories.ListRecentEDIMessageFailuresRequest{
			TenantInfo: req.TenantInfo,
			Limit:      feedLimit,
		},
	)
	if err != nil {
		return nil, err
	}
	quarantined, err := s.inboundFileRepo.ListRecentQuarantined(
		ctx,
		repositories.ListRecentQuarantinedEDIInboundFilesRequest{
			TenantInfo: req.TenantInfo,
			Limit:      feedLimit,
		},
	)
	if err != nil {
		return nil, err
	}

	return &EDISummary{
		DeliveryStatusCounts:        deliveryCounts,
		AckStatusCounts:             ackCounts,
		InboundFileStatusCounts:     fileCounts,
		InboundTransferStatusCounts: transferCounts,
		OverdueAckCount:             overdueAcks,
		RecentDeadLettered:          deadLettered,
		RecentQuarantined:           quarantined,
	}, nil
}
