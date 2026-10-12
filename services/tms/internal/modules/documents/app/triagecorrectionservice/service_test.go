package triagecorrectionservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/testutil/mocks/triageportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

var tenantInfo = pagination.TenantInfo{OrgID: "org_test", BuID: "bu_test"}

type fakeTx struct {
	kernelports.DBConnection
}

func (fakeTx) WithTx(
	ctx context.Context,
	_ kernelports.TxOptions,
	fn func(context.Context, bun.Tx) error,
) error {
	return fn(ctx, bun.Tx{})
}

type fakeMessages struct {
	inboundmessageports.InboundMessageRepository

	reads []*inboundmessage.InboundMessage
	next  int
}

func (f *fakeMessages) GetByID(
	context.Context,
	inboundmessageports.GetInboundMessageByIDRequest,
) (*inboundmessage.InboundMessage, error) {
	read := f.reads[min(f.next, len(f.reads)-1)]
	f.next++
	copied := *read
	copied.Facts = append([]inboundmessage.Fact(nil), read.Facts...)

	return &copied, nil
}

type fakeTargets struct {
	inboundmessageports.TriageTargetRepository

	rows []*triage.ResponseTarget
}

func (f fakeTargets) List(context.Context, pagination.TenantInfo) ([]*triage.ResponseTarget, error) {
	return f.rows, nil
}

type fakeActivities struct {
	recorded []*triage.Activity
}

func (f *fakeActivities) RecordActivity(_ context.Context, activities ...*triage.Activity) error {
	f.recorded = append(f.recorded, activities...)

	return nil
}

type fakeNotifier struct {
	changed int
}

func (f *fakeNotifier) MessagesChanged(context.Context, ...*inboundmessage.InboundMessage) {
	f.changed++
}

type rig struct {
	service    *Service
	messages   *fakeMessages
	store      *triageportsmock.MockTriageMessageRepository
	hints      *triageportsmock.MockSenderHintRepository
	cache      *triageportsmock.MockTriageCache
	recorder   *triageportsmock.MockInboundCorrectionRecorder
	activities *fakeActivities
	notifier   *fakeNotifier
}

func newRig(t *testing.T, reads ...*inboundmessage.InboundMessage) *rig {
	t.Helper()

	r := &rig{
		messages:   &fakeMessages{reads: reads},
		store:      triageportsmock.NewMockTriageMessageRepository(t),
		hints:      triageportsmock.NewMockSenderHintRepository(t),
		cache:      triageportsmock.NewMockTriageCache(t),
		recorder:   triageportsmock.NewMockInboundCorrectionRecorder(t),
		activities: &fakeActivities{},
		notifier:   &fakeNotifier{},
	}
	r.service = &Service{
		l:        zap.NewNop(),
		db:       fakeTx{},
		messages: r.messages,
		store:    r.store,
		hints:    r.hints,
		targets: fakeTargets{rows: []*triage.ResponseTarget{
			{Classification: inboundmessage.ClassificationInvoice, Minutes: 90},
		}},
		activities: r.activities,
		notifier:   r.notifier,
		cache:      r.cache,
		recorder:   r.recorder,
		now:        func() int64 { return 1_760_000_000 },
	}

	return r
}

func readMessage() *inboundmessage.InboundMessage {
	return &inboundmessage.InboundMessage{
		ID:             "imsg_1",
		OrganizationID: "org_test",
		BusinessUnitID: "bu_test",
		FromAddress:    "ap@kestrel.example",
		SenderDomain:   "kestrel.example",
		Classification: inboundmessage.ClassificationOther,
		Confidence:     0.62,
		Status:         inboundmessage.StatusInReview,
		NovaRead:       true,
		Version:        4,
		Facts: []inboundmessage.Fact{
			{Key: "invoiceNumber", Label: "Invoice", Value: "KF-5521", Quote: "Invoice KF-5521"},
			{Key: "amount", Label: "Amount", Value: "$2,180.00", Quote: "$2,180.00"},
		},
	}
}

func TestCorrect_RelabelsRecomputesDueAndTeachesTheSender(t *testing.T) {
	t.Parallel()

	before := readMessage()
	after := readMessage()
	after.Classification = inboundmessage.ClassificationInvoice
	after.ClassificationCorrected = true
	r := newRig(t, before, after)

	r.store.EXPECT().
		CorrectClassification(mock.Anything, &triageports.CorrectClassificationRequest{
			TenantInfo:     tenantInfo,
			MessageID:      "imsg_1",
			Classification: inboundmessage.ClassificationInvoice,
			TargetSeconds:  90 * 60,
			At:             1_760_000_000,
		}).
		Return(true, nil).
		Once()
	r.hints.EXPECT().
		Upsert(mock.Anything, &triageports.UpsertSenderHintRequest{
			TenantInfo:     tenantInfo,
			SenderDomain:   "kestrel.example",
			Classification: inboundmessage.ClassificationInvoice,
			At:             1_760_000_000,
		}).
		Return(nil).
		Once()
	r.cache.EXPECT().DropSenderHints(mock.Anything, tenantInfo, "kestrel.example").Return(nil).Once()
	r.recorder.EXPECT().
		RecordInboundCorrection(mock.Anything, mock.MatchedBy(func(c *triageports.InboundCorrection) bool {
			return c.PredictedClassification == inboundmessage.ClassificationOther &&
				c.ConfirmedClassification == inboundmessage.ClassificationInvoice &&
				c.CapturedByID == "usr_dana" && len(c.Facts) == 2
		})).
		Return(nil).
		Once()

	updated, err := r.service.Correct(t.Context(), &CorrectRequest{
		TenantInfo:     tenantInfo,
		ActorID:        "usr_dana",
		MessageID:      "imsg_1",
		Classification: inboundmessage.ClassificationInvoice,
	})

	require.NoError(t, err)
	assert.Equal(t, inboundmessage.ClassificationInvoice, updated.Classification)
	require.Len(t, r.activities.recorded, 1)
	activity := r.activities.recorded[0]
	assert.Equal(t, triage.ActivityCorrected, activity.Kind)
	assert.Equal(t, triage.ActorPerson, activity.ActorKind)
	assert.Equal(t, inboundmessage.ClassificationOther, activity.FromClassification)
	assert.Equal(t, inboundmessage.ClassificationInvoice, activity.ToClassification)
	assert.Equal(t, 1, r.notifier.changed)
}

func TestCorrect_TheSameKindChangesNothing(t *testing.T) {
	t.Parallel()

	r := newRig(t, readMessage())

	updated, err := r.service.Correct(t.Context(), &CorrectRequest{
		TenantInfo:     tenantInfo,
		ActorID:        "usr_dana",
		MessageID:      "imsg_1",
		Classification: inboundmessage.ClassificationOther,
	})

	require.NoError(t, err)
	assert.Equal(t, inboundmessage.ClassificationOther, updated.Classification)
	assert.Empty(t, r.activities.recorded)
	assert.Zero(t, r.notifier.changed)
}

func TestCorrect_ASecondCorrectionRemembersWhatNovaFirstRead(t *testing.T) {
	t.Parallel()

	before := readMessage()
	before.Classification = inboundmessage.ClassificationInvoice
	before.ClassificationCorrected = true
	after := readMessage()
	after.Classification = inboundmessage.ClassificationRateConfirmation
	r := newRig(t, before, after)

	r.store.EXPECT().OriginalClassification(mock.Anything, triageports.FirstCorrectionRequest{
		TenantInfo: tenantInfo,
		MessageID:  "imsg_1",
	}).Return(inboundmessage.ClassificationOther, true, nil).Once()
	r.store.EXPECT().CorrectClassification(mock.Anything, mock.Anything).Return(true, nil).Once()
	r.hints.EXPECT().Upsert(mock.Anything, mock.Anything).Return(nil).Once()
	r.cache.EXPECT().DropSenderHints(mock.Anything, tenantInfo, "kestrel.example").Return(nil).Once()
	r.recorder.EXPECT().
		RecordInboundCorrection(mock.Anything, mock.MatchedBy(func(c *triageports.InboundCorrection) bool {
			return c.PredictedClassification == inboundmessage.ClassificationOther &&
				c.ConfirmedClassification == inboundmessage.ClassificationRateConfirmation
		})).
		Return(nil).
		Once()

	_, err := r.service.Correct(t.Context(), &CorrectRequest{
		TenantInfo:     tenantInfo,
		ActorID:        "usr_dana",
		MessageID:      "imsg_1",
		Classification: inboundmessage.ClassificationRateConfirmation,
	})

	require.NoError(t, err)
}

func TestCorrect_RefusesAKindThatDoesNotExist(t *testing.T) {
	t.Parallel()

	r := newRig(t, readMessage())

	_, err := r.service.Correct(t.Context(), &CorrectRequest{
		TenantInfo:     tenantInfo,
		MessageID:      "imsg_1",
		Classification: inboundmessage.Classification("Urgent"),
	})

	require.Error(t, err)
	assert.True(t, errortypes.IsError(err))
}

func TestFlagFact_FlagsTheFactAndRecordsTheCorrection(t *testing.T) {
	t.Parallel()

	r := newRig(t, readMessage())
	r.store.EXPECT().
		SaveTriageFields(mock.Anything, mock.MatchedBy(func(req *triageports.SaveTriageFieldsRequest) bool {
			return req.Version == 4 && req.Facts != nil && (*req.Facts)[1].Flagged &&
				!(*req.Facts)[0].Flagged && req.Conflicts == nil
		})).
		Return(nil).
		Once()
	r.recorder.EXPECT().
		RecordInboundCorrection(mock.Anything, mock.MatchedBy(func(c *triageports.InboundCorrection) bool {
			return c.PredictedClassification == c.ConfirmedClassification && c.Facts[1].Flagged
		})).
		Return(nil).
		Once()

	updated, err := r.service.FlagFact(t.Context(), &FlagRequest{
		TenantInfo: tenantInfo,
		ActorID:    "usr_dana",
		MessageID:  "imsg_1",
		Key:        "amount",
	})

	require.NoError(t, err)
	assert.True(t, updated.Facts[1].Flagged)
	assert.Equal(t, int64(5), updated.Version)
	require.Len(t, r.activities.recorded, 1)
	assert.Equal(t, triage.ActivityFactFlagged, r.activities.recorded[0].Kind)
	assert.Equal(t, "Amount", r.activities.recorded[0].Subject)
}

func TestFlagFact_TriesAgainWhenTheMessageMovedUnderIt(t *testing.T) {
	t.Parallel()

	stale := readMessage()
	fresh := readMessage()
	fresh.Version = 5
	r := newRig(t, stale, fresh)
	r.store.EXPECT().SaveTriageFields(mock.Anything, mock.MatchedBy(func(req *triageports.SaveTriageFieldsRequest) bool {
		return req.Version == 4
	})).Return(triageports.ErrMessageChanged).Once()
	r.store.EXPECT().SaveTriageFields(mock.Anything, mock.MatchedBy(func(req *triageports.SaveTriageFieldsRequest) bool {
		return req.Version == 5
	})).Return(nil).Once()
	r.recorder.EXPECT().RecordInboundCorrection(mock.Anything, mock.Anything).Return(nil).Once()

	updated, err := r.service.FlagFact(t.Context(), &FlagRequest{
		TenantInfo: tenantInfo,
		ActorID:    "usr_dana",
		MessageID:  "imsg_1",
		Key:        "invoiceNumber",
	})

	require.NoError(t, err)
	assert.Equal(t, int64(6), updated.Version)
}

func TestFlagFact_AnUnknownKeyIsNotFound(t *testing.T) {
	t.Parallel()

	r := newRig(t, readMessage())

	_, err := r.service.FlagFact(t.Context(), &FlagRequest{
		TenantInfo: tenantInfo,
		MessageID:  "imsg_1",
		Key:        "nope",
	})

	assert.True(t, errortypes.IsNotFoundError(err))
}

func TestFlagFact_AFlaggedFactStaysFlagged(t *testing.T) {
	t.Parallel()

	message := readMessage()
	message.Facts[0].Flagged = true
	r := newRig(t, message)

	updated, err := r.service.FlagFact(t.Context(), &FlagRequest{
		TenantInfo: tenantInfo,
		MessageID:  "imsg_1",
		Key:        "invoiceNumber",
	})

	require.NoError(t, err)
	assert.True(t, updated.Facts[0].Flagged)
	assert.Empty(t, r.activities.recorded)
}

func TestHintsFor_ReadsThroughTheCache(t *testing.T) {
	t.Parallel()

	hints := triageportsmock.NewMockSenderHintRepository(t)
	cache := triageportsmock.NewMockTriageCache(t)
	provider := NewHintProvider(HintProviderParams{Logger: zap.NewNop(), Hints: hints, Cache: cache})
	stored := []*triage.SenderHint{{SenderDomain: "kestrel.example", Classification: inboundmessage.ClassificationInvoice, Corrections: 3}}

	cache.EXPECT().SenderHints(mock.Anything, tenantInfo, "kestrel.example").Return(nil, false, nil).Once()
	hints.EXPECT().ListByDomain(mock.Anything, triageports.ListSenderHintsRequest{
		TenantInfo:   tenantInfo,
		SenderDomain: "kestrel.example",
		Limit:        MaxHintsPerSender,
	}).Return(stored, nil).Once()
	cache.EXPECT().SetSenderHints(mock.Anything, tenantInfo, "kestrel.example", stored).Return(nil).Once()

	got, err := provider.HintsFor(t.Context(), tenantInfo, " Kestrel.Example ")
	require.NoError(t, err)
	assert.Equal(t, stored, got)

	cache.EXPECT().SenderHints(mock.Anything, tenantInfo, "kestrel.example").Return(stored, true, nil).Once()
	got, err = provider.HintsFor(t.Context(), tenantInfo, "kestrel.example")
	require.NoError(t, err)
	assert.Equal(t, stored, got)
}

func TestHintsFor_NoDomainNoHints(t *testing.T) {
	t.Parallel()

	provider := NewHintProvider(HintProviderParams{Logger: zap.NewNop()})

	got, err := provider.HintsFor(t.Context(), tenantInfo, "")

	require.NoError(t, err)
	assert.Empty(t, got)
}
