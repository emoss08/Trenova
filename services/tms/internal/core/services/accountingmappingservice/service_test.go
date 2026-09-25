package accountingmappingservice

import (
	"errors"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type harness struct {
	svc         *Service
	tenant      pagination.TenantInfo
	conn        *accountingsync.AccountingConnection
	connections *fakeConnections
	references  *fakeReferences
	mappings    *fakeMappings
	connector   *fakeConnector
	connService *fakeConnectionService
	audit       *fakeAudit
	completion  *fakeCompletion
	refresher   *fakeRefresher
	customerID  pulid.ID
	carrierID   pulid.ID
	chargeID    pulid.ID
}

func account(externalID, name, accountType string) *accountingsync.AccountingReferenceObject {
	return &accountingsync.AccountingReferenceObject{
		Kind:        accountingsync.ReferenceKindAccount,
		ExternalID:  externalID,
		Name:        name,
		AccountType: accountType,
		Active:      true,
	}
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	h := &harness{
		tenant: pagination.TenantInfo{
			OrgID:  pulid.MustNew("org_"),
			BuID:   pulid.MustNew("bu_"),
			UserID: pulid.MustNew("usr_"),
		},
		customerID: pulid.MustNew("cus_"),
		carrierID:  pulid.MustNew("car_"),
		chargeID:   pulid.MustNew("acc_"),
	}
	h.conn = &accountingsync.AccountingConnection{
		ID:              pulid.MustNew("acctc_"),
		OrganizationID:  h.tenant.OrgID,
		BusinessUnitID:  h.tenant.BuID,
		IntegrationType: integration.TypeQuickBooksOnline,
		Status:          accountingsync.ConnectionStatusConnected,
		ExternalRealmID: "9130",
		SetupStep:       accountingsync.SetupStepMappings,
	}
	h.connections = newFakeConnections(h.conn)
	h.references = newFakeReferences(
		account("1", "Accounts Receivable (A/R)", "Accounts Receivable"),
		account("2", "Freight Income", "Income"),
		account("3", "Checking", "Bank"),
		account("4", "Office Supplies", "Expense"),
		party(accountingsync.ReferenceKindCustomer, "50", "Acme Logistics, Inc.", ""),
		party(accountingsync.ReferenceKindVendor, "70", "Roadrunner Freight LLC", ""),
		&accountingsync.AccountingReferenceObject{
			Kind:       accountingsync.ReferenceKindItem,
			ExternalID: "90",
			Name:       "Freight",
			ItemType:   "Service",
			Active:     true,
		},
	)
	for _, ref := range h.references.rows {
		ref.ConnectionID = h.conn.ID
	}
	h.mappings = &fakeMappings{}
	h.connector = &fakeConnector{}
	h.connService = &fakeConnectionService{
		session: &services.AccountingSession{
			Connection:  h.conn,
			AccessToken: "access",
			Connector:   h.connector,
		},
	}
	h.audit = &fakeAudit{}
	h.completion = &fakeCompletion{}
	h.refresher = &fakeRefresher{}

	controls := mocks.NewMockAccountingControlRepository(t)
	controls.EXPECT().GetByOrgID(mock.Anything, h.tenant.OrgID).
		Return(nil, errortypes.NewNotFoundError("not found")).Maybe()

	charges := mocks.NewMockAccessorialChargeRepository(t)
	charges.EXPECT().List(mock.Anything, mock.Anything).
		Return(&pagination.ListResult[*accessorialcharge.AccessorialCharge]{
			Items: []*accessorialcharge.AccessorialCharge{{
				ID:          h.chargeID,
				Code:        "DET",
				Description: "Detention",
			}},
			Total: 1,
		}, nil).Maybe()
	charges.EXPECT().GetByID(mock.Anything, mock.Anything).
		Return(&accessorialcharge.AccessorialCharge{
			ID:          h.chargeID,
			Code:        "DET",
			Description: "Detention",
		}, nil).Maybe()

	cus := &customer.Customer{ID: h.customerID, Name: "Acme Logistics", Code: "ACME"}
	customers := mocks.NewMockCustomerRepository(t)
	customers.EXPECT().List(mock.Anything, mock.Anything).
		Return(&pagination.ListResult[*customer.Customer]{Items: []*customer.Customer{cus}, Total: 1}, nil).
		Maybe()
	customers.EXPECT().GetByIDs(mock.Anything, mock.Anything).
		Return([]*customer.Customer{cus}, nil).Maybe()

	carr := &carrier.Carrier{ID: h.carrierID, Name: "Roadrunner Freight", SCAC: "RRFT"}
	carriers := mocks.NewMockCarrierRepository(t)
	carriers.EXPECT().List(mock.Anything, mock.Anything).
		Return(&pagination.ListResult[*carrier.Carrier]{Items: []*carrier.Carrier{carr}, Total: 1}, nil).
		Maybe()
	carriers.EXPECT().GetByIDs(mock.Anything, mock.Anything).
		Return([]*carrier.Carrier{carr}, nil).Maybe()

	h.svc = New(Params{
		Logger:             zap.NewNop(),
		DB:                 dbtest.NopConnection{},
		Connections:        h.connections,
		References:         h.references,
		Mappings:           h.mappings,
		ConnectionService:  h.connService,
		AccountingControls: controls,
		GLAccounts:         mocks.NewMockGLAccountRepository(t),
		Customers:          customers,
		Carriers:           carriers,
		Accessorials:       charges,
		AuditService:       h.audit,
		Completion:         h.completion,
		Refresher:          h.refresher,
	})
	return h
}

func (h *harness) rescore(t *testing.T) *services.AccountingRescoreResult {
	t.Helper()
	result, err := h.svc.Rescore(t.Context(), h.tenant, h.conn.ID)
	require.NoError(t, err)
	return result
}

func (h *harness) keyed(
	t *testing.T,
	targetType accountingsync.MappingTargetType,
	key string,
) *accountingsync.AccountingMapping {
	t.Helper()
	row := h.mappings.byTarget(targetType, pulid.Nil, key)
	require.NotNil(t, row, "%s %s", targetType, key)
	return h.mappings.get(row.ID)
}

func (h *harness) object(
	t *testing.T,
	targetType accountingsync.MappingTargetType,
	id pulid.ID,
) *accountingsync.AccountingMapping {
	t.Helper()
	row := h.mappings.byTarget(targetType, id, "")
	require.NotNil(t, row, "%s %s", targetType, id)
	return h.mappings.get(row.ID)
}

func TestRescoreCreatesEveryTargetAndProposesMatches(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	result := h.rescore(t)

	keyed := 0
	for _, targetType := range accountingsync.AllMappingTargetTypes() {
		keyed += len(targetType.Keys())
	}
	assert.Equal(t, keyed+3, result.Targets)
	assert.EqualValues(t, result.Targets, result.Created)

	ar := h.keyed(t, accountingsync.TargetAccountRole, accountingsync.AccountRoleAR)
	assert.Equal(t, accountingsync.MappingStateProposed, ar.State)
	assert.Equal(t, "1", ar.ExternalID)

	revenue := h.keyed(t, accountingsync.TargetAccountRole, accountingsync.AccountRoleRevenue)
	assert.Equal(t, "2", revenue.ExternalID)

	deposit := h.keyed(t, accountingsync.TargetAccountRole, accountingsync.AccountRoleDeposit)
	assert.Equal(t, "3", deposit.ExternalID)

	writeOff := h.keyed(t, accountingsync.TargetAccountRole, accountingsync.AccountRoleWriteOff)
	assert.NotEqual(t, "4", writeOff.ExternalID, "an unrelated expense account is never proposed on its name")

	cus := h.object(t, accountingsync.TargetCustomer, h.customerID)
	assert.Equal(t, accountingsync.MappingStateProposed, cus.State)
	assert.Equal(t, "50", cus.ExternalID)

	vendor := h.object(t, accountingsync.TargetCarrier, h.carrierID)
	assert.Equal(t, "70", vendor.ExternalID)

	freight := h.keyed(t, accountingsync.TargetLineType, "Freight")
	assert.Equal(t, "90", freight.ExternalID)
}

func TestRescoreIsIdempotentAndNeverTouchesConfirmedRows(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.rescore(t)

	ar := h.keyed(t, accountingsync.TargetAccountRole, accountingsync.AccountRoleAR)
	confirmed, err := h.svc.Confirm(t.Context(), &services.ConfirmAccountingMappingsRequest{
		TenantInfo: h.tenant,
		UserID:     h.tenant.UserID,
		IDs:        []pulid.ID{ar.ID},
	})
	require.NoError(t, err)
	require.Len(t, confirmed, 1)

	h.references.rows = append(h.references.rows, account("8", "Accounts Receivable", "Accounts Receivable"))

	second := h.rescore(t)
	assert.Zero(t, second.Created)

	after := h.keyed(t, accountingsync.TargetAccountRole, accountingsync.AccountRoleAR)
	assert.Equal(t, accountingsync.MappingStateConfirmed, after.State)
	assert.Equal(t, "1", after.ExternalID)

	third := h.rescore(t)
	assert.Zero(t, third.Updated, "a second pass over unchanged data writes nothing")
}

func TestRejectedCandidateIsNotProposedAgain(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.rescore(t)

	cus := h.object(t, accountingsync.TargetCustomer, h.customerID)
	rejected, err := h.svc.Reject(t.Context(), &services.AccountingMappingActionRequest{
		TenantInfo: h.tenant,
		UserID:     h.tenant.UserID,
		ID:         cus.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, accountingsync.MappingStateUnmatched, rejected.State)
	assert.Contains(t, rejected.Signals.RejectedExternalIDs, "50")

	h.rescore(t)
	after := h.object(t, accountingsync.TargetCustomer, h.customerID)
	assert.Equal(t, accountingsync.MappingStateUnmatched, after.State)
	assert.Empty(t, after.ExternalID)
}

func TestConfirmRejectsRowsWithoutAProposal(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.rescore(t)

	ap := h.keyed(t, accountingsync.TargetAccountRole, accountingsync.AccountRoleAP)
	require.Equal(t, accountingsync.MappingStateUnmatched, ap.State)

	_, err := h.svc.Confirm(t.Context(), &services.ConfirmAccountingMappingsRequest{
		TenantInfo: h.tenant,
		IDs:        []pulid.ID{ap.ID},
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
}

func TestConfirmRefusesAReferenceThatWentInactive(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.rescore(t)

	cus := h.object(t, accountingsync.TargetCustomer, h.customerID)
	for _, ref := range h.references.rows {
		if ref.ExternalID == "50" {
			ref.Active = false
		}
	}

	_, err := h.svc.Confirm(t.Context(), &services.ConfirmAccountingMappingsRequest{
		TenantInfo: h.tenant,
		IDs:        []pulid.ID{cus.ID},
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsError(err))
	assert.Equal(t, accountingsync.MappingStateProposed, h.object(t, accountingsync.TargetCustomer, h.customerID).State)
}

func TestConfirmLimitsTheBatch(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	ids := make([]pulid.ID, maxConfirmBatch+1)
	for idx := range ids {
		ids[idx] = pulid.MustNew("acctm_")
	}
	_, err := h.svc.Confirm(t.Context(), &services.ConfirmAccountingMappingsRequest{
		TenantInfo: h.tenant,
		IDs:        ids,
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsError(err))
}

func TestConfirmRecordsTheActorAndAudits(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.rescore(t)

	ar := h.keyed(t, accountingsync.TargetAccountRole, accountingsync.AccountRoleAR)
	_, err := h.svc.Confirm(t.Context(), &services.ConfirmAccountingMappingsRequest{
		TenantInfo: h.tenant,
		UserID:     h.tenant.UserID,
		IDs:        []pulid.ID{ar.ID},
		Source:     accountingsync.MappingSourceAgent,
	})
	require.NoError(t, err)

	after := h.keyed(t, accountingsync.TargetAccountRole, accountingsync.AccountRoleAR)
	assert.Equal(t, h.tenant.UserID, after.ConfirmedByID)
	assert.Equal(t, accountingsync.MappingSourceAgent, after.Source)
	require.Len(t, h.audit.entries, 1)
	assert.Equal(t, ar.ID.String(), h.audit.entries[0].ResourceID)
}

func TestSetRefusesAnIneligibleAccount(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.rescore(t)

	_, err := h.svc.Set(t.Context(), &services.SetAccountingMappingRequest{
		TenantInfo:      h.tenant,
		IntegrationType: integration.TypeQuickBooksOnline,
		TargetType:      accountingsync.TargetAccountRole,
		TrenovaKey:      accountingsync.AccountRoleAR,
		ExternalID:      "4",
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsError(err))

	_, err = h.svc.Set(t.Context(), &services.SetAccountingMappingRequest{
		TenantInfo:      h.tenant,
		IntegrationType: integration.TypeQuickBooksOnline,
		TargetType:      accountingsync.TargetAccountRole,
		TrenovaKey:      accountingsync.AccountRoleAR,
		ExternalID:      "missing",
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsError(err))
}

func TestSetConfirmsAManualChoice(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.rescore(t)

	updated, err := h.svc.Set(t.Context(), &services.SetAccountingMappingRequest{
		TenantInfo:      h.tenant,
		UserID:          h.tenant.UserID,
		IntegrationType: integration.TypeQuickBooksOnline,
		TargetType:      accountingsync.TargetAccountRole,
		TrenovaKey:      accountingsync.AccountRoleWriteOff,
		ExternalID:      "4",
		Source:          accountingsync.MappingSourceModel,
		Reason:          "The bookkeeper posts short pays here",
	})
	require.NoError(t, err)
	assert.Equal(t, accountingsync.MappingStateConfirmed, updated.State)
	assert.Equal(t, accountingsync.MappingSourceManual, updated.Source, "a caller cannot claim a model source")
	assert.Equal(t, "Office Supplies", updated.ExternalName)
}

func TestClearReturnsARowToUnmatched(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.rescore(t)

	ap := h.keyed(t, accountingsync.TargetAccountRole, accountingsync.AccountRoleAP)
	_, err := h.svc.Clear(t.Context(), &services.AccountingMappingActionRequest{
		TenantInfo: h.tenant,
		ID:         ap.ID,
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))

	ar := h.keyed(t, accountingsync.TargetAccountRole, accountingsync.AccountRoleAR)
	cleared, err := h.svc.Clear(t.Context(), &services.AccountingMappingActionRequest{
		TenantInfo: h.tenant,
		ID:         ar.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, accountingsync.MappingStateUnmatched, cleared.State)
	assert.Empty(t, cleared.ExternalID)
}

func confirmRequired(t *testing.T, h *harness) {
	t.Helper()
	ids := make([]pulid.ID, 0, 4)
	for _, role := range []string{
		accountingsync.AccountRoleAR,
		accountingsync.AccountRoleRevenue,
		accountingsync.AccountRoleDeposit,
	} {
		ids = append(ids, h.keyed(t, accountingsync.TargetAccountRole, role).ID)
	}
	ids = append(ids, h.keyed(t, accountingsync.TargetLineType, "Freight").ID)
	_, err := h.svc.Confirm(t.Context(), &services.ConfirmAccountingMappingsRequest{
		TenantInfo: h.tenant,
		UserID:     h.tenant.UserID,
		IDs:        ids,
	})
	require.NoError(t, err)
}

func TestCompleteSetupWaitsForTheRequiredMappings(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.rescore(t)
	req := &services.AccountingSetupRequest{
		TenantInfo:      h.tenant,
		UserID:          h.tenant.UserID,
		IntegrationType: integration.TypeQuickBooksOnline,
	}

	summary, err := h.svc.Summary(t.Context(), h.tenant, integration.TypeQuickBooksOnline)
	require.NoError(t, err)
	assert.Equal(t, 4, summary.RequiredTotal)
	assert.Zero(t, summary.RequiredConfirmed)
	assert.False(t, summary.CanCompleteSetup)

	_, err = h.svc.CompleteSetup(t.Context(), req)
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))

	confirmRequired(t, h)

	conn, err := h.svc.CompleteSetup(t.Context(), req)
	require.NoError(t, err)
	assert.Equal(t, accountingsync.SetupStepComplete, conn.SetupStep)
	assert.Equal(t, accountingsync.SetupStepComplete, h.connections.rows[h.conn.ID].SetupStep)

	again, err := h.svc.CompleteSetup(t.Context(), req)
	require.NoError(t, err)
	assert.Equal(t, conn.Version, again.Version, "finishing twice writes nothing")
}

func TestSummaryWithoutAConnection(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	delete(h.connections.rows, h.conn.ID)

	summary, err := h.svc.Summary(t.Context(), h.tenant, integration.TypeQuickBooksOnline)
	require.NoError(t, err)
	assert.Nil(t, summary.Connection)
	assert.False(t, summary.CanCompleteSetup)

	_, err = h.svc.Summary(t.Context(), h.tenant, integration.Type("Samsara"))
	require.Error(t, err)
	assert.True(t, errortypes.IsError(err))
}

func TestRequestRefreshStartsTheWorkflow(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	_, err := h.svc.RequestRefresh(t.Context(), &services.AccountingSetupRequest{
		TenantInfo:      h.tenant,
		IntegrationType: integration.TypeQuickBooksOnline,
	})
	require.NoError(t, err)
	assert.Equal(t, []pulid.ID{h.conn.ID}, h.refresher.requested)

	h.connections.rows[h.conn.ID].Status = accountingsync.ConnectionStatusDisconnected
	_, err = h.svc.RequestRefresh(t.Context(), &services.AccountingSetupRequest{
		TenantInfo:      h.tenant,
		IntegrationType: integration.TypeQuickBooksOnline,
	})
	require.Error(t, err)
	assert.Len(t, h.refresher.requested, 1)
}

func TestPullReferencePagesAndMarksUnseenRecordsRemoved(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.connector.pages = map[accountingsync.ReferenceKind][][]*accountingsync.AccountingReferenceObject{
		accountingsync.ReferenceKindAccount: {
			{
				account("1", "Accounts Receivable (A/R)", "Accounts Receivable"),
				account("2", "Freight Income", "Income"),
			},
			{account("3", "Checking", "Bank")},
		},
	}

	pull, err := h.svc.PullReference(t.Context(), h.tenant, h.conn.ID, accountingsync.ReferenceKindAccount)
	require.NoError(t, err)
	assert.Equal(t, 3, pull.Fetched)
	assert.EqualValues(t, 1, pull.Removed, "Office Supplies was not returned")
	assert.Equal(t, []accountingsync.ReferenceKind{accountingsync.ReferenceKindAccount}, h.references.removed)
}

func TestPullReferenceReportsProviderFailures(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.connector.listErr = errors.New("401 unauthorized")

	_, err := h.svc.PullReference(t.Context(), h.tenant, h.conn.ID, accountingsync.ReferenceKindAccount)
	require.Error(t, err)
	require.Len(t, h.connService.failures, 1)
	assert.Empty(t, h.references.removed, "nothing is marked removed after a failed pull")

	_, err = h.svc.PullReference(t.Context(), h.tenant, h.conn.ID, accountingsync.ReferenceKind("Bogus"))
	require.Error(t, err)
}

func TestMarkRefreshFinishedRecordsTheOutcome(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	require.NoError(t, h.svc.MarkRefreshStarted(t.Context(), h.tenant, h.conn.ID))
	require.NoError(t, h.svc.MarkRefreshFinished(t.Context(), h.tenant, h.conn.ID, ""))
	require.NoError(t, h.svc.MarkRefreshFinished(t.Context(), h.tenant, h.conn.ID, strings.Repeat("x", 5000)))

	require.Len(t, h.connections.refresh, 3)
	assert.NotNil(t, h.connections.refresh[0].StartedAt)
	assert.NotNil(t, h.connections.refresh[1].RefreshedAt)
	assert.Empty(t, h.connections.refresh[1].Error)
	assert.Nil(t, h.connections.refresh[2].RefreshedAt)
	assert.Len(t, []rune(h.connections.refresh[2].Error), maxRefreshErrorLength)
}

func TestCreateReferenceRecordNeedsTheRevenueAccount(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.rescore(t)

	charge := h.object(t, accountingsync.TargetAccessorialCharge, h.chargeID)
	_, err := h.svc.CreateReferenceRecord(t.Context(), &services.CreateAccountingReferenceRecordRequest{
		TenantInfo: h.tenant,
		MappingID:  charge.ID,
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Empty(t, h.connector.created)
}

func TestCreateReferenceRecordCreatesAndConfirms(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.rescore(t)
	confirmRequired(t, h)
	h.connector.createdObj = &accountingsync.AccountingReferenceObject{
		Kind:       accountingsync.ReferenceKindItem,
		ExternalID: "91",
		Name:       "Detention",
		ItemType:   "Service",
		Active:     true,
	}

	charge := h.object(t, accountingsync.TargetAccessorialCharge, h.chargeID)
	updated, err := h.svc.CreateReferenceRecord(t.Context(), &services.CreateAccountingReferenceRecordRequest{
		TenantInfo: h.tenant,
		UserID:     h.tenant.UserID,
		MappingID:  charge.ID,
	})
	require.NoError(t, err)
	assert.Equal(t, accountingsync.MappingStateConfirmed, updated.State)
	assert.Equal(t, "91", updated.ExternalID)
	assert.Equal(t, accountingsync.MappingSourceCreatedInProvider, updated.Source)

	require.Len(t, h.connector.created, 1)
	sent := h.connector.created[0]
	require.NotNil(t, sent.Item)
	assert.Equal(t, "Detention", sent.Item.Name)
	assert.Equal(t, "DET", sent.Item.Sku)
	assert.Equal(t, "2", sent.Item.IncomeAccountID)
	assert.Equal(t, creationRequestID(charge), sent.RequestID)
	assert.LessOrEqual(t, len(sent.RequestID), 50)

	refs, err := h.references.GetByExternalIDs(t.Context(), &repositories.GetAccountingReferenceObjectsRequest{
		Kind:        accountingsync.ReferenceKindItem,
		ExternalIDs: []string{"91"},
	})
	require.NoError(t, err)
	assert.Len(t, refs, 1)

	_, err = h.svc.CreateReferenceRecord(t.Context(), &services.CreateAccountingReferenceRecordRequest{
		TenantInfo: h.tenant,
		MappingID:  charge.ID,
	})
	require.Error(t, err, "a confirmed mapping is not created twice")
	assert.Len(t, h.connector.created, 1)
}

func TestCreationRequestIDIsStablePerVersion(t *testing.T) {
	t.Parallel()
	row := &accountingsync.AccountingMapping{
		ID:           pulid.MustNew("acctm_"),
		ConnectionID: pulid.MustNew("acctc_"),
		Version:      3,
	}
	first := creationRequestID(row)
	assert.Equal(t, first, creationRequestID(row))
	assert.True(t, strings.HasPrefix(first, requestIDPrefix))

	row.Version++
	assert.NotEqual(t, first, creationRequestID(row))
}

func TestCreateReferenceRecordSuggestsAnExistingDuplicate(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.rescore(t)
	h.connector.createErr = errors.New("Duplicate Name Exists Error")
	h.connector.duplicate = true

	cus := h.object(t, accountingsync.TargetCustomer, h.customerID)
	_, err := h.svc.CreateReferenceRecord(t.Context(), &services.CreateAccountingReferenceRecordRequest{
		TenantInfo: h.tenant,
		MappingID:  cus.ID,
		Name:       "Acme Logistics, Inc.",
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Contains(t, err.Error(), "Acme Logistics, Inc.")
	assert.Empty(t, h.connService.failures, "a duplicate name is not a connection failure")

	sent := h.connector.created[0]
	require.NotNil(t, sent.Party)
	assert.Equal(t, "Acme Logistics, Inc.", sent.Party.DisplayName)
}

func TestCreateReferenceRecordRefusesAccounts(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.rescore(t)

	ap := h.keyed(t, accountingsync.TargetAccountRole, accountingsync.AccountRoleAP)
	_, err := h.svc.CreateReferenceRecord(t.Context(), &services.CreateAccountingReferenceRecordRequest{
		TenantInfo: h.tenant,
		MappingID:  ap.ID,
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
}

func modelHarness(t *testing.T) (*harness, *accountingsync.AccountingMapping) {
	t.Helper()
	h := newHarness(t)
	h.rescore(t)
	row := h.keyed(t, accountingsync.TargetAccountRole, accountingsync.AccountRoleAP)
	row.Signals.Candidates = []accountingsync.MappingCandidate{
		{ExternalID: "4", Name: "Office Supplies", Score: 0.4},
		{ExternalID: "5", Name: "Trade Payables", Score: 0.5},
	}
	_, err := h.mappings.ApplyScoring(t.Context(), []*accountingsync.AccountingMapping{row})
	require.NoError(t, err)
	return h, h.mappings.get(row.ID)
}

func TestModelPassAcceptsOnlyListedCandidates(t *testing.T) {
	t.Parallel()
	h, row := modelHarness(t)
	h.completion.reply = `{"answers":[
		{"target":"T1","candidate":"C9","confidence":0.99,"reason":"invented"},
		{"target":"T7","candidate":"C1","confidence":0.99,"reason":"no such target"}
	]}`

	applied, err := h.svc.ModelPass(t.Context(), h.tenant, h.conn.ID, []pulid.ID{row.ID})
	require.NoError(t, err)
	assert.Zero(t, applied)
	assert.Equal(t, accountingsync.MappingStateUnmatched, h.mappings.get(row.ID).State)

	require.Len(t, h.completion.requests, 1)
	sent := h.completion.requests[0]
	require.Len(t, sent.Context.Sections, 1)
	assert.False(t, sent.Context.Sections[0].Trusted)
	assert.NotContains(t, sent.Context.Sections[0].Content, "\"5\"", "external ids are never sent to the model")
}

func TestModelPassCapsConfidence(t *testing.T) {
	t.Parallel()
	h, row := modelHarness(t)
	h.completion.reply = `{"answers":[{"target":"T1","candidate":"C2","confidence":1,"reason":"Payables is the AP account"}]}`

	applied, err := h.svc.ModelPass(t.Context(), h.tenant, h.conn.ID, []pulid.ID{row.ID})
	require.NoError(t, err)
	assert.Equal(t, 1, applied)

	after := h.mappings.get(row.ID)
	assert.Equal(t, accountingsync.MappingStateProposed, after.State)
	assert.Equal(t, "5", after.ExternalID)
	assert.Equal(t, accountingsync.MappingSourceModel, after.Source)
	require.NotNil(t, after.Confidence)
	assert.InDelta(t, accountingsync.MaxModelConfidence, *after.Confidence, 1e-9)
	assert.False(t, after.Prechecked(), "a model pick is never pre-checked")
}

func TestModelPassWithoutAProvider(t *testing.T) {
	t.Parallel()
	h, row := modelHarness(t)
	h.completion.err = services.ErrNoProviderConfigured

	applied, err := h.svc.ModelPass(t.Context(), h.tenant, h.conn.ID, []pulid.ID{row.ID})
	require.NoError(t, err)
	assert.Zero(t, applied)

	h.completion.err = errors.New("model overloaded")
	_, err = h.svc.ModelPass(t.Context(), h.tenant, h.conn.ID, []pulid.ID{row.ID})
	require.Error(t, err)
}

func TestModelPassIgnoresAnUnreadableReply(t *testing.T) {
	t.Parallel()
	h, row := modelHarness(t)
	h.completion.reply = "not json"

	applied, err := h.svc.ModelPass(t.Context(), h.tenant, h.conn.ID, []pulid.ID{row.ID})
	require.NoError(t, err)
	assert.Zero(t, applied)
}
