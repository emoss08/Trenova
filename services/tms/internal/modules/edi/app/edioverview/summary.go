package edioverview

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ediinboundfile"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edimessage"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edimessageports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editenderports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	DefaultOverdueAckAgeSeconds = int64(4 * 60 * 60)
	defaultSummaryFeedLimit     = 10

	defaultAckHealthWindowSeconds = int64(24 * 60 * 60)
	defaultAckHealthOverdueLimit  = 25
	MaxAckHealthOverdueLimit      = 100

	scorecardPendingOver4hSeconds  = int64(4 * 60 * 60)
	scorecardPendingOver24hSeconds = int64(24 * 60 * 60)
	defaultVolumeSeriesWindow      = int64(7 * 24 * 60 * 60)
	maxVolumeSeriesBuckets         = 60
	volumeBucketHour               = int64(60 * 60)
	volumeBucketSixHours           = int64(6 * 60 * 60)
	volumeBucketDay                = int64(24 * 60 * 60)
	volumeBucketWeek               = int64(7 * 24 * 60 * 60)
)

type GetEDISummaryRequest struct {
	TenantInfo    pagination.TenantInfo
	Since         int64
	OverdueAckAge int64
	FeedLimit     int
}

type GetEDIAttentionCountRequest struct {
	TenantInfo    pagination.TenantInfo
	OverdueAckAge int64
}

type EDISummary struct {
	DeliveryStatusCounts        map[edimessage.MessageDeliveryStatus]int
	AckStatusCounts             map[edimessage.MessageAcknowledgmentStatus]int
	InboundFileStatusCounts     map[ediinboundfile.InboundFileStatus]int
	InboundTransferStatusCounts map[editransfer.TransferStatus]int
	OverdueAckCount             int
	RecentDeadLettered          []*edimessage.EDIMessage
	RecentQuarantined           []*ediinboundfile.EDIInboundFile
}

func (s *Service) GetEDISummary(
	ctx context.Context,
	req *GetEDISummaryRequest,
) (*EDISummary, error) {
	overdueAge := overdueAckAge(req.OverdueAckAge)
	feedLimit := req.FeedLimit
	if feedLimit <= 0 {
		feedLimit = defaultSummaryFeedLimit
	}
	countsReq := &edimessageports.GetEDIMessageStatusCountsRequest{
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
		&edimessageports.GetEDIOverdueAckCountRequest{
			TenantInfo:   req.TenantInfo,
			PendingSince: timeutils.NowUnix() - overdueAge,
		},
	)
	if err != nil {
		return nil, err
	}
	fileCounts, err := s.inboundFileRepo.GetInboundFileStatusCounts(
		ctx,
		edimessageports.GetEDIInboundFileStatusCountsRequest{
			TenantInfo: req.TenantInfo,
			Since:      req.Since,
		},
	)
	if err != nil {
		return nil, err
	}
	transferCounts, err := s.transferRepo.GetInboundStatusCounts(
		ctx,
		editenderports.GetEDITransferStatusCountsRequest{
			TenantInfo: req.TenantInfo,
			Since:      req.Since,
		},
	)
	if err != nil {
		return nil, err
	}
	deadLettered, err := s.messageRepo.ListRecentDeadLettered(
		ctx,
		&edimessageports.ListRecentEDIMessageFailuresRequest{
			TenantInfo: req.TenantInfo,
			Limit:      feedLimit,
		},
	)
	if err != nil {
		return nil, err
	}
	quarantined, err := s.inboundFileRepo.ListRecentQuarantined(
		ctx,
		edimessageports.ListRecentQuarantinedEDIInboundFilesRequest{
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

type GetEDIAckHealthRequest struct {
	TenantInfo     pagination.TenantInfo
	PartnerID      pulid.ID
	TransactionSet edidocument.TransactionSet
	Since          int64
	OverdueAckAge  int64
	OverdueLimit   int
	FeedLimit      int
}

type EDIOverduePartner struct {
	PartnerID         pulid.ID
	PartnerName       string
	PartnerCode       string
	OverdueCount      int
	OldestGeneratedAt int64
	Messages          []*edimessage.EDIMessage
}

type EDIPartnerAckCounts struct {
	PartnerID   pulid.ID
	PartnerName string
	PartnerCode string
	Counts      map[edimessage.MessageAcknowledgmentStatus]int
}

type EDIAckHealth struct {
	AsOf int64
	// AnswersEverySet is set when the call asked about the acknowledgments
	// themselves (997 or 999), so the counts cover every set they answer.
	AnswersEverySet    bool
	Since              int64
	OverdueAckAge      int64
	PendingSince       int64
	AckStatusCounts    map[edimessage.MessageAcknowledgmentStatus]int
	AckCountsByPartner []*EDIPartnerAckCounts
	OverdueAckCount    int
	OverdueByPartner   []*EDIOverduePartner
	OverdueOmitted     int
	RecentDeadLettered []*edimessage.EDIMessage
}

func (s *Service) GetEDIAckHealth(
	ctx context.Context,
	req *GetEDIAckHealthRequest,
) (*EDIAckHealth, error) {
	now := timeutils.NowUnix()
	health := &EDIAckHealth{
		AsOf:            now,
		AnswersEverySet: req.TransactionSet.IsAcknowledgment(),
		Since:           req.Since,
		OverdueAckAge:   overdueAckAge(req.OverdueAckAge),
	}
	set := req.TransactionSet
	if health.AnswersEverySet {
		set = ""
	}
	if health.Since <= 0 {
		health.Since = now - defaultAckHealthWindowSeconds
	}
	health.PendingSince = now - health.OverdueAckAge
	overdueLimit := req.OverdueLimit
	if overdueLimit <= 0 {
		overdueLimit = defaultAckHealthOverdueLimit
	}
	feedLimit := req.FeedLimit
	if feedLimit <= 0 {
		feedLimit = defaultSummaryFeedLimit
	}

	countsReq := &edimessageports.GetEDIMessageStatusCountsRequest{
		TenantInfo:     req.TenantInfo,
		Since:          health.Since,
		PartnerID:      req.PartnerID,
		TransactionSet: set,
	}
	counts, err := s.messageRepo.GetAckStatusCounts(ctx, countsReq)
	if err != nil {
		return nil, err
	}
	health.AckStatusCounts = counts

	byPartner, err := s.messageRepo.GetAckStatusCountsByPartner(ctx, countsReq)
	if err != nil {
		return nil, err
	}
	health.AckCountsByPartner = groupAckCounts(byPartner)

	health.OverdueAckCount, err = s.messageRepo.GetOverdueAckCount(
		ctx,
		&edimessageports.GetEDIOverdueAckCountRequest{
			TenantInfo:     req.TenantInfo,
			PendingSince:   health.PendingSince,
			PartnerID:      req.PartnerID,
			TransactionSet: set,
		},
	)
	if err != nil {
		return nil, err
	}

	overdue, err := s.messageRepo.ListOverdueAcks(ctx, &edimessageports.ListOverdueEDIAcksRequest{
		TenantInfo:     req.TenantInfo,
		PendingSince:   health.PendingSince,
		PartnerID:      req.PartnerID,
		TransactionSet: set,
		Limit:          min(overdueLimit, MaxAckHealthOverdueLimit),
	})
	if err != nil {
		return nil, err
	}
	health.OverdueByPartner, health.OverdueOmitted = groupOverdueAcks(overdue)

	health.RecentDeadLettered, err = s.messageRepo.ListRecentDeadLettered(
		ctx,
		&edimessageports.ListRecentEDIMessageFailuresRequest{
			TenantInfo:     req.TenantInfo,
			PartnerID:      req.PartnerID,
			TransactionSet: set,
			Limit:          feedLimit,
		},
	)
	if err != nil {
		return nil, err
	}

	return health, nil
}

func groupAckCounts(rows []*edimessageports.EDIPartnerAckStatusCount) []*EDIPartnerAckCounts {
	groups := make([]*EDIPartnerAckCounts, 0, len(rows))
	byPartner := make(map[pulid.ID]*EDIPartnerAckCounts, len(rows))
	for _, row := range rows {
		group, ok := byPartner[row.PartnerID]
		if !ok {
			group = &EDIPartnerAckCounts{
				PartnerID:   row.PartnerID,
				PartnerName: row.PartnerName,
				PartnerCode: row.PartnerCode,
				Counts:      make(map[edimessage.MessageAcknowledgmentStatus]int),
			}
			byPartner[row.PartnerID] = group
			groups = append(groups, group)
		}
		group.Counts[row.AckStatus] = row.Count
	}
	return groups
}

func groupOverdueAcks(
	overdue *edimessageports.OverdueEDIAcks,
) (groups []*EDIOverduePartner, omitted int) {
	if overdue == nil || len(overdue.Partners) == 0 {
		return []*EDIOverduePartner{}, 0
	}

	groups = make([]*EDIOverduePartner, 0, len(overdue.Partners))
	byPartner := make(map[pulid.ID]*EDIOverduePartner, len(overdue.Partners))
	for _, partner := range overdue.Partners {
		group := &EDIOverduePartner{
			PartnerID:         partner.PartnerID,
			PartnerName:       partner.PartnerName,
			PartnerCode:       partner.PartnerCode,
			OverdueCount:      partner.Count,
			OldestGeneratedAt: partner.OldestGeneratedAt,
			Messages:          make([]*edimessage.EDIMessage, 0, partner.Count),
		}
		groups = append(groups, group)
		byPartner[partner.PartnerID] = group
		omitted += partner.Count
	}
	for _, message := range overdue.Messages {
		if group, ok := byPartner[message.EDIPartnerID]; ok {
			group.Messages = append(group.Messages, message)
			omitted--
		}
	}

	return groups, max(omitted, 0)
}

func overdueAckAge(requested int64) int64 {
	if requested <= 0 {
		return DefaultOverdueAckAgeSeconds
	}

	return requested
}

func (s *Service) GetEDIAttentionCount(
	ctx context.Context,
	req *GetEDIAttentionCountRequest,
) (int, error) {
	overdueAge := overdueAckAge(req.OverdueAckAge)
	deliveryCounts, err := s.messageRepo.GetDeliveryStatusCounts(
		ctx,
		&edimessageports.GetEDIMessageStatusCountsRequest{TenantInfo: req.TenantInfo},
	)
	if err != nil {
		return 0, err
	}
	fileCounts, err := s.inboundFileRepo.GetInboundFileStatusCounts(
		ctx,
		edimessageports.GetEDIInboundFileStatusCountsRequest{TenantInfo: req.TenantInfo},
	)
	if err != nil {
		return 0, err
	}
	overdueAcks, err := s.messageRepo.GetOverdueAckCount(
		ctx,
		&edimessageports.GetEDIOverdueAckCountRequest{
			TenantInfo:   req.TenantInfo,
			PendingSince: timeutils.NowUnix() - overdueAge,
		},
	)
	if err != nil {
		return 0, err
	}

	return deliveryCounts[edimessage.MessageDeliveryStatusDeadLettered] +
		fileCounts[ediinboundfile.InboundFileStatusQuarantined] +
		overdueAcks, nil
}

type GetEDIPartnerScorecardsRequest struct {
	TenantInfo    pagination.TenantInfo
	Since         int64
	OverdueAckAge int64
}

func (s *Service) GetEDIPartnerScorecards(
	ctx context.Context,
	req *GetEDIPartnerScorecardsRequest,
) ([]*edimessageports.EDIPartnerScorecardRow, error) {
	overdueAge := overdueAckAge(req.OverdueAckAge)
	now := timeutils.NowUnix()
	return s.messageRepo.GetPartnerScorecards(ctx, &edimessageports.GetEDIPartnerScorecardsRequest{
		TenantInfo:             req.TenantInfo,
		Since:                  req.Since,
		OverdueAckPendingSince: now - overdueAge,
		PendingOver4hBefore:    now - scorecardPendingOver4hSeconds,
		PendingOver24hBefore:   now - scorecardPendingOver24hSeconds,
	})
}

type GetEDIVolumeSeriesRequest struct {
	TenantInfo pagination.TenantInfo
	Since      int64
}

type EDIVolumeSeries struct {
	BucketSeconds int64
	Points        []*edimessageports.EDIVolumePoint
}

func (s *Service) GetEDIVolumeSeries(
	ctx context.Context,
	req *GetEDIVolumeSeriesRequest,
) (*EDIVolumeSeries, error) {
	since := req.Since
	if since <= 0 {
		since = timeutils.NowUnix() - defaultVolumeSeriesWindow
	}
	bucket := volumeBucketFor(timeutils.NowUnix() - since)
	points, err := s.messageRepo.GetVolumeSeries(ctx, &edimessageports.GetEDIVolumeSeriesRequest{
		TenantInfo:    req.TenantInfo,
		Since:         since,
		BucketSeconds: bucket,
	})
	if err != nil {
		return nil, err
	}
	return &EDIVolumeSeries{BucketSeconds: bucket, Points: points}, nil
}

func volumeBucketFor(windowSeconds int64) int64 {
	for _, bucket := range []int64{
		volumeBucketHour,
		volumeBucketSixHours,
		volumeBucketDay,
		volumeBucketWeek,
	} {
		if windowSeconds/bucket <= maxVolumeSeriesBuckets {
			return bucket
		}
	}
	return volumeBucketWeek
}
