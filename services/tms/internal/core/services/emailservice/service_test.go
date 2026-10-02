package emailservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestHandleProviderEventUsesExplicitProviderMessageID(t *testing.T) {
	t.Parallel()

	tenantInfo := testTenantInfo()
	msg := testEmailMessage(tenantInfo)
	repo := &handleProviderEventRepo{
		message: msg,
	}
	svc := &Service{repo: repo}

	event := testEmailEvent(tenantInfo, email.EventTypeDelivered)
	event.Raw = map[string]any{
		"MessageID": "wrong-raw-provider-message-id",
	}

	err := svc.HandleProviderEvent(t.Context(), HandleProviderEventParams{
		TenantInfo:        tenantInfo,
		Event:             event,
		ProviderMessageID: msg.ProviderMessageID,
	})

	require.NoError(t, err)
	require.Equal(t, msg.ID, event.MessageID)
	require.Equal(t, msg.ProviderMessageID, repo.providerMessageIDLookup)
	require.Equal(t, email.MessageStatusDelivered, repo.updatedMessage.Status)
	require.Empty(t, repo.suppressions)
}

func TestUpsertAssignmentsRejectsInactiveProfile(t *testing.T) {
	t.Parallel()

	tenantInfo := testTenantInfo()
	profileID := pulid.MustNew("emlprof_")
	repo := &assignmentRepo{
		profiles: map[pulid.ID]*email.Profile{
			profileID: {
				ID:             profileID,
				BusinessUnitID: tenantInfo.BuID,
				OrganizationID: tenantInfo.OrgID,
				Status:         email.ProfileStatusInactive,
			},
		},
	}
	svc := &Service{
		repo:         repo,
		auditService: noopAuditService{},
		l:            zap.NewNop(),
	}

	_, err := svc.UpsertAssignments(
		t.Context(),
		tenantInfo,
		[]*email.ProfileAssignment{{
			Purpose:   email.PurposeBilling,
			ProfileID: profileID,
		}},
		pulid.MustNew("usr_"),
	)

	require.Error(t, err)
	require.Empty(t, repo.updatedAssignments)
}

func TestUpsertAssignmentsReplacesAssignmentSet(t *testing.T) {
	t.Parallel()

	tenantInfo := testTenantInfo()
	billingProfileID := pulid.MustNew("emlprof_")
	repo := &assignmentRepo{
		profiles: map[pulid.ID]*email.Profile{
			billingProfileID: {
				ID:             billingProfileID,
				BusinessUnitID: tenantInfo.BuID,
				OrganizationID: tenantInfo.OrgID,
				Status:         email.ProfileStatusActive,
			},
		},
	}
	svc := &Service{
		repo:         repo,
		auditService: noopAuditService{},
		l:            zap.NewNop(),
	}

	assignments, err := svc.UpsertAssignments(
		t.Context(),
		tenantInfo,
		[]*email.ProfileAssignment{{
			Purpose:   email.PurposeBilling,
			ProfileID: billingProfileID,
		}},
		pulid.MustNew("usr_"),
	)

	require.NoError(t, err)
	require.Len(t, repo.updatedAssignments, 1)
	require.Equal(t, email.PurposeBilling, repo.updatedAssignments[0].Purpose)
	require.Equal(t, billingProfileID, repo.updatedAssignments[0].ProfileID)
	require.Equal(t, repo.updatedAssignments, assignments)
}

type assignmentRepo struct {
	repositories.EmailRepository

	profiles           map[pulid.ID]*email.Profile
	updatedAssignments []*email.ProfileAssignment
}

func (r *assignmentRepo) GetProfile(
	_ context.Context,
	req repositories.GetEmailEntityRequest,
) (*email.Profile, error) {
	profile, ok := r.profiles[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("EmailProfile not found")
	}
	if profile.OrganizationID != req.TenantInfo.OrgID ||
		profile.BusinessUnitID != req.TenantInfo.BuID {
		return nil, errortypes.NewNotFoundError("EmailProfile not found")
	}
	return profile, nil
}

func (r *assignmentRepo) UpsertAssignments(
	_ context.Context,
	_ pagination.TenantInfo,
	assignments []*email.ProfileAssignment,
) ([]*email.ProfileAssignment, error) {
	r.updatedAssignments = append([]*email.ProfileAssignment{}, assignments...)
	return r.updatedAssignments, nil
}

type noopAuditService struct{ *mocks.MockAuditService }

func (noopAuditService) List(
	context.Context,
	*repositories.ListAuditEntriesRequest,
) (*pagination.ListResult[*audit.Entry], error) {
	return &pagination.ListResult[*audit.Entry]{Items: []*audit.Entry{}, Total: 0}, nil
}

func (noopAuditService) ListByResourceID(
	context.Context,
	*repositories.ListByResourceIDRequest,
) (*pagination.ListResult[*audit.Entry], error) {
	return &pagination.ListResult[*audit.Entry]{Items: []*audit.Entry{}, Total: 0}, nil
}

func (noopAuditService) GetByID(
	context.Context,
	repositories.GetAuditEntryByIDOptions,
) (*audit.Entry, error) {
	return nil, errortypes.NewNotFoundError("AuditEntry not found")
}

func (noopAuditService) LogAction(*services.LogActionParams, ...services.LogOption) error {
	return nil
}

func (noopAuditService) LogActions([]services.BulkLogEntry) error {
	return nil
}

func (noopAuditService) RegisterSensitiveFields(
	permission.Resource,
	[]services.SensitiveField,
) error {
	return nil
}

func TestHandleProviderEventCreatesSuppressionOnlyWithReason(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		eventType         email.EventType
		suppressionReason email.SuppressionReason
		expectedStatus    email.MessageStatus
		wantSuppression   bool
	}{
		{
			name:           "bounce without suppression reason",
			eventType:      email.EventTypeBounced,
			expectedStatus: email.MessageStatusBounced,
		},
		{
			name:              "hard bounce suppression",
			eventType:         email.EventTypeBounced,
			suppressionReason: email.SuppressionReasonHardBounce,
			expectedStatus:    email.MessageStatusBounced,
			wantSuppression:   true,
		},
		{
			name:              "complaint suppression",
			eventType:         email.EventTypeComplained,
			suppressionReason: email.SuppressionReasonComplaint,
			expectedStatus:    email.MessageStatusComplained,
			wantSuppression:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tenantInfo := testTenantInfo()
			msg := testEmailMessage(tenantInfo)
			event := testEmailEvent(tenantInfo, tt.eventType)
			event.MessageID = msg.ID
			repo := &handleProviderEventRepo{message: msg}
			svc := &Service{repo: repo}

			err := svc.HandleProviderEvent(t.Context(), HandleProviderEventParams{
				TenantInfo:        tenantInfo,
				Event:             event,
				SuppressionReason: tt.suppressionReason,
			})

			require.NoError(t, err)
			require.Equal(t, tt.expectedStatus, repo.updatedMessage.Status)
			if !tt.wantSuppression {
				require.Empty(t, repo.suppressions)
				return
			}
			require.Len(t, repo.suppressions, 1)
			require.Equal(t, tt.suppressionReason, repo.suppressions[0].Reason)
			require.Equal(t, event.ProviderEventID, repo.suppressions[0].SourceEventID)
		})
	}
}

type handleProviderEventRepo struct {
	repositories.EmailRepository

	message                 *email.Message
	updatedMessage          *email.Message
	providerMessageIDLookup string
	suppressions            []*email.Suppression
}

func (r *handleProviderEventRepo) GetMessage(
	_ context.Context,
	req repositories.GetEmailEntityRequest,
) (*email.Message, error) {
	if r.message == nil || r.message.ID != req.ID {
		return nil, errortypes.NewNotFoundError("EmailMessage not found")
	}
	return r.message, nil
}

func (r *handleProviderEventRepo) GetMessageByProviderID(
	_ context.Context,
	req repositories.GetEmailMessageByProviderIDRequest,
) (*email.Message, error) {
	r.providerMessageIDLookup = req.ProviderMessageID
	if r.message == nil || r.message.ProviderMessageID != req.ProviderMessageID {
		return nil, errortypes.NewNotFoundError("EmailMessage not found")
	}
	return r.message, nil
}

func (r *handleProviderEventRepo) UpdateMessage(
	_ context.Context,
	msg *email.Message,
) (*email.Message, error) {
	r.updatedMessage = msg
	return msg, nil
}

func (r *handleProviderEventRepo) CreateEvent(
	_ context.Context,
	event *email.Event,
) (bool, error) {
	return true, nil
}

func (r *handleProviderEventRepo) CreateSuppression(
	_ context.Context,
	suppression *email.Suppression,
) (*email.Suppression, error) {
	r.suppressions = append(r.suppressions, suppression)
	return suppression, nil
}

func testTenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}
}

func testEmailMessage(tenantInfo pagination.TenantInfo) *email.Message {
	return &email.Message{
		ID:                pulid.MustNew("emlmsg_"),
		BusinessUnitID:    tenantInfo.BuID,
		OrganizationID:    tenantInfo.OrgID,
		Provider:          email.ProviderPostmark,
		ProviderMessageID: "provider-message-id",
		Status:            email.MessageStatusSent,
	}
}

func testEmailEvent(
	tenantInfo pagination.TenantInfo,
	eventType email.EventType,
) *email.Event {
	return &email.Event{
		BusinessUnitID:  tenantInfo.BuID,
		OrganizationID:  tenantInfo.OrgID,
		Provider:        email.ProviderPostmark,
		ProviderEventID: "provider-event-id",
		Type:            eventType,
		Recipient:       "ops@example.com",
		Raw:             map[string]any{},
	}
}

func TestSendReturnsExistingMessageForRepeatedIdempotencyKey(t *testing.T) {
	t.Parallel()

	tenantInfo := testTenantInfo()
	existing := testEmailMessage(tenantInfo)
	existing.IdempotencyKey = "invoice-inv_1-part-1-1700000000"

	repo := mocks.NewMockEmailRepository(t)
	repo.EXPECT().
		GetMessageByIdempotencyKey(mock.Anything, repositories.GetEmailMessageByIdempotencyKeyRequest{
			IdempotencyKey: existing.IdempotencyKey,
			TenantInfo:     tenantInfo,
		}).
		Return(existing, nil).
		Once()

	svc := &Service{repo: repo, validator: NewValidator()}
	msg, err := svc.Send(t.Context(), &services.SendEmailRequest{
		TenantInfo:     tenantInfo,
		Purpose:        email.PurposeBilling,
		To:             []string{"billing@customer.example.com"},
		Subject:        "Invoice INV-1",
		Text:           "Attached.",
		IdempotencyKey: existing.IdempotencyKey,
	})

	require.NoError(t, err)
	require.Same(t, existing, msg)
}

type senderOriginRepo struct {
	repositories.EmailRepository

	profile *email.Profile
}

func (r *senderOriginRepo) GetProfile(
	_ context.Context,
	_ repositories.GetEmailEntityRequest,
) (*email.Profile, error) {
	if r.profile == nil {
		return nil, errortypes.NewNotFoundError("EmailProfile not found")
	}
	return r.profile, nil
}

func TestDescribeSenderRejectionNamesWhereTheAddressIsSet(t *testing.T) {
	t.Parallel()

	tenantInfo := testTenantInfo()
	msg := testEmailMessage(tenantInfo)
	msg.FromEmail = "mailbox@trenova.app"
	svc := &Service{
		repo: &senderOriginRepo{profile: &email.Profile{
			Name:        "Billing",
			SenderEmail: "mailbox@trenova.app",
		}},
		l: zap.NewNop(),
	}

	err := svc.describeSenderRejection(t.Context(), msg, "", &SenderRejectedError{
		Provider:  email.ProviderResend,
		Rejection: SenderDomainUnverified,
	})
	require.ErrorContains(t, err, "from mailbox@trenova.app")
	require.ErrorContains(t, err, `the sender email of the "Billing" email profile`)

	err = svc.describeSenderRejection(
		t.Context(),
		msg,
		"the From address on the Email profile tab of customer ACME",
		&SenderRejectedError{Provider: email.ProviderResend, Rejection: SenderDomainUnverified},
	)
	require.ErrorContains(t, err, "This address comes from the From address on the Email profile tab of customer ACME.")

	plain := errors.New("network down")
	require.Same(t, plain, svc.describeSenderRejection(t.Context(), msg, "", plain))
}

func TestSyncInvoiceAttemptsPublishesEachUpdatedInvoice(t *testing.T) {
	t.Parallel()

	tenantInfo := testTenantInfo()
	msg := testEmailMessage(tenantInfo)
	updated := &invoice.Invoice{
		ID:             pulid.MustNew("inv_"),
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		Number:         "INV-1",
		SendStatus:     invoice.SendStatusSent,
	}

	invoiceRepo := mocks.NewMockInvoiceRepository(t)
	invoiceRepo.EXPECT().
		SyncEmailAttemptsForMessage(mock.Anything, msg.ID, tenantInfo).
		Return([]*invoice.Invoice{updated}, nil).
		Once()

	realtime := mocks.NewMockRealtimeService(t)
	realtime.EXPECT().
		PublishResourceInvalidation(mock.Anything, mock.MatchedBy(
			func(req *services.PublishResourceInvalidationRequest) bool {
				return req.Resource == "invoice" &&
					req.Action == "send.updated" &&
					req.RecordID == updated.ID
			},
		)).
		Return(nil).
		Once()

	svc := &Service{invoiceRepo: invoiceRepo, realtime: realtime, l: zap.NewNop()}
	svc.syncInvoiceAttempts(t.Context(), msg)
}
