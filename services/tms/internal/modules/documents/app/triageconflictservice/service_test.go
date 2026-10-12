package triageconflictservice

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/documents/app/triagesupport"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/documentshipmentdraft"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/internal/testutil/mocks/triageportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var tenantInfo = pagination.TenantInfo{OrgID: "org_test", BuID: "bu_test"}

type fakeDocuments struct {
	onFile []*document.Document
	asked  *documentports.GetDocumentsByResourceRequest
}

func (f *fakeDocuments) GetByResourceID(
	_ context.Context,
	req *documentports.GetDocumentsByResourceRequest,
) ([]*document.Document, error) {
	f.asked = req

	return f.onFile, nil
}

type fakeContents struct {
	documentports.DocumentContentService

	rates map[pulid.ID]string
}

func (f *fakeContents) GetShipmentDraft(
	_ context.Context,
	documentID pulid.ID,
	_ pagination.TenantInfo,
) (*documentshipmentdraft.DocumentShipmentDraft, error) {
	rate, ok := f.rates[documentID]
	if !ok {
		return nil, errortypes.NewNotFoundError("no draft")
	}

	return &documentshipmentdraft.DocumentShipmentDraft{
		DraftData: map[string]any{
			"fields": map[string]any{"rate": map[string]any{"value": rate}},
		},
	}, nil
}

type fakeMessages struct {
	inboundmessageports.InboundMessageRepository

	stored *inboundmessage.InboundMessage
}

func (f *fakeMessages) GetByID(
	context.Context,
	inboundmessageports.GetInboundMessageByIDRequest,
) (*inboundmessage.InboundMessage, error) {
	copied := *f.stored
	copied.Conflicts = append([]inboundmessage.Conflict(nil), f.stored.Conflicts...)

	return &copied, nil
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

type chicagoOrganizations struct{}

func (chicagoOrganizations) GetByID(
	context.Context,
	tenantports.GetOrganizationByIDRequest,
) (*tenant.Organization, error) {
	return &tenant.Organization{Timezone: "America/Chicago"}, nil
}

type rig struct {
	service    *Service
	store      *triageportsmock.MockTriageMessageRepository
	overlaps   *triageportsmock.MockTenderOverlapFinder
	documents  *fakeDocuments
	contents   *fakeContents
	messages   *fakeMessages
	activities *fakeActivities
	notifier   *fakeNotifier
}

func newRig(t *testing.T, message *inboundmessage.InboundMessage) *rig {
	t.Helper()

	r := &rig{
		store:      triageportsmock.NewMockTriageMessageRepository(t),
		overlaps:   triageportsmock.NewMockTenderOverlapFinder(t),
		documents:  &fakeDocuments{},
		contents:   &fakeContents{rates: map[pulid.ID]string{}},
		messages:   &fakeMessages{stored: message},
		activities: &fakeActivities{},
		notifier:   &fakeNotifier{},
	}
	now := time.Date(2026, time.October, 12, 15, 0, 0, 0, time.UTC).Unix()
	r.service = &Service{
		l:          zap.NewNop(),
		messages:   r.messages,
		store:      r.store,
		documents:  r.documents,
		activities: r.activities,
		notifier:   r.notifier,
		contents:   r.contents,
		overlaps:   r.overlaps,
		clock:      triagesupport.NewClock(chicagoOrganizations{}, func() int64 { return now }),
	}

	return r
}

func rateConMessage() *inboundmessage.InboundMessage {
	return &inboundmessage.InboundMessage{
		ID:                "imsg_rc",
		OrganizationID:    "org_test",
		BusinessUnitID:    "bu_test",
		Classification:    inboundmessage.ClassificationRateConfirmation,
		Status:            inboundmessage.StatusInReview,
		MatchedShipmentID: "shp_104251",
		Version:           3,
		Attachments: []*inboundmessage.InboundAttachment{
			{ID: "imsga_rc", DocumentID: "doc_new", Kind: inboundmessage.AttachmentRateConfirmation},
			{ID: "imsga_other", DocumentID: "doc_misc", Kind: inboundmessage.AttachmentOther},
		},
	}
}

func rateConOnFile() *document.Document {
	return &document.Document{
		ID:               "doc_old",
		LineageID:        "doc_old",
		IsCurrentVersion: true,
		Status:           document.StatusActive,
		CreatedAt:        1_759_700_000,
		DocumentType:     &documenttype.DocumentType{Code: "RATECONF"},
	}
}

func TestCheckConflicts_FindsADuplicateDocumentWithBothAmounts(t *testing.T) {
	t.Parallel()

	message := rateConMessage()
	r := newRig(t, message)
	r.documents.onFile = []*document.Document{rateConOnFile()}
	r.contents.rates = map[pulid.ID]string{"doc_old": "$1,180.00", "doc_new": "1240"}
	r.store.EXPECT().
		SaveTriageFields(mock.Anything, mock.MatchedBy(func(req *triageports.SaveTriageFieldsRequest) bool {
			return req.Version == 3 && req.Conflicts != nil && len(*req.Conflicts) == 1 && req.Facts == nil
		})).
		Return(nil).
		Once()

	conflicts, err := r.service.CheckConflicts(t.Context(), message)

	require.NoError(t, err)
	require.Len(t, conflicts, 1)
	conflict := conflicts[0]
	assert.Equal(t, "dup:imsga_rc", conflict.ID)
	assert.Equal(t, inboundmessage.ConflictDuplicateDocument, conflict.Kind)
	assert.Equal(t, pulid.ID("doc_old"), conflict.ExistingDocumentID)
	assert.Equal(t, pulid.ID("shp_104251"), conflict.ExistingShipmentID)
	assert.Equal(t, int64(1_759_700_000), conflict.ExistingFiledAt)
	assert.Equal(t, "1180.00", conflict.ExistingAmount)
	assert.Equal(t, "1240.00", conflict.NewAmount)
	assert.Equal(t, int64(4), message.Version)
	assert.Equal(t, inboundmessageports.DocumentResourceType, r.documents.asked.ResourceType)
}

func TestCheckConflicts_LeavesAmountsOutUnlessBothAreRead(t *testing.T) {
	t.Parallel()

	message := rateConMessage()
	r := newRig(t, message)
	r.documents.onFile = []*document.Document{rateConOnFile()}
	r.contents.rates = map[pulid.ID]string{"doc_new": "1240"}
	r.store.EXPECT().SaveTriageFields(mock.Anything, mock.Anything).Return(nil).Once()

	conflicts, err := r.service.CheckConflicts(t.Context(), message)

	require.NoError(t, err)
	require.Len(t, conflicts, 1)
	assert.Empty(t, conflicts[0].ExistingAmount)
	assert.Empty(t, conflicts[0].NewAmount)
}

func TestCheckConflicts_IgnoresWhatIsNotADuplicate(t *testing.T) {
	t.Parallel()

	tests := map[string]func(*inboundmessage.InboundMessage, *document.Document){
		"another kind on file":  func(_ *inboundmessage.InboundMessage, d *document.Document) { d.DocumentType.Code = "POD" },
		"an archived document":  func(_ *inboundmessage.InboundMessage, d *document.Document) { d.Status = document.StatusArchived },
		"an old version":        func(_ *inboundmessage.InboundMessage, d *document.Document) { d.IsCurrentVersion = false },
		"the attachment itself": func(_ *inboundmessage.InboundMessage, d *document.Document) { d.ID = "doc_new" },
		"an already filed file": func(m *inboundmessage.InboundMessage, _ *document.Document) {
			m.Attachments[0].FiledShipmentID = "shp_104251"
		},
		"an unreadable file":  func(m *inboundmessage.InboundMessage, _ *document.Document) { m.Attachments[0].FailureText = "refused" },
		"no matched shipment": func(m *inboundmessage.InboundMessage, _ *document.Document) { m.MatchedShipmentID = pulid.Nil },
		"a kind that never files": func(m *inboundmessage.InboundMessage, _ *document.Document) {
			m.Classification = inboundmessage.ClassificationStatusRequest
		},
	}

	for name, change := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			message := rateConMessage()
			onFile := rateConOnFile()
			change(message, onFile)
			r := newRig(t, message)
			r.documents.onFile = []*document.Document{onFile}

			conflicts, err := r.service.CheckConflicts(t.Context(), message)

			require.NoError(t, err)
			assert.Empty(t, conflicts)
			assert.Equal(t, int64(3), message.Version, "nothing changed, so nothing was written")
		})
	}
}

func TestCheckConflicts_KeepsADecisionAlreadyMade(t *testing.T) {
	t.Parallel()

	message := rateConMessage()
	message.Conflicts = []inboundmessage.Conflict{{
		ID:                 "dup:imsga_rc",
		Kind:               inboundmessage.ConflictDuplicateDocument,
		AttachmentID:       "imsga_rc",
		ExistingDocumentID: "doc_old",
		ExistingShipmentID: "shp_104251",
		ExistingFiledAt:    1_759_700_000,
		Resolution:         inboundmessage.ConflictKeepBoth,
		ResolvedAt:         1_759_900_000,
		ResolvedBy:         "usr_dana",
	}}
	r := newRig(t, message)
	r.documents.onFile = []*document.Document{rateConOnFile()}

	conflicts, err := r.service.CheckConflicts(t.Context(), message)

	require.NoError(t, err)
	require.Len(t, conflicts, 1)
	assert.Equal(t, inboundmessage.ConflictKeepBoth, conflicts[0].Resolution)
}

func tenderMessage() *inboundmessage.InboundMessage {
	return &inboundmessage.InboundMessage{
		ID:                "imsg_tender",
		OrganizationID:    "org_test",
		BusinessUnitID:    "bu_test",
		Classification:    inboundmessage.ClassificationTender,
		Status:            inboundmessage.StatusInReview,
		MatchedCustomerID: "cus_acme",
		Version:           1,
		LoadDraft: &inboundmessage.LoadDraft{Fields: []inboundmessage.LoadDraftField{
			{Key: inboundmessage.LoadFieldCustomer, Value: "Acme"},
			{Key: inboundmessage.LoadFieldOrigin, Value: "Columbus, OH"},
			{Key: inboundmessage.LoadFieldDestination, Value: "Louisville, KY"},
			{Key: inboundmessage.LoadFieldPickup, Value: "Oct 12"},
			{Key: inboundmessage.LoadFieldOrigin, Value: "Toledo, OH 43604"},
			{Key: inboundmessage.LoadFieldDestination, Value: "Lansing MI"},
			{Key: inboundmessage.LoadFieldPickup, Value: "10/13/2026 08:00"},
			{Key: inboundmessage.LoadFieldOrigin, Value: "Nowhere"},
			{Key: inboundmessage.LoadFieldDestination, Value: "Dayton, OH"},
			{Key: inboundmessage.LoadFieldPickup, Value: "Oct 14"},
		}},
	}
}

func TestCheckConflicts_FindsALoadTheTenderRepeats(t *testing.T) {
	t.Parallel()

	message := tenderMessage()
	r := newRig(t, message)
	chicago, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)
	r.overlaps.EXPECT().
		FindOverlappingLoads(mock.Anything, mock.MatchedBy(func(req *triageports.FindOverlappingLoadsRequest) bool {
			if req.CustomerID != "cus_acme" || len(req.Loads) != 2 {
				return false
			}
			second := req.Loads[1]

			return second.Index == 2 && second.OriginCity == "Toledo" && second.OriginState == "OH" &&
				second.DestinationCity == "Lansing" && second.DestinationState == "MI" &&
				second.PickupFrom == time.Date(2026, time.October, 13, 0, 0, 0, 0, chicago).Unix() &&
				second.PickupTo == time.Date(2026, time.October, 14, 0, 0, 0, 0, chicago).Unix()
		})).
		Return([]triageports.OverlappingLoad{{LoadIndex: 2, ShipmentID: "shp_104190"}}, nil).
		Once()
	r.store.EXPECT().SaveTriageFields(mock.Anything, mock.Anything).Return(nil).Once()

	conflicts, err := r.service.CheckConflicts(t.Context(), message)

	require.NoError(t, err)
	require.Len(t, conflicts, 1)
	assert.Equal(t, "load:2", conflicts[0].ID)
	assert.Equal(t, inboundmessage.ConflictOverlappingLoad, conflicts[0].Kind)
	assert.Equal(t, 2, conflicts[0].LoadIndex)
	assert.Equal(t, pulid.ID("shp_104190"), conflicts[0].ExistingShipmentID)
}

func TestCheckConflicts_OnlyComparesTendersForAKnownCustomer(t *testing.T) {
	t.Parallel()

	message := tenderMessage()
	message.MatchedCustomerID = pulid.Nil
	r := newRig(t, message)

	conflicts, err := r.service.CheckConflicts(t.Context(), message)

	require.NoError(t, err)
	assert.Empty(t, conflicts)
}

func TestResolve_SettlesAConflictTheWayItsKindAllows(t *testing.T) {
	t.Parallel()

	message := rateConMessage()
	message.Conflicts = []inboundmessage.Conflict{{
		ID:                 "dup:imsga_rc",
		Kind:               inboundmessage.ConflictDuplicateDocument,
		AttachmentID:       "imsga_rc",
		ExistingShipmentID: "shp_104251",
	}}
	r := newRig(t, message)
	r.store.EXPECT().
		SaveTriageFields(mock.Anything, mock.MatchedBy(func(req *triageports.SaveTriageFieldsRequest) bool {
			conflicts := *req.Conflicts

			return conflicts[0].Resolution == inboundmessage.ConflictReplaceExisting &&
				conflicts[0].ResolvedBy == "usr_dana" && conflicts[0].ResolvedAt > 0
		})).
		Return(nil).
		Once()

	updated, err := r.service.Resolve(t.Context(), &ResolveRequest{
		TenantInfo: tenantInfo,
		ActorID:    "usr_dana",
		MessageID:  "imsg_rc",
		ConflictID: "dup:imsga_rc",
		Resolution: inboundmessage.ConflictReplaceExisting,
	})

	require.NoError(t, err)
	assert.Equal(t, inboundmessage.ConflictReplaceExisting, updated.Conflicts[0].Resolution)
	require.Len(t, r.activities.recorded, 1)
	assert.Equal(t, triage.ActivityConflictResolved, r.activities.recorded[0].Kind)
	assert.Equal(t, 1, r.notifier.changed)
}

func TestResolve_RefusesAResolutionForAnotherKind(t *testing.T) {
	t.Parallel()

	message := rateConMessage()
	message.Conflicts = []inboundmessage.Conflict{{
		ID:   "dup:imsga_rc",
		Kind: inboundmessage.ConflictDuplicateDocument,
	}}
	r := newRig(t, message)

	_, err := r.service.Resolve(t.Context(), &ResolveRequest{
		TenantInfo: tenantInfo,
		MessageID:  "imsg_rc",
		ConflictID: "dup:imsga_rc",
		Resolution: inboundmessage.ConflictSkipDuplicate,
	})

	require.Error(t, err)
	assert.Empty(t, r.activities.recorded)
}

func TestResolve_RefusesOnceTheAttachmentIsFiled(t *testing.T) {
	t.Parallel()

	message := rateConMessage()
	message.Attachments[0].FiledShipmentID = "shp_104251"
	message.Conflicts = []inboundmessage.Conflict{{
		ID:           "dup:imsga_rc",
		Kind:         inboundmessage.ConflictDuplicateDocument,
		AttachmentID: "imsga_rc",
	}}
	r := newRig(t, message)

	_, err := r.service.Resolve(t.Context(), &ResolveRequest{
		TenantInfo: tenantInfo,
		MessageID:  "imsg_rc",
		ConflictID: "dup:imsga_rc",
		Resolution: inboundmessage.ConflictKeepBoth,
	})

	assert.True(t, errortypes.IsBusinessError(err))
}

func TestResolve_AnUnknownConflictIsNotFound(t *testing.T) {
	t.Parallel()

	r := newRig(t, rateConMessage())

	_, err := r.service.Resolve(t.Context(), &ResolveRequest{
		TenantInfo: tenantInfo,
		MessageID:  "imsg_rc",
		ConflictID: "dup:nope",
		Resolution: inboundmessage.ConflictKeepBoth,
	})

	assert.True(t, errortypes.IsNotFoundError(err))
}

func TestReopen_ClearsTheResolution(t *testing.T) {
	t.Parallel()

	message := tenderMessage()
	message.Conflicts = []inboundmessage.Conflict{{
		ID:         "load:2",
		Kind:       inboundmessage.ConflictOverlappingLoad,
		LoadIndex:  2,
		Resolution: inboundmessage.ConflictSkipDuplicate,
		ResolvedAt: 1,
		ResolvedBy: "usr_dana",
	}}
	r := newRig(t, message)
	r.store.EXPECT().
		SaveTriageFields(mock.Anything, mock.MatchedBy(func(req *triageports.SaveTriageFieldsRequest) bool {
			conflict := (*req.Conflicts)[0]

			return conflict.Resolution == "" && conflict.ResolvedAt == 0 && conflict.ResolvedBy.IsNil()
		})).
		Return(nil).
		Once()

	updated, err := r.service.Reopen(t.Context(), &ReopenRequest{
		TenantInfo: tenantInfo,
		ActorID:    "usr_dana",
		MessageID:  "imsg_tender",
		ConflictID: "load:2",
	})

	require.NoError(t, err)
	assert.Empty(t, updated.Conflicts[0].Resolution)
	require.Len(t, r.activities.recorded, 1)
	assert.Equal(t, triage.ActivityConflictReopened, r.activities.recorded[0].Kind)
}

func TestOptions_FollowTheKind(t *testing.T) {
	t.Parallel()

	assert.Equal(t,
		[]inboundmessage.ConflictResolution{inboundmessage.ConflictReplaceExisting, inboundmessage.ConflictKeepBoth},
		Options(inboundmessage.ConflictDuplicateDocument),
	)
	assert.Equal(t,
		[]inboundmessage.ConflictResolution{inboundmessage.ConflictSkipDuplicate, inboundmessage.ConflictConfirmNew},
		Options(inboundmessage.ConflictOverlappingLoad),
	)
}

func TestMerge_DropsOpenConflictsThatNoLongerHold(t *testing.T) {
	t.Parallel()

	current := []inboundmessage.Conflict{
		{ID: "dup:a", Kind: inboundmessage.ConflictDuplicateDocument},
		{ID: "dup:b", Kind: inboundmessage.ConflictDuplicateDocument, Resolution: inboundmessage.ConflictKeepBoth},
	}

	merged := Merge(current, nil)

	require.Len(t, merged, 1)
	assert.Equal(t, "dup:b", merged[0].ID)
}

func TestCheckMessage_LeavesSettledMailAlone(t *testing.T) {
	t.Parallel()

	message := rateConMessage()
	message.Status = inboundmessage.StatusActioned
	r := newRig(t, message)

	require.NoError(t, r.service.CheckMessage(t.Context(), CheckMessageRequest{
		TenantInfo: tenantInfo,
		MessageID:  "imsg_rc",
	}))
	assert.Nil(t, r.documents.asked)
	assert.Zero(t, r.notifier.changed)
}

func TestCheckMessage_PublishesWhatItFound(t *testing.T) {
	t.Parallel()

	r := newRig(t, rateConMessage())
	r.documents.onFile = []*document.Document{rateConOnFile()}
	r.store.EXPECT().SaveTriageFields(mock.Anything, mock.Anything).Return(nil).Once()

	require.NoError(t, r.service.CheckMessage(t.Context(), CheckMessageRequest{
		TenantInfo: tenantInfo,
		MessageID:  "imsg_rc",
	}))
	assert.Equal(t, 1, r.notifier.changed)
}
