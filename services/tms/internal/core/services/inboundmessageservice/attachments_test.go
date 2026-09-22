package inboundmessageservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/documentupload"
	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type attachmentRepo struct {
	repositories.InboundMessageRepository

	rows []*inboundmessage.InboundAttachment
}

func (a *attachmentRepo) ListAttachments(
	_ context.Context, _ pulid.ID, _ pagination.TenantInfo,
) ([]*inboundmessage.InboundAttachment, error) {
	return a.rows, nil
}

func (a *attachmentRepo) UpdateAttachment(
	_ context.Context, entity *inboundmessage.InboundAttachment,
) (*inboundmessage.InboundAttachment, error) {
	return entity, nil
}

type docRepo struct {
	repositories.DocumentRepository

	doc *document.Document
}

func (d *docRepo) GetByID(
	_ context.Context, _ repositories.GetDocumentByIDRequest,
) (*document.Document, error) {
	return d.doc, nil
}

type uploads struct {
	services.DocumentUploadService

	refuse    error
	sessionID pulid.ID
	parts     int
}

func (u *uploads) CreateSession(
	_ context.Context, _ *services.CreateSessionRequest,
) (*documentupload.DocumentUploadSession, error) {
	if u.refuse != nil {
		return nil, u.refuse
	}

	return &documentupload.DocumentUploadSession{ID: u.sessionID}, nil
}

func (u *uploads) UploadPart(
	_ context.Context, _ *services.UploadPartRequest,
) (*documentupload.DocumentUploadSession, error) {
	u.parts++

	return &documentupload.DocumentUploadSession{ID: u.sessionID}, nil
}

func attachmentService(repo *attachmentRepo, up *uploads, docs *docRepo) *Service {
	svc := &Service{l: zap.NewNop(), messageRepo: repo}
	if up != nil {
		svc.uploads = up
	}
	if docs != nil {
		svc.documents = docs
	}

	return svc
}

func messageWithAttachment(name string) *inboundmessage.InboundMessage {
	return &inboundmessage.InboundMessage{
		ID:             pulid.MustNew("imsg_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		FromAddress:    "dispatch@bigshipper.com",
		Attachments: []*inboundmessage.InboundAttachment{
			{ID: pulid.MustNew("imsga_"), FileName: name, ContentType: "application/pdf"},
		},
	}
}

func TestStageAttachments_RecordsAFileTheValidatorRefuses(t *testing.T) {
	// A refused file has to leave a row saying so. Dropping it makes the
	// message's attachment count disagree with what the sender attached, and
	// nothing anywhere says why.
	repo := &attachmentRepo{}
	up := &uploads{refuse: errors.New("Executable files cannot be uploaded")}
	svc := attachmentService(repo, up, nil)

	message := messageWithAttachment("payload.exe")
	svc.stageAttachments(t.Context(), message, &providerMessage{
		Attachments: []providerAttachment{{FileName: "payload.exe", Content: []byte("MZ")}},
	})

	row := message.Attachments[0]
	assert.Contains(t, row.FailureText, "Executable files cannot be uploaded")
	assert.Equal(t, inboundmessage.AttachmentUnknown, row.Kind)
	assert.True(t, row.UploadSessionID.IsNil())
	assert.Zero(t, up.parts)
}

func TestStageAttachments_RecordsAnEmptyFileRatherThanUploadingIt(t *testing.T) {
	repo := &attachmentRepo{}
	up := &uploads{sessionID: pulid.MustNew("dus_")}
	svc := attachmentService(repo, up, nil)

	message := messageWithAttachment("rate-con.pdf")
	svc.stageAttachments(t.Context(), message, &providerMessage{
		Attachments: []providerAttachment{{FileName: "rate-con.pdf"}},
	})

	assert.NotEmpty(t, message.Attachments[0].FailureText)
	assert.Zero(t, up.parts, "an empty file is not worth a storage round trip")
}

func TestStageAttachments_KeepsTheSession(t *testing.T) {
	repo := &attachmentRepo{}
	sessionID := pulid.MustNew("dus_")
	up := &uploads{sessionID: sessionID}
	svc := attachmentService(repo, up, nil)

	message := messageWithAttachment("rate-con.pdf")
	svc.stageAttachments(t.Context(), message, &providerMessage{
		Attachments: []providerAttachment{{FileName: "rate-con.pdf", Content: []byte("%PDF-1.7")}},
	})

	assert.Equal(t, sessionID, message.Attachments[0].UploadSessionID)
	assert.Empty(t, message.Attachments[0].FailureText)
	assert.Equal(t, 1, up.parts)
}

func TestStageAttachments_IsASilentNoOpWithoutTheUploadService(t *testing.T) {
	// An installation without the document pipeline still receives mail. The
	// attachment rows record what arrived and say it was never read.
	svc := attachmentService(&attachmentRepo{}, nil, nil)

	message := messageWithAttachment("rate-con.pdf")
	svc.stageAttachments(t.Context(), message, &providerMessage{
		Attachments: []providerAttachment{{FileName: "rate-con.pdf", Content: []byte("%PDF-1.7")}},
	})

	assert.True(t, message.Attachments[0].UploadSessionID.IsNil())
}

func TestPendingAttachments_SkipsWhatIsDoneAndWhatNeverStarted(t *testing.T) {
	messageID := pulid.MustNew("imsg_")
	done := &inboundmessage.InboundAttachment{
		ID:              pulid.MustNew("imsga_"),
		UploadSessionID: pulid.MustNew("dus_"),
		DocumentID:      pulid.MustNew("doc_"),
	}
	refused := &inboundmessage.InboundAttachment{
		ID:          pulid.MustNew("imsga_"),
		FailureText: "Executable files cannot be uploaded",
	}
	owed := &inboundmessage.InboundAttachment{
		ID:              pulid.MustNew("imsga_"),
		FileName:        "rate-con.pdf",
		UploadSessionID: pulid.MustNew("dus_"),
	}
	svc := attachmentService(&attachmentRepo{
		rows: []*inboundmessage.InboundAttachment{done, refused, owed},
	}, nil, nil)

	refs, err := svc.PendingAttachments(t.Context(), messageID, pagination.TenantInfo{})
	require.NoError(t, err)
	require.Len(t, refs, 1)
	assert.Equal(t, owed.ID, refs[0].AttachmentID)
	assert.Equal(t, messageID, refs[0].MessageID,
		"the reference carries its message because that is how the row is read back")
}

func TestPollAttachmentExtraction_TreatsIndexedAsTheSuccessTerminal(t *testing.T) {
	// Extracted is a way-station the pipeline writes on the way past; Indexed
	// is where it settles. Polling for Extracted would wait forever.
	row := &inboundmessage.InboundAttachment{
		ID:         pulid.MustNew("imsga_"),
		DocumentID: pulid.MustNew("doc_"),
	}
	svc := attachmentService(
		&attachmentRepo{rows: []*inboundmessage.InboundAttachment{row}},
		nil,
		&docRepo{doc: &document.Document{
			ContentStatus: document.ContentStatusIndexed,
			DetectedKind:  "RateConfirmation",
		}},
	)

	state, err := svc.PollAttachmentExtraction(t.Context(),
		AttachmentRef{AttachmentID: row.ID}, pagination.TenantInfo{})
	require.NoError(t, err)
	assert.True(t, state.Terminal)
	assert.Equal(t, inboundmessage.AttachmentRateConfirmation, state.Kind)
	assert.Equal(t, inboundmessage.AttachmentRateConfirmation, row.Kind)
}

func TestPollAttachmentExtraction_KeepsWaitingWhileTheDocumentIsStillMoving(t *testing.T) {
	for _, status := range []document.ContentStatus{
		document.ContentStatusPending,
		document.ContentStatusExtracting,
		document.ContentStatusExtracted,
	} {
		t.Run(string(status), func(t *testing.T) {
			row := &inboundmessage.InboundAttachment{
				ID:         pulid.MustNew("imsga_"),
				DocumentID: pulid.MustNew("doc_"),
			}
			svc := attachmentService(
				&attachmentRepo{rows: []*inboundmessage.InboundAttachment{row}},
				nil,
				&docRepo{doc: &document.Document{ContentStatus: status}},
			)

			state, err := svc.PollAttachmentExtraction(t.Context(),
				AttachmentRef{AttachmentID: row.ID}, pagination.TenantInfo{})
			require.NoError(t, err)
			assert.False(t, state.Terminal)
			assert.Empty(t, row.Kind, "nothing is decided until the pipeline settles")
		})
	}
}

func TestPollAttachmentExtraction_SaysSoWhenTheDocumentFailed(t *testing.T) {
	row := &inboundmessage.InboundAttachment{
		ID:         pulid.MustNew("imsga_"),
		DocumentID: pulid.MustNew("doc_"),
	}
	svc := attachmentService(
		&attachmentRepo{rows: []*inboundmessage.InboundAttachment{row}},
		nil,
		&docRepo{doc: &document.Document{ContentStatus: document.ContentStatusFailed}},
	)

	state, err := svc.PollAttachmentExtraction(t.Context(),
		AttachmentRef{AttachmentID: row.ID}, pagination.TenantInfo{})
	require.NoError(t, err)
	assert.True(t, state.Terminal)
	assert.NotEmpty(t, row.FailureText)
}

func TestGiveUpOnAttachment_CallsAStuckDocumentAFailure(t *testing.T) {
	// This is the case that actually happens: EnqueueExtraction returns nil on
	// every one of its gates, so a document whose extraction was never started
	// sits at Pending with no error anywhere.
	row := &inboundmessage.InboundAttachment{ID: pulid.MustNew("imsga_")}
	svc := attachmentService(
		&attachmentRepo{rows: []*inboundmessage.InboundAttachment{row}}, nil, nil,
	)

	require.NoError(t, svc.GiveUpOnAttachment(t.Context(),
		AttachmentRef{AttachmentID: row.ID},
		document.ContentStatusPending,
		pagination.TenantInfo{},
	))
	assert.Contains(t, row.FailureText, "pending")
	assert.Equal(t, inboundmessage.AttachmentUnknown, row.Kind)
}

func TestAttachmentKindFor(t *testing.T) {
	cases := map[string]inboundmessage.AttachmentKind{
		"":                  inboundmessage.AttachmentUnknown,
		"RateConfirmation":  inboundmessage.AttachmentRateConfirmation,
		"rate_confirmation": inboundmessage.AttachmentRateConfirmation,
		"ProofOfDelivery":   inboundmessage.AttachmentProofOfDelivery,
		"BillOfLading":      inboundmessage.AttachmentBillOfLading,
		"Invoice":           inboundmessage.AttachmentInvoice,
		// The pipeline did read the file and did decide something. Calling that
		// Unknown would claim otherwise.
		"Other":        inboundmessage.AttachmentOther,
		"SomethingNew": inboundmessage.AttachmentOther,
	}

	for detected, want := range cases {
		assert.Equal(t, want, attachmentKindFor(detected), detected)
	}
}
