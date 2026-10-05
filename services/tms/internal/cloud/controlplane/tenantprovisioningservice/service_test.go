package tenantprovisioningservice

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/require"
)

type fakeProvisioningRepo struct {
	called bool
	result *tenant.ProvisioningResult
}

func (r *fakeProvisioningRepo) UpsertProvisioningSnapshot(
	_ context.Context,
	req *tenant.ProvisioningRequest,
) (*tenant.ProvisioningResult, error) {
	r.called = true
	if r.result != nil {
		return r.result, nil
	}
	return &tenant.ProvisioningResult{
		BusinessUnitID: req.Customer.ID,
		OrganizationID: req.Workspace.ID,
	}, nil
}

func TestService_ProvisionTenant(t *testing.T) {
	t.Run("upserts valid provisioning snapshot", func(t *testing.T) {
		repo := &fakeProvisioningRepo{}
		svc := &Service{
			cfg: &config.Config{
				Platform: config.PlatformConfig{InstanceID: "inst_test"},
			},
			repo: repo,
			now:  func() time.Time { return time.Unix(200, 0) },
		}

		result, err := svc.ProvisionTenant(t.Context(), validProvisioningRequest())

		require.NoError(t, err)
		require.True(t, repo.called)
		require.True(t, result.Accepted)
		require.Equal(t, int64(200), result.ReceivedAt)
	})

	t.Run("returns field errors for incomplete workspace", func(t *testing.T) {
		repo := &fakeProvisioningRepo{}
		svc := &Service{
			cfg: &config.Config{
				Platform: config.PlatformConfig{InstanceID: "inst_test"},
			},
			repo: repo,
			now:  time.Now,
		}
		req := validProvisioningRequest()
		req.Workspace.BucketName = ""
		req.Workspace.LoginSlug = ""

		_, err := svc.ProvisionTenant(t.Context(), req)

		require.Error(t, err)
		require.False(t, repo.called)
		var multiErr *errortypes.MultiError
		require.ErrorAs(t, err, &multiErr)
		require.True(t, multiErr.HasErrors())
	})

	t.Run("rejects snapshots for another instance", func(t *testing.T) {
		repo := &fakeProvisioningRepo{}
		svc := &Service{
			cfg: &config.Config{
				Platform: config.PlatformConfig{InstanceID: "inst_expected"},
			},
			repo: repo,
			now:  time.Now,
		}
		req := validProvisioningRequest()
		req.InstanceID = "inst_other"

		_, err := svc.ProvisionTenant(t.Context(), req)

		require.Error(t, err)
		require.False(t, repo.called)
	})
}

func validProvisioningRequest() *tenant.ProvisioningRequest {
	customerID := pulid.MustNew("bu_")
	return &tenant.ProvisioningRequest{
		InstanceID: "inst_test",
		Customer: tenant.ProvisioningCustomer{
			ID:   customerID,
			Name: "Acme Logistics",
			Code: "ACME",
		},
		Workspace: tenant.ProvisioningWorkspace{
			ID:             pulid.MustNew("org_"),
			BusinessUnitID: customerID,
			Name:           "Acme Northeast",
			State:          "NY",
			AddressLine1:   "100 Main Street",
			City:           "Albany",
			PostalCode:     "12207",
			Timezone:       "America/New_York",
			BucketName:     "acme-northeast",
			TaxID:          "12-3456789",
			ScacCode:       "ACME",
			DOTNumber:      "123456",
			LoginSlug:      "acme",
		},
		SentAt: 100,
	}
}
