package detentionservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var errPreviewWrote = errors.New("a preview wrote")

// previewOccurrenceRepo serves one occurrence and records what is saved. With
// readOnly set, any save fails the test's call, which is how a preview is held
// to reading.
type previewOccurrenceRepo struct {
	repositories.DetentionOccurrenceRepository

	stored   detention.DetentionOccurrence
	sibling  *detention.DetentionOccurrence
	saved    []*detention.DetentionOccurrence
	readOnly bool
}

func (r *previewOccurrenceRepo) GetByID(
	context.Context,
	*repositories.GetDetentionOccurrenceByIDRequest,
) (*detention.DetentionOccurrence, error) {
	copied := r.stored

	return &copied, nil
}

func (r *previewOccurrenceRepo) GetByShipment(
	context.Context,
	*repositories.GetOccurrencesByShipmentRequest,
) ([]*detention.DetentionOccurrence, error) {
	stored := r.stored
	out := []*detention.DetentionOccurrence{&stored}
	if r.sibling != nil {
		out = append(out, r.sibling)
	}

	return out, nil
}

func (r *previewOccurrenceRepo) Update(
	_ context.Context,
	entity *detention.DetentionOccurrence,
) (*detention.DetentionOccurrence, error) {
	if r.readOnly {
		return nil, errPreviewWrote
	}
	r.saved = append(r.saved, entity)

	return entity, nil
}

func pendingCharge() detention.DetentionOccurrence {
	return detention.DetentionOccurrence{
		ID:                 pulid.MustNew("dto_"),
		OrganizationID:     pulid.MustNew("org_"),
		BusinessUnitID:     pulid.MustNew("bu_"),
		ShipmentID:         pulid.MustNew("shp_"),
		CustomerID:         pulid.MustNew("cus_"),
		Status:             detention.OccurrenceStatusPending,
		RequiresApproval:   true,
		BillableAmount:     decimal.NewFromInt(425),
		DriverPayAmount:    decimal.NewFromInt(100),
		Currency:           "USD",
		NotificationStatus: detention.NotificationStatusPending,
		FreeTimeExpiresAt:  1_767_225_600,
		PolicySnapshot:     &detention.PolicySnapshot{Currency: "USD"},
		LocationName:       "Memphis DC",
		ShipmentProNumber:  "SHP-1001",
		Version:            4,
	}
}

func previewService(repo *previewOccurrenceRepo) *Service {
	return &Service{
		l:              zap.NewNop(),
		occurrenceRepo: repo,
		now:            func() int64 { return 1_767_230_000 },
	}
}

func tenantOfOccurrence(occ *detention.DetentionOccurrence) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: occ.OrganizationID, BuID: occ.BusinessUnitID}
}

func TestPreviewWaive_IsWhatWaiveSaves(t *testing.T) {
	t.Parallel()

	stored := pendingCharge()
	repo := &previewOccurrenceRepo{stored: stored, readOnly: true}
	svc := previewService(repo)
	params := WaiveParams{
		OccurrenceID: stored.ID,
		TenantInfo:   tenantOfOccurrence(&stored),
		Reason:       detention.WaiverReasonWeather,
		Note:         "Ice storm closed the yard",
		UserID:       pulid.MustNew("usr_"),
	}

	preview, err := svc.PreviewWaive(t.Context(), &params)
	require.NoError(t, err)
	require.Empty(t, repo.saved, "a preview must not save")

	repo.readOnly = false
	saved, err := svc.Waive(t.Context(), params)
	require.NoError(t, err)

	assert.Equal(t, saved, preview.After)
	assert.Equal(t, detention.OccurrenceStatusPending, preview.Before.Status)
	assert.True(t, preview.Before.BillableAmount.Equal(decimal.NewFromInt(425)))
	assert.True(t, preview.After.WaivedAmount.Equal(decimal.NewFromInt(425)))
	assert.True(t, preview.After.ChargedAmount().IsZero())
	require.Len(t, preview.Shipment, 1)
}

func TestPreviewApprove_IsWhatApproveSaves(t *testing.T) {
	t.Parallel()

	stored := pendingCharge()
	repo := &previewOccurrenceRepo{stored: stored, readOnly: true}
	svc := previewService(repo)
	params := ApproveParams{
		OccurrenceID: stored.ID,
		TenantInfo:   tenantOfOccurrence(&stored),
		UserID:       pulid.MustNew("usr_"),
		Note:         "ELD times hold up",
	}

	preview, err := svc.PreviewApprove(t.Context(), &params)
	require.NoError(t, err)

	repo.readOnly = false
	saved, err := svc.Approve(t.Context(), params)
	require.NoError(t, err)

	assert.Equal(t, saved, preview.After)
	assert.True(t, preview.Before.HoldsBilling())
	assert.False(t, preview.After.HoldsBilling())
}

func TestPreviewApprove_RefusesWhatApproveRefuses(t *testing.T) {
	t.Parallel()

	stored := pendingCharge()
	stored.Status = detention.OccurrenceStatusWaived
	svc := previewService(&previewOccurrenceRepo{stored: stored, readOnly: true})
	params := ApproveParams{OccurrenceID: stored.ID, TenantInfo: tenantOfOccurrence(&stored)}

	_, previewErr := svc.PreviewApprove(t.Context(), &params)
	_, executeErr := svc.Approve(t.Context(), params)

	require.Error(t, previewErr)
	require.Error(t, executeErr)
	assert.Equal(t, executeErr.Error(), previewErr.Error())
}

type noticeCustomers struct {
	repositories.CustomerRepository

	recipients string
}

func (r *noticeCustomers) GetByID(
	context.Context,
	repositories.GetCustomerByIDRequest,
) (*customer.Customer, error) {
	return &customer.Customer{
		Name:         "Acme Foods",
		EmailProfile: &customer.CustomerEmailProfile{ToRecipients: r.recipients},
	}, nil
}

type noticeMailer struct {
	sent []*services.SendEmailRequest
}

func (m *noticeMailer) Send(
	_ context.Context,
	req *services.SendEmailRequest,
) (*email.Message, error) {
	m.sent = append(m.sent, req)

	return &email.Message{}, nil
}

func (m *noticeMailer) SendPersisted(
	context.Context,
	*services.SendPersistedEmailRequest,
) (*email.Message, error) {
	return nil, nil
}

type noticeSenders struct{}

func (noticeSenders) ResolveSender(
	context.Context,
	*services.SendEmailRequest,
) (*services.EmailSender, error) {
	return &services.EmailSender{Email: "notices@carrier.test", Name: "Carrier Billing"}, nil
}

func TestPreviewOccurrenceNotice_IsTheNoticeSendOccurrenceNoticeSends(t *testing.T) {
	t.Parallel()

	stored := pendingCharge()
	policyID := pulid.MustNew("dtp_")
	stored.DetentionPolicyID = &policyID
	repo := &previewOccurrenceRepo{stored: stored, readOnly: true}
	mailer := &noticeMailer{}
	resolver := mocks.NewMockDocumentTemplateResolver(t)
	resolver.EXPECT().
		RenderMessage(mock.Anything, mock.Anything).
		Return(&services.RenderedMessage{
			Subject: "Detention charges started at Memphis DC",
			Text:    "Detention began at 09:00.",
			HTML:    "<p>Detention began at 09:00.</p>",
		}, nil)
	resolver.EXPECT().
		RenderDocument(mock.Anything, mock.Anything).
		Return(&services.RenderedDocument{PDF: []byte("%PDF-1.7")}, nil).
		Maybe()

	svc := previewService(repo)
	svc.customerRepo = &noticeCustomers{recipients: "ap@acme.test, ops@acme.test"}
	svc.noticeRepo = &stubNoticeRepo{}
	svc.policyRepo = stubPolicyRepo{
		policy: &detention.DetentionPolicy{ID: policyID, AttachNoticePDF: true},
	}
	svc.templates = resolver
	svc.contextBuilder = &ContextBuilder{}
	svc.emailService = mailer
	svc.senders = noticeSenders{}

	params := SendOccurrenceNoticeParams{
		OccurrenceID: stored.ID,
		TenantInfo:   tenantOfOccurrence(&stored),
		UserID:       pulid.MustNew("usr_"),
	}

	preview, err := svc.PreviewOccurrenceNotice(t.Context(), &params)
	require.NoError(t, err)
	require.Empty(t, mailer.sent, "a preview must not send")
	require.Empty(t, repo.saved, "a preview must not save")

	repo.readOnly = false
	updated, err := svc.SendOccurrenceNotice(t.Context(), params)
	require.NoError(t, err)
	require.Len(t, mailer.sent, 1)

	sent := mailer.sent[0]
	assert.Equal(t, sent.To, preview.Recipients)
	assert.Equal(t, sent.Subject, preview.Content.Subject)
	assert.Equal(t, sent.Text, preview.Content.Text)
	require.Len(t, sent.Attachments, 1)
	assert.Equal(t, sent.Attachments[0].FileName, preview.Attachment)
	assert.Equal(t, "Carrier Billing <notices@carrier.test>", preview.Sender.Address())
	assert.Equal(t, updated.NotificationStatus, preview.After.NotificationStatus)
	assert.Equal(t, updated.NoticeSentAt, preview.After.NoticeSentAt)
	assert.Equal(t, detention.NotificationStatusPending, preview.Before.NotificationStatus)
}
