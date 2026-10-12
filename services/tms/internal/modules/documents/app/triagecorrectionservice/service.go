package triagecorrectionservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const saveAttempts = 3

type Params struct {
	fx.In

	Logger     *zap.Logger
	DB         kernelports.DBConnection
	Messages   inboundmessageports.InboundMessageRepository
	Store      triageports.TriageMessageRepository
	Hints      triageports.SenderHintRepository
	Targets    inboundmessageports.TriageTargetRepository
	Activities inboundmessageports.TriageActivityRecorder
	Notifier   inboundmessageports.TriageMessageNotifier
	Cache      triageports.TriageCache               `optional:"true"`
	Recorder   triageports.InboundCorrectionRecorder `optional:"true"`
}

type Service struct {
	l          *zap.Logger
	db         kernelports.DBConnection
	messages   inboundmessageports.InboundMessageRepository
	store      triageports.TriageMessageRepository
	hints      triageports.SenderHintRepository
	targets    inboundmessageports.TriageTargetRepository
	activities inboundmessageports.TriageActivityRecorder
	notifier   inboundmessageports.TriageMessageNotifier
	cache      triageports.TriageCache
	recorder   triageports.InboundCorrectionRecorder
	now        func() int64
}

func New(p Params) *Service { //nolint:gocritic // fx params are passed by value
	return &Service{
		l:          p.Logger.Named("service.triage-correction"),
		db:         p.DB,
		messages:   p.Messages,
		store:      p.Store,
		hints:      p.Hints,
		targets:    p.Targets,
		activities: p.Activities,
		notifier:   p.Notifier,
		cache:      p.Cache,
		recorder:   p.Recorder,
		now:        timeutils.NowUnix,
	}
}

type CorrectRequest struct {
	TenantInfo     pagination.TenantInfo
	ActorID        pulid.ID
	MessageID      pulid.ID
	Classification inboundmessage.Classification
}

type FlagRequest struct {
	TenantInfo pagination.TenantInfo
	ActorID    pulid.ID
	MessageID  pulid.ID
	Key        string
}

func (s *Service) Correct(
	ctx context.Context,
	req *CorrectRequest,
) (*inboundmessage.InboundMessage, error) {
	if !req.Classification.IsValid() {
		return nil, errortypes.NewValidationError(
			"classification", errortypes.ErrInvalid, "Kind is not a known one",
		)
	}

	message, err := s.read(ctx, req.TenantInfo, req.MessageID)
	if err != nil {
		return nil, err
	}
	if message.Classification == req.Classification {
		return message, nil
	}

	predicted, err := s.predicted(ctx, message)
	if err != nil {
		return nil, err
	}

	rows, err := s.targets.List(ctx, req.TenantInfo)
	if err != nil {
		return nil, fmt.Errorf("read the response targets: %w", err)
	}
	minutes := triage.ResolveTargets(rows).For(req.Classification)
	at := s.now()

	changed := false
	err = s.db.WithTx(ctx, kernelports.TxOptions{}, func(ctx context.Context, _ bun.Tx) error {
		var txErr error
		changed, txErr = s.store.CorrectClassification(
			ctx,
			&triageports.CorrectClassificationRequest{
				TenantInfo:     req.TenantInfo,
				MessageID:      message.ID,
				Classification: req.Classification,
				TargetSeconds:  int64(minutes) * 60,
				At:             at,
			},
		)
		if txErr != nil || !changed {
			return txErr
		}

		corrected := triage.NewActivity(message, triage.ActivityCorrected).
			ByPerson(req.ActorID).
			About(string(req.Classification), message.SenderDomain)
		corrected.FromClassification = message.Classification
		corrected.ToClassification = req.Classification
		if txErr = s.activities.RecordActivity(ctx, corrected); txErr != nil {
			return txErr
		}
		if message.SenderDomain == "" {
			return nil
		}

		return s.hints.Upsert(ctx, &triageports.UpsertSenderHintRequest{
			TenantInfo:     req.TenantInfo,
			SenderDomain:   message.SenderDomain,
			Classification: req.Classification,
			At:             at,
		})
	})
	if err != nil {
		return nil, err
	}

	updated, err := s.read(ctx, req.TenantInfo, req.MessageID)
	if err != nil {
		return nil, err
	}
	if !changed {
		return updated, nil
	}

	s.dropHints(ctx, req.TenantInfo, message.SenderDomain)
	s.record(ctx, &triageports.InboundCorrection{
		TenantInfo:              req.TenantInfo,
		MessageID:               updated.ID,
		CapturedByID:            req.ActorID,
		SenderDomain:            updated.SenderDomain,
		PredictedClassification: predicted,
		ConfirmedClassification: updated.Classification,
		PredictedConfidence:     updated.Confidence,
		Facts:                   updated.Facts,
		CapturedAt:              at,
	})
	s.notifier.MessagesChanged(ctx, updated)

	return updated, nil
}

func (s *Service) FlagFact(
	ctx context.Context,
	req *FlagRequest,
) (*inboundmessage.InboundMessage, error) {
	for attempt := range saveAttempts {
		message, err := s.read(ctx, req.TenantInfo, req.MessageID)
		if err != nil {
			return nil, err
		}

		facts := append([]inboundmessage.Fact(nil), message.Facts...)
		index := factIndex(facts, req.Key)
		if index < 0 {
			return nil, errortypes.NewNotFoundError("Nova did not read that from this message")
		}
		if facts[index].Flagged {
			return message, nil
		}
		facts[index].Flagged = true

		err = s.store.SaveTriageFields(ctx, &triageports.SaveTriageFieldsRequest{
			TenantInfo: req.TenantInfo,
			MessageID:  message.ID,
			Version:    message.Version,
			Facts:      &facts,
		})
		if errors.Is(err, triageports.ErrMessageChanged) && attempt < saveAttempts-1 {
			continue
		}
		if err != nil {
			return nil, err
		}

		message.Version++
		message.Facts = facts
		s.afterFlag(ctx, req, message, &facts[index])

		return message, nil
	}

	return nil, errortypes.NewConflictError(
		"The message kept changing while the fact was being flagged. Try again.",
	)
}

func (s *Service) afterFlag(
	ctx context.Context,
	req *FlagRequest,
	message *inboundmessage.InboundMessage,
	fact *inboundmessage.Fact,
) {
	flagged := triage.NewActivity(message, triage.ActivityFactFlagged).
		ByPerson(req.ActorID).
		About(fact.Label, fact.Value)
	if err := s.activities.RecordActivity(ctx, flagged); err != nil {
		s.l.Warn("could not record a flagged fact",
			zap.String("messageId", message.ID.String()), zap.Error(err))
	}

	predicted, err := s.predicted(ctx, message)
	if err != nil {
		s.l.Warn("could not read what Nova first read a message as",
			zap.String("messageId", message.ID.String()), zap.Error(err))
		predicted = message.Classification
	}
	s.record(ctx, &triageports.InboundCorrection{
		TenantInfo:              req.TenantInfo,
		MessageID:               message.ID,
		CapturedByID:            req.ActorID,
		SenderDomain:            message.SenderDomain,
		PredictedClassification: predicted,
		ConfirmedClassification: message.Classification,
		PredictedConfidence:     message.Confidence,
		Facts:                   message.Facts,
		CapturedAt:              s.now(),
	})
	s.notifier.MessagesChanged(ctx, message)
}

func (s *Service) read(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	messageID pulid.ID,
) (*inboundmessage.InboundMessage, error) {
	return s.messages.GetByID(ctx, inboundmessageports.GetInboundMessageByIDRequest{
		ID:                 messageID,
		TenantInfo:         tenantInfo,
		IncludeAttachments: true,
	})
}

func (s *Service) predicted(
	ctx context.Context,
	message *inboundmessage.InboundMessage,
) (inboundmessage.Classification, error) {
	if !message.ClassificationCorrected {
		return message.Classification, nil
	}

	original, found, err := s.store.OriginalClassification(ctx, triageports.FirstCorrectionRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: message.OrganizationID,
			BuID:  message.BusinessUnitID,
		},
		MessageID: message.ID,
	})
	if err != nil {
		return "", err
	}
	if !found {
		return message.Classification, nil
	}

	return original, nil
}

func (s *Service) record(ctx context.Context, correction *triageports.InboundCorrection) {
	if s.recorder == nil || correction.CapturedByID.IsNil() ||
		correction.PredictedClassification == "" || correction.ConfirmedClassification == "" {
		return
	}
	if err := s.recorder.RecordInboundCorrection(ctx, correction); err != nil {
		s.l.Warn("could not record the correction for Nova's training",
			zap.String("messageId", correction.MessageID.String()), zap.Error(err))
	}
}

func (s *Service) dropHints(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	senderDomain string,
) {
	if s.cache == nil || senderDomain == "" {
		return
	}
	if err := s.cache.DropSenderHints(ctx, tenantInfo, senderDomain); err != nil {
		s.l.Warn("could not drop the cached sender hints",
			zap.String("senderDomain", senderDomain), zap.Error(err))
	}
}

func factIndex(facts []inboundmessage.Fact, key string) int {
	for i := range facts {
		if facts[i].Key == key {
			return i
		}
	}

	return -1
}
