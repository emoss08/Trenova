package invoiceservice

import (
	"errors"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ladingproject"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ladingports"
	"github.com/emoss08/trenova/internal/modules/identity/jobs/billingjobs/billingcontract"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/testutil"
	"github.com/emoss08/trenova/internal/testutil/mocks/customerportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/edipartnerportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/invoiceportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/kernelportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/ladingportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/lading"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
)

func TestBuildEDISendPlanBlockers(t *testing.T) {
	t.Parallel()

	posted := &invoice.Invoice{
		ID:            pulid.MustNew("inv_"),
		Status:        invoice.StatusPosted,
		EDISendStatus: invoice.EDISendStatusNotSent,
	}
	enabled := &customer.CustomerBillingProfile{
		EDIInvoiceEnabled:           true,
		AutoSendInvoiceOnGeneration: true,
	}
	partner := &edi.EDIPartner{
		ID:   pulid.MustNew("edip_"),
		Name: "AMD EDI",
		Kind: edi.PartnerKindExternal,
	}
	version := &ladingproject.LadingVersion{ID: pulid.MustNew("ldv_")}

	t.Run("profile switched off", func(t *testing.T) {
		t.Parallel()
		plan := buildEDISendPlan(posted, &customer.CustomerBillingProfile{}, nil)
		assert.False(t, plan.Enabled)
		assert.Equal(t, []string{ediBlockerProfileDisabled}, plan.Blockers)
		assert.Equal(t, invoice.EDISendStatusNotSent, plan.Status)
	})

	t.Run("no profile at all", func(t *testing.T) {
		t.Parallel()
		plan := buildEDISendPlan(
			&invoice.Invoice{ID: posted.ID, Status: invoice.StatusPosted},
			nil,
			nil,
		)
		assert.False(t, plan.Enabled)
		assert.Equal(t, invoice.EDISendStatusNotSent, plan.Status, "an empty status reads NotSent")
	})

	t.Run("voided", func(t *testing.T) {
		t.Parallel()
		plan := buildEDISendPlan(
			&invoice.Invoice{ID: posted.ID, Status: invoice.StatusVoided},
			enabled,
			nil,
		)
		assert.True(t, plan.Enabled)
		assert.Equal(t, []string{ediBlockerVoided, ediBlockerNoPartner}, plan.Blockers)
	})

	t.Run("draft", func(t *testing.T) {
		t.Parallel()
		plan := buildEDISendPlan(
			&invoice.Invoice{ID: posted.ID, Status: invoice.StatusDraft},
			enabled,
			&ediPartnerTarget{
				Partner:             partner,
				LadingVersion:       version,
				CommunicationMethod: edi.ConnectionMethodAS2,
			},
		)
		assert.Equal(t, []string{ediBlockerNotPosted}, plan.Blockers)
	})

	t.Run("no partner", func(t *testing.T) {
		t.Parallel()
		plan := buildEDISendPlan(posted, enabled, nil)
		assert.Equal(t, []string{ediBlockerNoPartner}, plan.Blockers)
		assert.True(t, plan.AutoSend)
	})

	t.Run("partner without a version or transport", func(t *testing.T) {
		t.Parallel()
		plan := buildEDISendPlan(posted, enabled, &ediPartnerTarget{
			Partner:  partner,
			Blockers: []string{ediBlockerNoLadingVersion, ediBlockerNoCommunication},
		})
		assert.Equal(t, partner.ID, plan.PartnerID)
		assert.Equal(t, "AMD EDI", plan.PartnerName)
		assert.True(t, plan.LadingVersionID.IsNil())
		assert.Equal(
			t,
			[]string{ediBlockerNoLadingVersion, ediBlockerNoCommunication},
			plan.Blockers,
		)
	})

	t.Run("ready external partner", func(t *testing.T) {
		t.Parallel()
		plan := buildEDISendPlan(posted, enabled, &ediPartnerTarget{
			Partner: partner, LadingVersion: version, CommunicationMethod: edi.ConnectionMethodSFTP,
		})
		assert.Empty(t, plan.Blockers)
		assert.Equal(t, version.ID, plan.LadingVersionID)
		assert.Equal(t, "SFTP", plan.CommunicationMethod)
	})
}

func TestResolvePartnerTargetInternalPartnerNeedsNoTransport(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	partner := &edi.EDIPartner{ID: pulid.MustNew("edip_"), Kind: edi.PartnerKindInternal}

	projects := ladingportsmock.NewMockLadingProjectRepository(t)
	commProfiles := edipartnerportsmock.NewMockEDICommunicationProfileRepository(t)

	svc := &Service{
		l:                           zap.NewNop(),
		ediLadingProjects:           projects,
		ediCommunicationProfileRepo: commProfiles,
	}

	target, err := svc.resolvePartnerTarget(t.Context(), tenantInfo, partner)

	require.NoError(t, err)
	assert.Equal(t, edi.ConnectionMethodInternal, target.CommunicationMethod)
	assert.Empty(t, target.Blockers)
	assert.Nil(t, target.LadingVersion, "internal partners exchange no X12")
	commProfiles.AssertNotCalled(t, "GetActiveProfileByPartner", mock.Anything, mock.Anything)
	projects.AssertNotCalled(t, "GetRunnableVersions", mock.Anything, mock.Anything)
}

func TestInvoiceLadingVersionNeedsAnOutbound210Map(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	partner := &edi.EDIPartner{ID: pulid.MustNew("edip_"), Kind: edi.PartnerKindExternal}
	withMaps := func(maps ...lading.Map) *ladingports.RunnableLadingVersions {
		return &ladingports.RunnableLadingVersions{Active: &ladingproject.LadingVersion{
			ID:       pulid.MustNew("ldv_"),
			Manifest: lading.Check{Maps: maps},
		}}
	}

	tests := []struct {
		name     string
		runnable *ladingports.RunnableLadingVersions
		err      error
		found    bool
	}{
		{
			name: "an outbound 210 map",
			runnable: withMaps(
				lading.Map{Name: "invoiceOut", Set: "210", Direction: lading.DirectionOutbound},
			),
			found: true,
		},
		{
			name: "only an inbound 210 map",
			runnable: withMaps(
				lading.Map{Name: "invoiceIn", Set: "210", Direction: lading.DirectionInbound},
			),
		},
		{
			name:     "no active version",
			runnable: &ladingports.RunnableLadingVersions{},
		},
		{
			name: "no project",
			err:  errortypes.NewNotFoundError("LadingProject"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			projects := ladingportsmock.NewMockLadingProjectRepository(t)
			projects.EXPECT().
				GetRunnableVersions(mock.Anything, ladingports.GetRunnableLadingVersionRequest{
					TenantInfo: tenantInfo,
					PartnerID:  partner.ID,
				}).
				Return(tt.runnable, tt.err).
				Once()
			svc := &Service{l: zap.NewNop(), ediLadingProjects: projects}

			version, found, err := svc.invoiceLadingVersion(t.Context(), tenantInfo, partner)

			require.NoError(t, err)
			assert.Equal(t, tt.found, found)
			if tt.found {
				require.NotNil(t, version)
				assert.Equal(t, tt.runnable.Active.ID, version.ID)
				return
			}
			assert.Nil(t, version)
		})
	}
}

func TestResolvePartnerTargetReportsMissingProfiles(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	partner := &edi.EDIPartner{ID: pulid.MustNew("edip_"), Kind: edi.PartnerKindExternal}

	projects := ladingportsmock.NewMockLadingProjectRepository(t)
	projects.EXPECT().
		GetRunnableVersions(mock.Anything, mock.Anything).
		Return(nil, errortypes.NewNotFoundError("LadingProject")).
		Once()
	commProfiles := edipartnerportsmock.NewMockEDICommunicationProfileRepository(t)
	commProfiles.EXPECT().
		GetActiveProfileByPartner(mock.Anything, edipartnerports.GetActiveEDICommunicationProfileByPartnerRequest{
			PartnerID:  partner.ID,
			TenantInfo: tenantInfo,
		}).
		Return(nil, errortypes.NewNotFoundError("no transport")).
		Once()

	svc := &Service{
		l:                           zap.NewNop(),
		ediLadingProjects:           projects,
		ediCommunicationProfileRepo: commProfiles,
	}

	target, err := svc.resolvePartnerTarget(t.Context(), tenantInfo, partner)

	require.NoError(t, err)
	assert.Equal(
		t,
		[]string{ediBlockerNoLadingVersion, ediBlockerNoCommunication},
		target.Blockers,
	)
}

func TestResolveEDISendPlansWithoutEDIWiringNeverTouchesCustomers(t *testing.T) {
	t.Parallel()

	customerRepo := customerportsmock.NewMockCustomerRepository(t)
	svc := &Service{l: zap.NewNop(), customerRepo: customerRepo}
	inv := &invoice.Invoice{
		ID:            pulid.MustNew("inv_"),
		CustomerID:    pulid.MustNew("cus_"),
		EDISendStatus: invoice.EDISendStatusFailed,
	}

	plans, err := svc.ResolveEDISendPlans(
		t.Context(),
		&invoiceports.ResolveInvoiceEDISendPlansRequest{
			TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_")},
			Invoices:   []*invoice.Invoice{inv, nil},
		},
	)

	require.NoError(t, err)
	require.Len(t, plans, 1)
	plan := plans[inv.ID]
	require.NotNil(t, plan)
	assert.False(t, plan.Enabled)
	assert.Equal(t, invoice.EDISendStatusFailed, plan.Status)
	assert.Equal(t, []string{ediBlockerNotWired}, plan.Blockers)
	customerRepo.AssertNotCalled(t, "GetByIDs", mock.Anything, mock.Anything)
}

// ediWiredFixture is an invoice whose customer has EDI invoicing on, an active
// external partner, an active Lading version that writes a 210 and an SFTP channel: nothing blocks a send.
type ediWiredFixture struct {
	tenantInfo pagination.TenantInfo
	actor      *permissionports.RequestActor
	inv        *invoice.Invoice
	repo       *invoiceportsmock.MockInvoiceRepository
	starter    *kernelportsmock.MockWorkflowStarter
	svc        *Service
}

func newEDIWiredFixture(t *testing.T, status invoice.EDISendStatus) *ediWiredFixture {
	t.Helper()
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	customerID := pulid.MustNew("cus_")
	partnerID := pulid.MustNew("edip_")
	tenantInfo := pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID}

	customerRepo := customerportsmock.NewMockCustomerRepository(t)
	customerRepo.EXPECT().
		GetByIDs(mock.Anything, customerports.GetCustomersByIDsRequest{
			TenantInfo:            tenantInfo,
			CustomerIDs:           []pulid.ID{customerID},
			CustomerFilterOptions: customerports.CustomerFilterOptions{IncludeBillingProfile: true},
		}).
		Return([]*customer.Customer{{
			ID: customerID,
			BillingProfile: &customer.CustomerBillingProfile{
				EDIInvoiceEnabled:           true,
				AutoSendInvoiceOnGeneration: true,
			},
		}}, nil).
		Maybe()

	partners := edipartnerportsmock.NewMockEDIPartnerRepository(t)
	partners.EXPECT().
		ListOutboundPartnersByCustomerIDs(mock.Anything, edipartnerports.ListEDIPartnersByCustomerIDsRequest{
			CustomerIDs: []pulid.ID{customerID},
			TenantInfo:  tenantInfo,
		}).
		Return([]*edi.EDIPartner{{ID: partnerID, CustomerID: customerID, Name: "AMD EDI", Kind: edi.PartnerKindExternal}}, nil).
		Maybe()

	projects := ladingportsmock.NewMockLadingProjectRepository(t)
	projects.EXPECT().
		GetRunnableVersions(mock.Anything, mock.Anything).
		Return(&ladingports.RunnableLadingVersions{Active: &ladingproject.LadingVersion{
			ID: pulid.MustNew("ldv_"),
			Manifest: lading.Check{Maps: []lading.Map{
				{Name: "invoiceOut", Set: "210", Direction: lading.DirectionOutbound},
			}},
		}}, nil).
		Maybe()
	commProfiles := edipartnerportsmock.NewMockEDICommunicationProfileRepository(t)
	commProfiles.EXPECT().
		GetActiveProfileByPartner(mock.Anything, mock.Anything).
		Return(&edi.EDICommunicationProfile{Method: edi.ConnectionMethodSFTP}, nil).
		Maybe()

	f := &ediWiredFixture{
		tenantInfo: tenantInfo,
		actor:      testutil.NewSessionActor(userID, orgID, buID),
		inv: &invoice.Invoice{
			ID:             pulid.MustNew("inv_"),
			OrganizationID: orgID,
			BusinessUnitID: buID,
			CustomerID:     customerID,
			Number:         "INV-3001",
			Status:         invoice.StatusPosted,
			EDISendStatus:  status,
		},
		repo:    invoiceportsmock.NewMockInvoiceRepository(t),
		starter: kernelportsmock.NewMockWorkflowStarter(t),
	}
	f.svc = &Service{
		l:                           zap.NewNop(),
		repo:                        f.repo,
		customerRepo:                customerRepo,
		workflowStarter:             f.starter,
		ediPartnerRepo:              partners,
		ediLadingProjects:           projects,
		ediCommunicationProfileRepo: commProfiles,
	}

	return f
}

func (f *ediWiredFixture) expectLoad() {
	f.repo.EXPECT().
		GetByID(mock.Anything, invoiceports.GetInvoiceByIDRequest{ID: f.inv.ID, TenantInfo: f.tenantInfo}).
		Return(f.inv, nil).
		Once()
}

func (f *ediWiredFixture) expectWorkflowStart(t *testing.T, force bool) {
	t.Helper()
	f.starter.EXPECT().Enabled().Return(true).Once()
	f.starter.EXPECT().
		StartWorkflow(
			mock.Anything,
			mock.MatchedBy(func(options client.StartWorkflowOptions) bool {
				return strings.HasPrefix(options.ID, "invoice-edi-send-"+f.inv.ID.String())
			}),
			billingcontract.SendInvoiceEDIWorkflowName,
			mock.MatchedBy(func(args []any) bool {
				require.Len(t, args, 1)
				payload, ok := args[0].(*billingcontract.SendInvoiceEDIPayload)
				return ok && payload.InvoiceID == f.inv.ID && payload.Force == force &&
					payload.OrganizationID == f.tenantInfo.OrgID
			}),
		).
		Return(fakeWorkflowRun{id: "wf-edi", runID: "run-edi"}, nil).
		Once()
	f.repo.EXPECT().
		UpdateEDISendStatus(mock.Anything, invoiceports.UpdateInvoiceEDISendStatusRequest{
			TenantInfo: f.tenantInfo,
			InvoiceID:  f.inv.ID,
			Status:     invoice.EDISendStatusQueued,
		}).
		Return(nil).
		Once()
}

func TestSendEDIRefusesADraft(t *testing.T) {
	t.Parallel()

	f := newEDIWiredFixture(t, invoice.EDISendStatusNotSent)
	f.inv.Status = invoice.StatusDraft
	f.expectLoad()

	_, err := f.svc.SendEDI(
		t.Context(),
		&invoiceports.SendInvoiceEDIRequest{InvoiceID: f.inv.ID, TenantInfo: f.tenantInfo},
		f.actor,
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Only a posted invoice")
}

func TestSendEDIRefusesAnInFlightSendUnlessForced(t *testing.T) {
	t.Parallel()

	for _, status := range []invoice.EDISendStatus{invoice.EDISendStatusQueued, invoice.EDISendStatusSending} {
		t.Run(string(status), func(t *testing.T) {
			t.Parallel()
			f := newEDIWiredFixture(t, status)
			f.expectLoad()

			_, err := f.svc.SendEDI(
				t.Context(),
				&invoiceports.SendInvoiceEDIRequest{InvoiceID: f.inv.ID, TenantInfo: f.tenantInfo},
				f.actor,
			)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "already in progress")
		})
	}
}

func TestSendEDIRefusesASentInvoiceUnlessForced(t *testing.T) {
	t.Parallel()

	f := newEDIWiredFixture(t, invoice.EDISendStatusSent)
	f.expectLoad()

	_, err := f.svc.SendEDI(
		t.Context(),
		&invoiceports.SendInvoiceEDIRequest{InvoiceID: f.inv.ID, TenantInfo: f.tenantInfo},
		f.actor,
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "resend it with force")
}

func TestSendEDIForceQueuesTheWorkflow(t *testing.T) {
	t.Parallel()

	f := newEDIWiredFixture(t, invoice.EDISendStatusSent)
	f.expectLoad()
	f.expectWorkflowStart(t, true)

	result, err := f.svc.SendEDI(
		t.Context(),
		&invoiceports.SendInvoiceEDIRequest{
			InvoiceID:  f.inv.ID,
			TenantInfo: f.tenantInfo,
			Force:      true,
		},
		f.actor,
	)

	require.NoError(t, err)
	assert.Equal(t, invoice.EDISendStatusQueued, result.Status)
	assert.Equal(t, "wf-edi", result.WorkflowID)
	assert.Equal(t, "run-edi", result.WorkflowRunID)
	assert.Equal(
		t,
		invoice.EDISendStatusQueued,
		f.inv.EDISendStatus,
		"the in-memory invoice follows the write",
	)
}

func TestSendEDIRefusesWhenTheChannelIsBlocked(t *testing.T) {
	t.Parallel()

	f := newEDIWiredFixture(t, invoice.EDISendStatusNotSent)
	f.svc.ediPartnerRepo = nil
	f.expectLoad()

	_, err := f.svc.SendEDI(
		t.Context(),
		&invoiceports.SendInvoiceEDIRequest{InvoiceID: f.inv.ID, TenantInfo: f.tenantInfo},
		f.actor,
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), ediBlockerNotWired)
}

func TestEnqueueEDIAfterPostQueuesWhenAutoSendIsOn(t *testing.T) {
	t.Parallel()

	f := newEDIWiredFixture(t, invoice.EDISendStatusNotSent)
	f.expectWorkflowStart(t, false)

	f.svc.enqueueEDIAfterPost(t.Context(), f.inv, f.tenantInfo, f.actor)

	assert.Equal(t, invoice.EDISendStatusQueued, f.inv.EDISendStatus)
}

func TestEnqueueEDIAfterPostRecordsTheFirstBlocker(t *testing.T) {
	t.Parallel()

	f := newEDIWiredFixture(t, invoice.EDISendStatusNotSent)
	f.svc.ediLadingProjects = nil
	f.repo.EXPECT().
		UpdateEDISendStatus(mock.Anything, invoiceports.UpdateInvoiceEDISendStatusRequest{
			TenantInfo: f.tenantInfo,
			InvoiceID:  f.inv.ID,
			Status:     invoice.EDISendStatusNotConfigured,
			Error:      ediBlockerNoLadingVersion,
		}).
		Return(nil).
		Once()

	f.svc.enqueueEDIAfterPost(t.Context(), f.inv, f.tenantInfo, f.actor)

	assert.Equal(t, invoice.EDISendStatusNotConfigured, f.inv.EDISendStatus)
	assert.Equal(t, ediBlockerNoLadingVersion, f.inv.LastEDIError)
	f.starter.AssertNotCalled(
		t,
		"StartWorkflow",
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
	)
}

func TestEnqueueEDIAfterPostMarksFailedWhenTheWorkflowCannotStart(t *testing.T) {
	t.Parallel()

	f := newEDIWiredFixture(t, invoice.EDISendStatusNotSent)
	f.starter.EXPECT().Enabled().Return(true).Once()
	f.starter.EXPECT().
		StartWorkflow(mock.Anything, mock.Anything, billingcontract.SendInvoiceEDIWorkflowName, mock.Anything).
		Return(nil, errors.New("temporal down")).
		Once()
	f.repo.EXPECT().
		UpdateEDISendStatus(mock.Anything, mock.MatchedBy(func(req invoiceports.UpdateInvoiceEDISendStatusRequest) bool {
			return req.InvoiceID == f.inv.ID && req.Status == invoice.EDISendStatusFailed &&
				strings.Contains(req.Error, "Failed to start invoice EDI send")
		})).
		Return(nil).
		Once()

	f.svc.enqueueEDIAfterPost(t.Context(), f.inv, f.tenantInfo, f.actor)

	assert.Equal(t, invoice.EDISendStatusFailed, f.inv.EDISendStatus)
}

func TestEnqueueEDIAfterPostIsSilentWhenAutoSendIsOff(t *testing.T) {
	t.Parallel()

	f := newEDIWiredFixture(t, invoice.EDISendStatusNotSent)
	customerRepo := customerportsmock.NewMockCustomerRepository(t)
	customerRepo.EXPECT().
		GetByIDs(mock.Anything, mock.Anything).
		Return([]*customer.Customer{{
			ID:             f.inv.CustomerID,
			BillingProfile: &customer.CustomerBillingProfile{EDIInvoiceEnabled: true},
		}}, nil).
		Once()
	f.svc.customerRepo = customerRepo

	f.svc.enqueueEDIAfterPost(t.Context(), f.inv, f.tenantInfo, f.actor)

	assert.Equal(t, invoice.EDISendStatusNotSent, f.inv.EDISendStatus)
	f.starter.AssertNotCalled(
		t,
		"StartWorkflow",
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
	)
}
