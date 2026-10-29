package edioverview

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/modules/edi/app/edilading"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediprofilesecret"
	"github.com/emoss08/trenova/internal/modules/edi/app/editransport"
	"github.com/emoss08/trenova/internal/modules/edi/app/edivalidation"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ediinboundfile"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edimessage"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ladingproject"
	"github.com/emoss08/trenova/internal/modules/edi/domain/x12field"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediconnectiondraftports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edimessageports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/maputils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/smime"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	PartnerCardActivityWindowSeconds = int64(7 * 24 * 60 * 60)
	PartnerCardSeriesWindowSeconds   = int64(14 * 24 * 60 * 60)
	PartnerCardSeriesBucketSeconds   = volumeBucketHour
	PartnerCardAttentionLimit        = 5
)

type PartnerCertificateRole string

const (
	PartnerCertificateRoleLocal             = PartnerCertificateRole("Local")
	PartnerCertificateRolePartnerSigning    = PartnerCertificateRole("PartnerSigning")
	PartnerCertificateRolePartnerEncryption = PartnerCertificateRole("PartnerEncryption")
)

var partnerCertificateKeys = []struct {
	role PartnerCertificateRole
	key  string
}{
	{
		role: PartnerCertificateRolePartnerEncryption,
		key:  editransport.ConfigKeyPartnerEncryptionCertificate,
	},
	{
		role: PartnerCertificateRolePartnerSigning,
		key:  editransport.ConfigKeyPartnerSigningCertificate,
	},
	{role: PartnerCertificateRoleLocal, key: editransport.ConfigKeyLocalCertificate},
}

type GetEDIPartnerCardRequest struct {
	TenantInfo      pagination.TenantInfo
	PartnerID       pulid.ID
	IncludeCarriers bool
}

type EDIPartnerCertificate struct {
	Role       PartnerCertificateRole
	Summary    *smime.CertificateSummary
	Unreadable bool
}

type EDIPartnerTransport struct {
	Profile       *edi.EDICommunicationProfile
	Credentials   []ediconnectiondraftports.CredentialRequirement
	Certificates  []EDIPartnerCertificate
	LastPollError string
}

type EDIPartnerActivity struct {
	Since           int64
	SeriesSince     int64
	BucketSeconds   int64
	Scorecard       *edimessageports.EDIPartnerScorecardRow
	AckCounts       map[edimessage.MessageAcknowledgmentStatus]int
	OverdueAckCount int
	Sets            []*edimessageports.EDIPartnerSetActivityRow
	Series          []*edimessageports.EDIVolumePoint
}

type EDIPartnerAttention struct {
	OverdueAckAge   int64
	OverdueAcks     []*edimessage.EDIMessage
	OverdueAckCount int
	OldestOverdueAt int64
	DeadLettered    []*edimessage.EDIMessage
	Quarantined     []*ediinboundfile.EDIInboundFile
}

type EDIPartnerLading struct {
	Project *ladingproject.LadingProject
	Active  *ladingproject.LadingVersion
	Shadow  *ladingproject.LadingVersion
}

type EDIPartnerCard struct {
	Partner    *edi.EDIPartner
	Lading     *EDIPartnerLading
	Transports []*EDIPartnerTransport
	Readiness  *EDIPartnerReadiness
	Mappings   *edipartnerports.EDIMappingSummary
	Activity   *EDIPartnerActivity
	Attention  *EDIPartnerAttention
	Carriers   []*carrier.CarrierEDIChannel
}

func (s *Service) GetPartnerCard(
	ctx context.Context,
	req *GetEDIPartnerCardRequest,
) (*EDIPartnerCard, error) {
	partner, err := s.partnerRepo.GetByID(ctx, edipartnerports.GetEDIPartnerByIDRequest{
		ID:           req.PartnerID,
		TenantInfo:   req.TenantInfo,
		IncludeLinks: true,
	})
	if err != nil {
		return nil, err
	}

	card := &EDIPartnerCard{Partner: partner}
	if err = s.fillPartnerCardLading(ctx, req, card); err != nil {
		return nil, err
	}
	if card.Transports, err = s.partnerCardTransports(ctx, req); err != nil {
		return nil, err
	}
	if card.Readiness, err = s.partnerCardReadiness(ctx, req); err != nil {
		return nil, err
	}
	if card.Mappings, err = s.mappingProfileRepo.GetMappingSummary(
		ctx,
		edipartnerports.GetEDIMappingSummaryRequest{
			TenantInfo: req.TenantInfo,
			PartnerID:  req.PartnerID,
		},
	); err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	if card.Activity, err = s.partnerCardActivity(ctx, req, now); err != nil {
		return nil, err
	}
	if card.Attention, err = s.partnerCardAttention(ctx, req, now); err != nil {
		return nil, err
	}
	card.Activity.OverdueAckCount = card.Attention.OverdueAckCount

	if req.IncludeCarriers && s.carrierRepo != nil {
		if card.Carriers, err = s.carrierRepo.ListEDIChannelsByPartner(
			ctx,
			carrierports.ListCarrierEDIChannelsByPartnerRequest{
				TenantInfo:   req.TenantInfo,
				EDIPartnerID: req.PartnerID,
			},
		); err != nil {
			return nil, err
		}
	}

	return card, nil
}

func (s *Service) fillPartnerCardLading(
	ctx context.Context,
	req *GetEDIPartnerCardRequest,
	card *EDIPartnerCard,
) error {
	programs, err := s.lading.Load(ctx, req.TenantInfo, req.PartnerID)
	if errors.Is(err, edilading.ErrNoProject) {
		return nil
	}
	if err != nil {
		return err
	}
	card.Lading = &EDIPartnerLading{Project: programs.Project}
	if programs.Active != nil {
		card.Lading.Active = withoutFiles(programs.Active.Version)
	}
	if programs.Shadow != nil {
		card.Lading.Shadow = withoutFiles(programs.Shadow.Version)
	}
	return nil
}

func withoutFiles(version *ladingproject.LadingVersion) *ladingproject.LadingVersion {
	summary := *version
	summary.Files = nil
	summary.Samples = nil
	return &summary
}

func (s *Service) partnerCardTransports(
	ctx context.Context,
	req *GetEDIPartnerCardRequest,
) ([]*EDIPartnerTransport, error) {
	profiles, err := s.profileRepo.ListProfilesByPartner(
		ctx,
		edipartnerports.ListEDICommunicationProfilesByPartnerRequest{
			TenantInfo: req.TenantInfo,
			PartnerID:  req.PartnerID,
		},
	)
	if err != nil {
		return nil, err
	}

	transports := make([]*EDIPartnerTransport, 0, len(profiles))
	for _, profile := range profiles {
		transports = append(transports, partnerTransport(profile))
	}

	return transports, nil
}

func partnerTransport(profile *edi.EDICommunicationProfile) *EDIPartnerTransport {
	transport := &EDIPartnerTransport{
		Credentials:   edivalidation.CredentialRequirements(profile),
		Certificates:  partnerCertificates(profile),
		LastPollError: x12field.RedactSecrets(profile.LastPollError),
	}
	transport.Profile = ediprofilesecret.WithSecretState(profile)

	return transport
}

func partnerCertificates(profile *edi.EDICommunicationProfile) []EDIPartnerCertificate {
	if profile.Method != edi.ConnectionMethodAS2 {
		return []EDIPartnerCertificate{}
	}

	certificates := make([]EDIPartnerCertificate, 0, len(partnerCertificateKeys))
	for _, entry := range partnerCertificateKeys {
		pemData := strings.TrimSpace(maputils.StringValue(profile.Config, entry.key))
		if pemData == "" {
			continue
		}
		parsed, err := smime.ParseCertificate([]byte(pemData))
		if err != nil {
			certificates = append(
				certificates,
				EDIPartnerCertificate{Role: entry.role, Unreadable: true},
			)
			continue
		}
		summary := smime.SummarizeCertificate(parsed)
		certificates = append(
			certificates,
			EDIPartnerCertificate{Role: entry.role, Summary: &summary},
		)
	}

	return certificates
}

func (s *Service) partnerCardReadiness(
	ctx context.Context,
	req *GetEDIPartnerCardRequest,
) (*EDIPartnerReadiness, error) {
	states, err := s.GetPartnerReadiness(ctx, &GetEDIPartnerReadinessRequest{
		TenantInfo: req.TenantInfo,
		PartnerIDs: []pulid.ID{req.PartnerID},
	})
	if err != nil {
		return nil, err
	}
	for _, state := range states {
		if state.PartnerID == req.PartnerID {
			return state, nil
		}
	}

	return &EDIPartnerReadiness{
		PartnerID: req.PartnerID,
		Items:     []EDIPartnerReadinessItem{},
	}, nil
}

func (s *Service) partnerCardActivity(
	ctx context.Context,
	req *GetEDIPartnerCardRequest,
	now int64,
) (*EDIPartnerActivity, error) {
	activity := &EDIPartnerActivity{
		Since:         now - PartnerCardActivityWindowSeconds,
		SeriesSince:   now - PartnerCardSeriesWindowSeconds,
		BucketSeconds: PartnerCardSeriesBucketSeconds,
	}

	scorecards, err := s.messageRepo.GetPartnerScorecards(
		ctx,
		&edimessageports.GetEDIPartnerScorecardsRequest{
			TenantInfo:             req.TenantInfo,
			Since:                  activity.Since,
			PartnerID:              req.PartnerID,
			OverdueAckPendingSince: now - DefaultOverdueAckAgeSeconds,
			PendingOver4hBefore:    now - scorecardPendingOver4hSeconds,
			PendingOver24hBefore:   now - scorecardPendingOver24hSeconds,
		},
	)
	if err != nil {
		return nil, err
	}
	for _, row := range scorecards {
		if row.PartnerID == req.PartnerID {
			activity.Scorecard = row
			break
		}
	}
	if activity.Scorecard == nil {
		activity.Scorecard = &edimessageports.EDIPartnerScorecardRow{PartnerID: req.PartnerID}
	}

	if activity.AckCounts, err = s.messageRepo.GetAckStatusCounts(
		ctx,
		&edimessageports.GetEDIMessageStatusCountsRequest{
			TenantInfo: req.TenantInfo,
			Since:      activity.Since,
			PartnerID:  req.PartnerID,
		},
	); err != nil {
		return nil, err
	}

	if activity.Sets, err = s.messageRepo.GetPartnerSetActivity(
		ctx,
		&edimessageports.GetEDIPartnerSetActivityRequest{
			TenantInfo: req.TenantInfo,
			PartnerID:  req.PartnerID,
			Since:      activity.Since,
		},
	); err != nil {
		return nil, err
	}

	if activity.Series, err = s.messageRepo.GetVolumeSeries(
		ctx,
		&edimessageports.GetEDIVolumeSeriesRequest{
			TenantInfo:    req.TenantInfo,
			Since:         activity.SeriesSince,
			BucketSeconds: activity.BucketSeconds,
			PartnerID:     req.PartnerID,
		},
	); err != nil {
		return nil, err
	}

	return activity, nil
}

func (s *Service) partnerCardAttention(
	ctx context.Context,
	req *GetEDIPartnerCardRequest,
	now int64,
) (*EDIPartnerAttention, error) {
	attention := &EDIPartnerAttention{OverdueAckAge: DefaultOverdueAckAgeSeconds}

	overdue, err := s.messageRepo.ListOverdueAcks(ctx, &edimessageports.ListOverdueEDIAcksRequest{
		TenantInfo:   req.TenantInfo,
		PendingSince: now - attention.OverdueAckAge,
		PartnerID:    req.PartnerID,
		Limit:        PartnerCardAttentionLimit,
	})
	if err != nil {
		return nil, err
	}
	attention.OverdueAcks = []*edimessage.EDIMessage{}
	if overdue != nil {
		attention.OverdueAcks = overdue.Messages
		for _, group := range overdue.Partners {
			if group.PartnerID == req.PartnerID {
				attention.OverdueAckCount = group.Count
				attention.OldestOverdueAt = group.OldestGeneratedAt
			}
		}
	}

	since := now - PartnerCardActivityWindowSeconds
	if attention.DeadLettered, err = s.messageRepo.ListRecentDeadLettered(
		ctx,
		&edimessageports.ListRecentEDIMessageFailuresRequest{
			TenantInfo: req.TenantInfo,
			PartnerID:  req.PartnerID,
			Since:      since,
			Limit:      PartnerCardAttentionLimit,
		},
	); err != nil {
		return nil, err
	}

	if attention.Quarantined, err = s.inboundFileRepo.ListRecentQuarantined(
		ctx,
		edimessageports.ListRecentQuarantinedEDIInboundFilesRequest{
			TenantInfo: req.TenantInfo,
			PartnerID:  req.PartnerID,
			Since:      since,
			Limit:      PartnerCardAttentionLimit,
		},
	); err != nil {
		return nil, err
	}

	return attention, nil
}
