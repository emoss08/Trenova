//go:build integration

package customerrepository

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype/documenttypedb"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks/platformplanportsmock"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type customerTenant struct {
	ctx  context.Context
	info pagination.TenantInfo
	data *seedtest.TestData
}

type customerFixture struct {
	conn  *postgres.Connection
	admin *keel.DB
	repo  customerports.CustomerRepository
	a     customerTenant
	b     customerTenant
}

func newCustomerFixture(t *testing.T) *customerFixture {
	t.Helper()

	ctx, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	a := seedtest.SeedFullTestData(t, ctx, admin)
	b := seedtest.SeedAdditionalTenant(t, ctx, admin, "CB")
	conn := dbtest.NewRLSConnection(t, admin, dsn)

	return &customerFixture{
		conn:  conn,
		admin: admin,
		repo:  New(Params{DB: conn, Logger: zap.NewNop()}),
		a:     customerTenant{ctx: a.Ctx(t.Context()), info: a.Info(), data: a},
		b:     customerTenant{ctx: b.Ctx(t.Context()), info: b.Info(), data: b},
	}
}

func newCustomer(tenant customerTenant, code string) *customer.Customer {
	return &customer.Customer{
		OrganizationID: tenant.info.OrgID,
		BusinessUnitID: tenant.info.BuID,
		StateID:        tenant.data.State.ID,
		Code:           code,
		Name:           code + " Customer",
		AddressLine1:   "100 Customer Way",
		City:           "Charlotte",
		PostalCode:     "28202",
	}
}

func (f *customerFixture) documentType(
	t *testing.T,
	tenant customerTenant,
	code string,
) *documenttype.DocumentType {
	t.Helper()

	entity := &documenttype.DocumentType{
		OrganizationID:         tenant.info.OrgID,
		BusinessUnitID:         tenant.info.BuID,
		Code:                   code,
		Name:                   code + " document",
		DocumentClassification: documenttype.ClassificationPublic,
		DocumentCategory:       documenttype.CategoryShipment,
	}
	require.NoError(t, documenttypedb.DocumentTypes.Create(tenant.ctx, f.admin, entity))

	return entity
}

func (f *customerFixture) linkedDocumentTypes(
	t *testing.T,
	tenant customerTenant,
	profileID pulid.ID,
) []pulid.ID {
	t.Helper()

	links := customerdb.CustomerBillingProfileDocumentTypes
	rows, err := links.Where(links.BillingProfileID.Eq(profileID)).
		Order(links.DocumentTypeID.Asc()).
		All(tenant.ctx, f.admin)
	require.NoError(t, err)

	ids := make([]pulid.ID, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.DocumentTypeID)
	}

	return ids
}

func TestCreateAddsDefaultBillingProfileWhenMissing(t *testing.T) {
	f := newCustomerFixture(t)

	created, err := f.repo.Create(f.a.ctx, newCustomer(f.a, "CUST1"))
	require.NoError(t, err)
	require.NotNil(t, created.BillingProfile)
	assert.Equal(t, domaintypes.StatusActive, created.Status)

	profile, err := f.repo.GetBillingProfile(
		f.a.ctx,
		customerports.GetCustomerBillingProfileRequest{
			CustomerID: created.ID,
			TenantInfo: f.a.info,
		},
	)
	require.NoError(t, err)
	assert.Equal(t, created.BillingProfile.ID, profile.ID)
	assert.Equal(t, created.ID, profile.CustomerID)
	assert.Equal(t, customer.CreditStatusActive, profile.CreditStatus)
	assert.Empty(t, profile.DocumentTypes)
}

func TestCreateRefusesACustomerPastTheQuota(t *testing.T) {
	f := newCustomerFixture(t)
	guard := platformplanportsmock.NewMockQuotaGuard(t)
	repo := New(Params{DB: f.conn, Logger: zap.NewNop(), Quota: guard})
	entity := newCustomer(f.a, "CUST9")

	guard.EXPECT().Enforce(mock.Anything, &platformplanports.QuotaRequest{
		TenantInfo: pagination.TenantInfo{OrgID: f.a.info.OrgID, BuID: f.a.info.BuID},
		Meter:      platformcatalog.MeterCustomersTotal,
		Quantity:   1,
	}).Return(errortypes.NewQuotaExceededError(string(platformcatalog.MeterCustomersTotal), 8, 8, "free_demo"))

	created, err := repo.Create(f.a.ctx, entity)

	require.Nil(t, created)
	require.True(t, errortypes.IsQuotaExceededError(err))

	c := customerdb.Customers
	exists, err := c.Where(c.Code.Eq("CUST9")).Exists(f.a.ctx, f.admin)
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestUpdateUpsertsBillingProfileAndSyncsDocumentTypes(t *testing.T) {
	f := newCustomerFixture(t)
	pod := f.documentType(t, f.a, "POD")
	bol := f.documentType(t, f.a, "BOL")
	rate := f.documentType(t, f.a, "RATE")

	entity := newCustomer(f.a, "CUST2")
	entity.BillingProfile = customer.NewDefaultBillingProfile(f.a.info.OrgID, f.a.info.BuID, "")
	entity.BillingProfile.DocumentTypes = []*documenttype.DocumentType{pod, bol}
	created, err := f.repo.Create(f.a.ctx, entity)
	require.NoError(t, err)
	profileID := created.BillingProfile.ID
	assert.ElementsMatch(t, []pulid.ID{pod.ID, bol.ID}, f.linkedDocumentTypes(t, f.a, profileID))

	stored, err := f.repo.GetByID(f.a.ctx, customerports.GetCustomerByIDRequest{
		ID:         created.ID,
		TenantInfo: f.a.info,
		CustomerFilterOptions: customerports.CustomerFilterOptions{
			IncludeBillingProfile: true,
		},
	})
	require.NoError(t, err)

	stored.Name = "Renamed Customer"
	stored.BillingProfile.ID = pulid.MustNew("cbp_")
	stored.BillingProfile.CreditStatus = customer.CreditStatusHold
	stored.BillingProfile.InvoiceCopies = 3
	stored.BillingProfile.DocumentTypes = []*documenttype.DocumentType{bol, rate, nil}

	updated, err := f.repo.Update(f.a.ctx, stored)
	require.NoError(t, err)
	assert.Equal(t, "Renamed Customer", updated.Name)
	assert.Equal(t, profileID, updated.BillingProfile.ID)
	assert.Equal(t, customer.CreditStatusHold, updated.BillingProfile.CreditStatus)
	assert.Equal(t, int16(3), updated.BillingProfile.InvoiceCopies)
	assert.Greater(t, updated.BillingProfile.Version, created.BillingProfile.Version)
	assert.ElementsMatch(t, []pulid.ID{bol.ID, rate.ID}, f.linkedDocumentTypes(t, f.a, profileID))

	cbp := customerdb.CustomerBillingProfiles
	count, err := cbp.Where(cbp.CustomerID.Eq(created.ID)).Count(f.a.ctx, f.admin)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestBillingProfileUpsertClearsAutomaticHoldWhenStatusChanges(t *testing.T) {
	f := newCustomerFixture(t)

	created, err := f.repo.Create(f.a.ctx, newCustomer(f.a, "CUST3"))
	require.NoError(t, err)

	cbp := customerdb.CustomerBillingProfiles
	_, err = cbp.Where(cbp.ID.Eq(created.BillingProfile.ID)).Update(
		f.a.ctx,
		f.admin,
		cbp.CreditStatus.To(customer.CreditStatusHold),
		cbp.CreditHoldAutomatic.To(true),
	)
	require.NoError(t, err)

	created.BillingProfile.CreditStatus = customer.CreditStatusHold
	created.BillingProfile.CreditHoldAutomatic = false
	_, err = f.repo.Update(f.a.ctx, created)
	require.NoError(t, err)

	profile, err := cbp.Get(f.a.ctx, f.admin, created.BillingProfile.ID)
	require.NoError(t, err)
	assert.True(t, profile.CreditHoldAutomatic, "an unchanged status keeps the automatic hold")

	stored, err := f.repo.GetByID(f.a.ctx, customerports.GetCustomerByIDRequest{
		ID:                    created.ID,
		TenantInfo:            f.a.info,
		CustomerFilterOptions: customerports.CustomerFilterOptions{IncludeBillingProfile: true},
	})
	require.NoError(t, err)
	stored.BillingProfile.CreditStatus = customer.CreditStatusActive
	_, err = f.repo.Update(f.a.ctx, stored)
	require.NoError(t, err)

	profile, err = cbp.Get(f.a.ctx, f.admin, created.BillingProfile.ID)
	require.NoError(t, err)
	assert.False(t, profile.CreditHoldAutomatic, "a status change clears the automatic hold")
}

func TestUpdateReturnsVersionMismatchForAStaleCustomer(t *testing.T) {
	f := newCustomerFixture(t)

	created, err := f.repo.Create(f.a.ctx, newCustomer(f.a, "CUST4"))
	require.NoError(t, err)

	stale := *created
	stale.BillingProfile = nil
	stale.Version--

	_, err = f.repo.Update(f.a.ctx, &stale)
	require.Error(t, err)
	assert.True(t, errortypes.IsVersionMismatchError(err), err.Error())
}

func TestCreateMapsDuplicateCodeToAFieldError(t *testing.T) {
	f := newCustomerFixture(t)

	_, err := f.repo.Create(f.a.ctx, newCustomer(f.a, "DUP1"))
	require.NoError(t, err)

	_, err = f.repo.Create(f.a.ctx, newCustomer(f.a, "DUP1"))
	require.Error(t, err)

	var fieldErr *errortypes.Error
	require.True(t, errors.As(err, &fieldErr), err.Error())
	assert.Equal(t, "code", fieldErr.Field)

	_, err = f.repo.Create(f.b.ctx, newCustomer(f.b, "DUP1"))
	require.NoError(t, err, "codes are unique per tenant")
}

func TestBulkUpdateStatusReturnsVersionMismatchWhenNoRowMatches(t *testing.T) {
	f := newCustomerFixture(t)

	other, err := f.repo.Create(f.b.ctx, newCustomer(f.b, "CUST5"))
	require.NoError(t, err)
	customerIDs := []pulid.ID{pulid.MustNew("cus_"), other.ID}

	updated, err := f.repo.BulkUpdateStatus(f.a.ctx, &customerports.BulkUpdateCustomerStatusRequest{
		TenantInfo:  f.a.info,
		CustomerIDs: customerIDs,
		Status:      domaintypes.StatusInactive,
	})

	require.Nil(t, updated)
	require.Error(t, err)
	assert.True(t, errortypes.IsVersionMismatchError(err), err.Error())
	assert.Contains(t, err.Error(), customerIDs[0].String())
	assert.Contains(t, err.Error(), customerIDs[1].String())
}

func TestBulkUpdateStatusUpdatesTheTenantsCustomers(t *testing.T) {
	f := newCustomerFixture(t)

	first, err := f.repo.Create(f.a.ctx, newCustomer(f.a, "CUST6"))
	require.NoError(t, err)
	second, err := f.repo.Create(f.a.ctx, newCustomer(f.a, "CUST7"))
	require.NoError(t, err)

	updated, err := f.repo.BulkUpdateStatus(f.a.ctx, &customerports.BulkUpdateCustomerStatusRequest{
		TenantInfo:  f.a.info,
		CustomerIDs: []pulid.ID{first.ID, second.ID},
		Status:      domaintypes.StatusInactive,
	})
	require.NoError(t, err)
	require.Len(t, updated, 2)
	for _, entity := range updated {
		assert.Equal(t, domaintypes.StatusInactive, entity.Status)
		assert.Equal(t, first.Version+1, entity.Version)
	}
}

func TestGetBillingProfileStaysInsideTheTenant(t *testing.T) {
	f := newCustomerFixture(t)

	created, err := f.repo.Create(f.b.ctx, newCustomer(f.b, "CUST8"))
	require.NoError(t, err)

	_, err = f.repo.GetBillingProfile(f.a.ctx, customerports.GetCustomerBillingProfileRequest{
		CustomerID: created.ID,
		TenantInfo: f.a.info,
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err))

	profile, err := f.repo.GetBillingProfile(
		f.b.ctx,
		customerports.GetCustomerBillingProfileRequest{
			CustomerID: created.ID,
			TenantInfo: f.b.info,
		},
	)
	require.NoError(t, err)
	assert.Equal(t, created.ID, profile.CustomerID)
}
