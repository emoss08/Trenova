package invoiceshareservice

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/notificationservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeShareRepo struct {
	mu        sync.Mutex
	upserted  []*invoice.InvoiceShare
	upsertErr error
	listed    []*invoice.InvoiceShare
}

func (f *fakeShareRepo) Upsert(_ context.Context, shares []*invoice.InvoiceShare) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.upsertErr != nil {
		return f.upsertErr
	}
	f.upserted = append(f.upserted, shares...)
	return nil
}

func (f *fakeShareRepo) ListByInvoiceID(
	_ context.Context,
	_ *repositories.ListInvoiceSharesRequest,
) ([]*invoice.InvoiceShare, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.listed != nil {
		return f.listed, nil
	}
	return f.upserted, nil
}

type harness struct {
	t             *testing.T
	svc           *Service
	cfg           *config.Config
	shares        *fakeShareRepo
	invoices      *mocks.MockInvoiceRepository
	users         *mocks.MockUserRepository
	orgs          *mocks.MockOrganizationRepository
	permissions   *mocks.MockPermissionEngine
	templates     *mocks.MockDocumentTemplateResolver
	emails        *mocks.MockEmailService
	audit         *mocks.MockAuditService
	notifications []*notification.Notification
	renders       []*servicesports.RenderMessageRequest
	sentEmails    []*servicesports.SendEmailRequest
	mu            sync.Mutex
	tenant        pagination.TenantInfo
	actor         *servicesports.RequestActor
	sharer        *tenant.User
	invoice       *invoice.Invoice
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	h := &harness{
		t:           t,
		cfg:         &config.Config{},
		shares:      &fakeShareRepo{},
		invoices:    mocks.NewMockInvoiceRepository(t),
		users:       mocks.NewMockUserRepository(t),
		orgs:        mocks.NewMockOrganizationRepository(t),
		permissions: mocks.NewMockPermissionEngine(t),
		templates:   mocks.NewMockDocumentTemplateResolver(t),
		emails:      mocks.NewMockEmailService(t),
		audit:       mocks.NewMockAuditService(t),
		tenant: pagination.TenantInfo{
			OrgID:  pulid.MustNew("org_"),
			BuID:   pulid.MustNew("bu_"),
			UserID: pulid.MustNew("usr_"),
		},
	}
	h.cfg.App.WebBaseURL = "https://app.example.com/"
	h.sharer = &tenant.User{
		ID:     h.tenant.UserID,
		Name:   "Marcus Bell",
		Status: domaintypes.StatusActive,
	}
	h.actor = &servicesports.RequestActor{
		PrincipalType:  servicesports.PrincipalTypeUser,
		PrincipalID:    h.sharer.ID,
		UserID:         h.sharer.ID,
		OrganizationID: h.tenant.OrgID,
		BusinessUnitID: h.tenant.BuID,
	}
	h.invoice = &invoice.Invoice{
		ID:             pulid.MustNew("inv_"),
		OrganizationID: h.tenant.OrgID,
		BusinessUnitID: h.tenant.BuID,
		Number:         "INV-2026-1042",
		BillToName:     "Halstead Grocery Group",
	}

	notificationRepo := mocks.NewMockNotificationRepository(t)
	notificationRepo.On("Create", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			entity, ok := args.Get(1).(*notification.Notification)
			require.True(t, ok)
			h.mu.Lock()
			h.notifications = append(h.notifications, entity)
			h.mu.Unlock()
		}).
		Return(func(_ context.Context, entity *notification.Notification) *notification.Notification {
			return entity
		}, nil).
		Maybe()

	h.templates.On("RenderMessage", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			req, ok := args.Get(1).(*servicesports.RenderMessageRequest)
			require.True(t, ok)
			h.mu.Lock()
			h.renders = append(h.renders, req)
			h.mu.Unlock()
		}).
		Return(func(_ context.Context, req *servicesports.RenderMessageRequest) *servicesports.RenderedMessage {
			return &servicesports.RenderedMessage{
				Subject: "subject:" + string(req.Kind),
				Text:    "text:" + string(req.Kind),
				HTML:    "html:" + string(req.Kind),
			}
		}, nil).
		Maybe()

	h.orgs.On("GetByID", mock.Anything, mock.Anything).
		Return(&tenant.Organization{Name: "Trenova Logistics"}, nil).
		Maybe()

	h.svc = New(Params{
		Logger:           zap.NewNop(),
		Config:           h.cfg,
		Repo:             h.shares,
		InvoiceRepo:      h.invoices,
		UserRepo:         h.users,
		OrganizationRepo: h.orgs,
		Permissions:      h.permissions,
		NotificationService: notificationservice.New(notificationservice.Params{
			Logger:   zap.NewNop(),
			Repo:     notificationRepo,
			Realtime: &mocks.NoopRealtimeService{},
		}),
		EmailService: h.emails,
		Templates:    h.templates,
		AuditService: h.audit,
	})

	return h
}

func (h *harness) recipient(name string) *tenant.User {
	return &tenant.User{
		ID:                    pulid.MustNew("usr_"),
		BusinessUnitID:        h.tenant.BuID,
		CurrentOrganizationID: h.tenant.OrgID,
		Name:                  name,
		Username:              strings.ToLower(strings.ReplaceAll(name, " ", "")),
		EmailAddress:          strings.ToLower(strings.ReplaceAll(name, " ", ".")) + "@example.com",
		Status:                domaintypes.StatusActive,
		Locale:                "es",
	}
}

func (h *harness) expectInvoiceAndSharer() {
	h.invoices.On("GetByID", mock.Anything, repositories.GetInvoiceByIDRequest{
		ID:         h.invoice.ID,
		TenantInfo: h.tenant,
	}).Return(h.invoice, nil)
	h.users.On("GetByID", mock.Anything, repositories.GetUserByIDRequest{
		TenantInfo:   h.tenant,
		LookupUserID: h.sharer.ID,
	}).Return(h.sharer, nil)
}

func (h *harness) expectUsers(users ...*tenant.User) {
	h.users.On("GetByIDs", mock.Anything, mock.MatchedBy(func(req repositories.GetUsersByIDsRequest) bool {
		return req.TenantInfo == h.tenant
	})).
		Return(users, nil)
}

func (h *harness) allow(users ...*tenant.User) {
	for _, user := range users {
		h.expectPermission(user, true)
	}
}

func (h *harness) expectPermission(user *tenant.User, allowed bool) {
	h.permissions.On(
		"Check",
		mock.MatchedBy(func(ctx context.Context) bool {
			_, hasActivation := authctx.GetSessionRoleActivation(ctx)
			return !hasActivation
		}),
		mock.MatchedBy(func(req *servicesports.PermissionCheckRequest) bool {
			return req.UserID == user.ID &&
				req.PrincipalID == user.ID &&
				req.PrincipalType == servicesports.PrincipalTypeUser &&
				req.OrganizationID == h.tenant.OrgID &&
				req.BusinessUnitID == h.tenant.BuID &&
				req.Resource == permission.ResourceInvoice.String() &&
				req.Operation == permission.OpRead
		}),
	).Return(&servicesports.PermissionCheckResult{Allowed: allowed}, nil)
}

func (h *harness) expectEmails(err error) {
	h.emails.On("Send", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			req, ok := args.Get(1).(*servicesports.SendEmailRequest)
			require.True(h.t, ok)
			h.mu.Lock()
			h.sentEmails = append(h.sentEmails, req)
			h.mu.Unlock()
		}).
		Return(&email.Message{}, err)
}

func (h *harness) expectAudit() {
	h.audit.On("LogAction", mock.Anything, mock.Anything, mock.Anything).Return(nil)
}

func (h *harness) request(
	userIDs []pulid.ID,
	note string,
	tab invoice.ShareTab,
) *servicesports.ShareInvoiceRequest {
	return &servicesports.ShareInvoiceRequest{
		TenantInfo: h.tenant,
		InvoiceID:  h.invoice.ID,
		UserIDs:    userIDs,
		Note:       note,
		Tab:        tab,
	}
}

func (h *harness) ctxWithSharerActivation(t *testing.T) context.Context {
	return authctx.WithSessionRoleActivation(t.Context(), []pulid.ID{pulid.MustNew("rol_")}, true)
}

func requireFieldError(t *testing.T, err error, field string) {
	t.Helper()

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	for _, fieldErr := range multiErr.Errors {
		if fieldErr.Field == field {
			return
		}
	}
	t.Fatalf("expected a field error on %q, got %v", field, multiErr.Errors)
}

func TestShareStoresNotifiesAndEmailsEveryRecipient(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	dana := h.recipient("Dana Whitfield")
	priya := h.recipient("Priya Nair")
	h.expectInvoiceAndSharer()
	h.expectUsers(dana, priya)
	h.allow(dana, priya)
	h.expectAudit()
	h.expectEmails(nil)

	result, err := h.svc.Share(
		h.ctxWithSharerActivation(t),
		h.request(
			[]pulid.ID{dana.ID, priya.ID, dana.ID},
			"  Check the detention line  ",
			invoice.ShareTabCharges,
		),
		h.actor,
	)
	require.NoError(t, err)

	assert.Equal(t, 2, result.RecipientCount)
	assert.Equal(t, 2, result.EmailsQueued)
	assert.Equal(t, servicesports.InvoiceShareEmailQueued, result.EmailStatus)
	require.Len(t, result.Shares, 2)

	require.Len(t, h.shares.upserted, 2)
	for _, share := range h.shares.upserted {
		assert.Equal(t, h.invoice.ID, share.InvoiceID)
		assert.Equal(t, h.sharer.ID, share.SharedByID)
		assert.Equal(t, "Check the detention line", share.Note)
		assert.Equal(t, invoice.ShareTabCharges, share.Tab)
		assert.Equal(t, 1, share.ShareCount)
	}

	require.Len(t, h.notifications, 2)
	for _, created := range h.notifications {
		assert.Equal(t, notification.ChannelUser, created.Channel)
		assert.Equal(t, invoiceSharedEventType, created.EventType)
		assert.Equal(
			t,
			"/billing/invoices?item="+h.invoice.ID.String()+"&tab=charges",
			created.Data["link"],
		)
		assert.NotContains(t, created.Title, "detention")
		assert.NotContains(t, created.Message, "detention")
		for _, value := range created.Data {
			assert.NotContains(t, value, "detention")
		}
	}

	require.Len(t, h.sentEmails, 2)
	assert.Equal(t, []string{dana.EmailAddress}, h.sentEmails[0].To)
	assert.Equal(t, email.PurposeNotifications, h.sentEmails[0].Purpose)
}

func TestShareRendersTheNoteOnlyIntoTheEmailInTheRecipientsLanguage(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	dana := h.recipient("Dana Whitfield")
	h.expectInvoiceAndSharer()
	h.expectUsers(dana)
	h.allow(dana)
	h.expectAudit()
	h.expectEmails(nil)

	_, err := h.svc.Share(
		t.Context(),
		h.request([]pulid.ID{dana.ID}, "Look at line 3", ""),
		h.actor,
	)
	require.NoError(t, err)

	require.Len(t, h.renders, 2)
	for _, render := range h.renders {
		assert.Equal(t, "es", render.Locale.String())
		assert.True(t, render.FallbackToBuiltIn)

		switch data := render.Data.(type) {
		case *documenttemplate.InvoiceShareNotificationContext:
			assert.Equal(t, documenttemplate.KindNotificationInvoiceShared, render.Kind)
			assert.Equal(t, "Marcus Bell", data.SharedByName)
		case *documenttemplate.InvoiceShareEmailContext:
			assert.Equal(t, documenttemplate.KindInvoiceShareEmail, render.Kind)
			assert.Equal(t, "Look at line 3", data.Note)
			assert.Equal(t, "Dana", data.RecipientFirstName)
			assert.Equal(t, "Trenova Logistics", data.CompanyName)
			assert.Equal(t,
				"https://app.example.com/billing/invoices?item="+h.invoice.ID.String(),
				string(data.InvoiceURL))
		default:
			t.Fatalf("unexpected render context %T", render.Data)
		}
	}
}

func TestShareSkipsEmailWhenTheWebBaseURLIsNotConfigured(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.cfg.App.WebBaseURL = ""
	dana := h.recipient("Dana Whitfield")
	h.expectInvoiceAndSharer()
	h.expectUsers(dana)
	h.allow(dana)
	h.expectAudit()

	result, err := h.svc.Share(t.Context(), h.request([]pulid.ID{dana.ID}, "", ""), h.actor)
	require.NoError(t, err)

	assert.Equal(t, servicesports.InvoiceShareEmailNotConfigured, result.EmailStatus)
	assert.Zero(t, result.EmailsQueued)
	assert.Len(t, h.notifications, 1)
	h.emails.AssertNotCalled(t, "Send", mock.Anything, mock.Anything)
}

func TestShareReportsEmailFailuresWithoutFailingTheShare(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	dana := h.recipient("Dana Whitfield")
	h.expectInvoiceAndSharer()
	h.expectUsers(dana)
	h.allow(dana)
	h.expectAudit()
	h.expectEmails(errors.New("no email profile"))

	result, err := h.svc.Share(t.Context(), h.request([]pulid.ID{dana.ID}, "", ""), h.actor)
	require.NoError(t, err)

	assert.Equal(t, servicesports.InvoiceShareEmailFailed, result.EmailStatus)
	assert.Len(t, h.shares.upserted, 1)
	assert.Len(t, h.notifications, 1)
}

func TestShareRejectsARecipientWhoCannotViewInvoices(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	dana := h.recipient("Dana Whitfield")
	priya := h.recipient("Priya Nair")
	h.expectInvoiceAndSharer()
	h.expectUsers(dana, priya)
	h.allow(dana)
	h.expectPermission(priya, false)

	_, err := h.svc.Share(
		h.ctxWithSharerActivation(t),
		h.request([]pulid.ID{dana.ID, priya.ID}, "", ""),
		h.actor,
	)

	requireFieldError(t, err, "userIds[1]")
	assert.Empty(t, h.shares.upserted)
	assert.Empty(t, h.notifications)
}

func TestShareRejectsRecipientsOutsideTheOrganizationOrInactive(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	locked := h.recipient("Locked User")
	locked.IsLocked = true
	inactive := h.recipient("Inactive User")
	inactive.Status = domaintypes.StatusInactive
	system := h.recipient("System")
	system.Username = systemUsername
	stranger := pulid.MustNew("usr_")
	h.expectInvoiceAndSharer()
	h.expectUsers(locked, inactive, system)

	_, err := h.svc.Share(
		t.Context(),
		h.request([]pulid.ID{locked.ID, inactive.ID, system.ID, stranger}, "", ""),
		h.actor,
	)

	for _, field := range []string{"userIds[0]", "userIds[1]", "userIds[2]", "userIds[3]"} {
		requireFieldError(t, err, field)
	}
	h.permissions.AssertNotCalled(t, "Check", mock.Anything, mock.Anything)
	assert.Empty(t, h.shares.upserted)
}

func TestShareValidatesTheRequestBeforeReadingAnything(t *testing.T) {
	t.Parallel()

	tooMany := make([]pulid.ID, 0, invoice.MaxShareRecipients+1)
	for range invoice.MaxShareRecipients + 1 {
		tooMany = append(tooMany, pulid.MustNew("usr_"))
	}

	tests := []struct {
		name  string
		build func(h *harness) *servicesports.ShareInvoiceRequest
		field string
	}{
		{
			name:  "no recipients",
			build: func(h *harness) *servicesports.ShareInvoiceRequest { return h.request(nil, "", "") },
			field: "userIds",
		},
		{
			name:  "too many recipients",
			build: func(h *harness) *servicesports.ShareInvoiceRequest { return h.request(tooMany, "", "") },
			field: "userIds",
		},
		{
			name: "sharing with yourself",
			build: func(h *harness) *servicesports.ShareInvoiceRequest {
				return h.request([]pulid.ID{h.sharer.ID}, "", "")
			},
			field: "userIds[0]",
		},
		{
			name: "unknown tab",
			build: func(h *harness) *servicesports.ShareInvoiceRequest {
				return h.request(
					[]pulid.ID{pulid.MustNew("usr_")},
					"",
					invoice.ShareTab("payments"),
				)
			},
			field: "tab",
		},
		{
			name: "note too long",
			build: func(h *harness) *servicesports.ShareInvoiceRequest {
				return h.request(
					[]pulid.ID{pulid.MustNew("usr_")},
					strings.Repeat("é", invoice.MaxShareNoteLength+1),
					"",
				)
			},
			field: "note",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			h := newHarness(t)
			_, err := h.svc.Share(t.Context(), tt.build(h), h.actor)

			requireFieldError(t, err, tt.field)
			h.invoices.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
		})
	}
}

func TestShareAcceptsANoteAtTheLimitCountedInCharacters(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	dana := h.recipient("Dana Whitfield")
	h.cfg.App.WebBaseURL = ""
	h.expectInvoiceAndSharer()
	h.expectUsers(dana)
	h.allow(dana)
	h.expectAudit()

	_, err := h.svc.Share(
		t.Context(),
		h.request([]pulid.ID{dana.ID}, strings.Repeat("é", invoice.MaxShareNoteLength), ""),
		h.actor,
	)

	require.NoError(t, err)
}

func TestShareRequiresASignedInUser(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	apiKey := &servicesports.RequestActor{
		PrincipalType: servicesports.PrincipalTypeAPIKey,
		PrincipalID:   pulid.MustNew("ak_"),
		APIKeyID:      pulid.MustNew("ak_"),
	}

	_, err := h.svc.Share(t.Context(), h.request([]pulid.ID{pulid.MustNew("usr_")}, "", ""), apiKey)

	require.Error(t, err)
	h.invoices.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
}

func TestShareDoesNotNotifyWhenStoringFails(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.shares.upsertErr = errors.New("database unavailable")
	dana := h.recipient("Dana Whitfield")
	h.expectInvoiceAndSharer()
	h.expectUsers(dana)
	h.allow(dana)

	_, err := h.svc.Share(t.Context(), h.request([]pulid.ID{dana.ID}, "", ""), h.actor)

	require.Error(t, err)
	assert.Empty(t, h.notifications)
	h.emails.AssertNotCalled(t, "Send", mock.Anything, mock.Anything)
}

func TestListRequiresTheInvoiceInTheTenant(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.invoices.On("GetByID", mock.Anything, mock.Anything).
		Return(nil, errortypes.NewNotFoundError("Invoice not found"))

	_, err := h.svc.List(t.Context(), &repositories.ListInvoiceSharesRequest{
		TenantInfo: h.tenant,
		InvoiceID:  h.invoice.ID,
	})

	require.Error(t, err)
}

func TestEmailStatus(t *testing.T) {
	t.Parallel()

	assert.Equal(t, servicesports.InvoiceShareEmailNotConfigured, emailStatus(false, 0, 3))
	assert.Equal(t, servicesports.InvoiceShareEmailQueued, emailStatus(true, 3, 3))
	assert.Equal(t, servicesports.InvoiceShareEmailPartial, emailStatus(true, 1, 3))
	assert.Equal(t, servicesports.InvoiceShareEmailFailed, emailStatus(true, 0, 3))
}

func TestInvoiceSharePathOmitsTheDefaultTab(t *testing.T) {
	t.Parallel()

	assert.Equal(
		t,
		"/billing/invoices?item=inv_1",
		invoiceSharePath("inv_1", invoice.ShareTabOverview),
	)
	assert.Equal(t, "/billing/invoices?item=inv_1", invoiceSharePath("inv_1", ""))
	assert.Equal(t,
		"/billing/invoices?item=inv_1&tab=documents",
		invoiceSharePath("inv_1", invoice.ShareTabDocuments))
}

func (h *harness) expectInvoice() {
	h.invoices.On("GetByID", mock.Anything, repositories.GetInvoiceByIDRequest{
		ID:         h.invoice.ID,
		TenantInfo: h.tenant,
	}).Return(h.invoice, nil)
}

func TestListCandidatesOffersOnlyTeammatesWhoCanOpenTheLink(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	dana := h.recipient("Dana Whitfield")
	normal := h.recipient("Normal User")
	locked := h.recipient("Locked User")
	locked.IsLocked = true
	h.expectInvoice()
	h.users.On("SelectOptions", mock.Anything, mock.MatchedBy(func(req *pagination.SelectQueryRequest) bool {
		return req.TenantInfo == h.tenant && req.Query == "a" && req.Pagination.Offset == 0
	})).
		Return(&pagination.ListResult[*tenant.User]{
			Items: []*tenant.User{h.sharer, dana, normal, locked},
			Total: 4,
		}, nil)
	h.expectPermission(dana, true)
	h.expectPermission(normal, false)

	result, err := h.svc.ListCandidates(t.Context(), &servicesports.ListShareCandidatesRequest{
		TenantInfo: h.tenant,
		InvoiceID:  h.invoice.ID,
		Query:      "a",
		Pagination: pagination.Info{Limit: 20},
	})
	require.NoError(t, err)

	require.Len(t, result.Items, 1)
	assert.Equal(t, 1, result.Total)
	assert.Equal(t, dana.ID, result.Items[0].ID)
	assert.Equal(t, dana.EmailAddress, result.Items[0].EmailAddress)
}

func TestListCandidatesPagesThroughTheEligibleTeammates(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	users := make([]*tenant.User, 0, 5)
	for _, name := range []string{"Ana A", "Ben B", "Cara C", "Dev D", "Eli E"} {
		user := h.recipient(name)
		users = append(users, user)
		h.expectPermission(user, true)
	}
	h.expectInvoice()
	h.users.On("SelectOptions", mock.Anything, mock.Anything).
		Return(&pagination.ListResult[*tenant.User]{Items: users, Total: len(users)}, nil)

	result, err := h.svc.ListCandidates(t.Context(), &servicesports.ListShareCandidatesRequest{
		TenantInfo: h.tenant,
		InvoiceID:  h.invoice.ID,
		Pagination: pagination.Info{Limit: 2, Offset: 4},
	})
	require.NoError(t, err)

	assert.Equal(t, 5, result.Total)
	require.Len(t, result.Items, 1)
	assert.Equal(t, users[4].ID, result.Items[0].ID)
}

func TestGetCandidateHidesTeammatesWhoCannotReceiveAShare(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	dana := h.recipient("Dana Whitfield")
	normal := h.recipient("Normal User")
	h.expectInvoice()
	for _, user := range []*tenant.User{dana, normal, h.sharer} {
		h.users.On("GetByID", mock.Anything, repositories.GetUserByIDRequest{
			TenantInfo:   h.tenant,
			LookupUserID: user.ID,
		}).Return(user, nil)
	}
	h.expectPermission(dana, true)
	h.expectPermission(normal, false)

	candidate, err := h.svc.GetCandidate(t.Context(), &servicesports.GetShareCandidateRequest{
		TenantInfo: h.tenant,
		InvoiceID:  h.invoice.ID,
		UserID:     dana.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, dana.Name, candidate.Name)

	for _, hidden := range []*tenant.User{normal, h.sharer} {
		_, err = h.svc.GetCandidate(t.Context(), &servicesports.GetShareCandidateRequest{
			TenantInfo: h.tenant,
			InvoiceID:  h.invoice.ID,
			UserID:     hidden.ID,
		})
		assert.True(t, errortypes.IsNotFoundError(err), "expected not found for %s", hidden.Name)
	}
}
