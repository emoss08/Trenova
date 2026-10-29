package ediinboundpoll

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/modules/edi/app/ediinboundservice"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editransportports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

func (s *Service) ListPollableProfiles(ctx context.Context) ([]PollableProfile, error) {
	profiles, err := s.profileRepo.ListInboundPollingProfiles(ctx)
	if err != nil {
		return nil, err
	}
	pollable := make([]PollableProfile, 0, len(profiles))
	for _, profile := range profiles {
		if profile == nil {
			continue
		}
		pollable = append(pollable, PollableProfile{
			ProfileID: profile.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: profile.OrganizationID,
				BuID:  profile.BusinessUnitID,
			},
		})
	}
	return pollable, nil
}

func (s *Service) PollMailbox(
	ctx context.Context,
	req *PollMailboxRequest,
) (*PollMailboxResult, error) {
	if req == nil || req.ProfileID.IsNil() {
		return nil, errors.New("EDI communication profile ID is required for inbound polling")
	}
	profile, err := s.profileRepo.GetProfileByID(
		ctx,
		edipartnerports.GetEDICommunicationProfileByIDRequest{
			ID:         req.ProfileID,
			TenantInfo: req.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}
	if profile.EDIPartnerID.IsNil() {
		return nil, errors.New(
			"EDI communication profile requires a partner before inbound polling",
		)
	}
	secrets, err := s.ediService.ProfileTransportSecrets(profile)
	if err != nil {
		return nil, err
	}
	fetchReq := &editransportports.EDIInboundFetchRequest{Profile: profile, Secrets: secrets}
	remoteFiles, err := s.transport.FetchInbound(ctx, profile.Method, fetchReq)
	if err != nil {
		s.metrics.RecordInboundPoll(string(profile.Method), "failed")
		s.recordPollOutcome(ctx, profile, false, err.Error())
		return nil, err
	}
	s.metrics.RecordInboundPoll(string(profile.Method), "success")
	s.recordPollOutcome(ctx, profile, true, "")

	result := &PollMailboxResult{ProfileID: profile.ID}
	for _, remoteFile := range remoteFiles {
		if remoteFile == nil || strings.TrimSpace(remoteFile.Contents) == "" {
			result.SkippedFiles++
			continue
		}
		staged, skipped, stageErr := s.inbound.StageInboundFile(ctx, profile, remoteFile)
		if stageErr != nil {
			return result, stageErr
		}
		if skipped {
			result.SkippedFiles++
		} else {
			result.StagedFileIDs = append(result.StagedFileIDs, staged.ID)
		}
		if archiveErr := s.transport.ArchiveInbound(
			ctx,
			profile.Method,
			fetchReq,
			remoteFile.Path,
		); archiveErr != nil {
			s.l.Warn(
				"failed to archive processed inbound EDI file on remote mailbox",
				zap.String("profileId", profile.ID.String()),
				zap.String("remotePath", remoteFile.Path),
				zap.Error(archiveErr),
			)
		}
	}
	return result, nil
}

func (s *Service) recordPollOutcome(
	ctx context.Context,
	profile *edi.EDICommunicationProfile,
	success bool,
	pollErr string,
) {
	if err := s.profileRepo.RecordInboundPollOutcome(
		ctx,
		edipartnerports.RecordEDIProfilePollOutcomeRequest{
			ProfileID: profile.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: profile.OrganizationID,
				BuID:  profile.BusinessUnitID,
			},
			PolledAt: timeutils.NowUnix(),
			Success:  success,
			Error:    pollErr,
		},
	); err != nil {
		s.l.Warn(
			"failed to record EDI inbound poll outcome",
			zap.String("profileId", profile.ID.String()),
			zap.Error(err),
		)
	}
}

func (s *Service) PollAndProcessMailbox(
	ctx context.Context,
	req *PollMailboxRequest,
) (*PollMailboxResult, error) {
	result, err := s.PollMailbox(ctx, req)
	if err != nil {
		return nil, err
	}
	for _, fileID := range result.StagedFileIDs {
		if _, procErr := s.inbound.ProcessInboundFile(ctx, &ediinboundservice.ProcessInboundFileRequest{
			FileID:     fileID,
			TenantInfo: req.TenantInfo,
		}); procErr != nil {
			s.l.Error(
				"failed to process staged inbound EDI file during manual poll",
				zap.String("fileId", fileID.String()),
				zap.Error(procErr),
			)
		}
	}
	return result, nil
}

type PollMailboxRequest struct {
	ProfileID  pulid.ID              `json:"profileId"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}

type PollMailboxResult struct {
	ProfileID     pulid.ID   `json:"profileId"`
	StagedFileIDs []pulid.ID `json:"stagedFileIds"`
	SkippedFiles  int        `json:"skippedFiles"`
}

type PollableProfile struct {
	ProfileID  pulid.ID              `json:"profileId"`
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
}
