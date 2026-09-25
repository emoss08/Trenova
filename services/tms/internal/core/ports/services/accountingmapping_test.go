package services

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestSetAccountingMappingRequestValidateTarget(t *testing.T) {
	t.Parallel()

	recordID := pulid.MustNew("cus_")
	cases := []struct {
		name  string
		req   SetAccountingMappingRequest
		valid bool
	}{
		{name: "by mapping id", req: SetAccountingMappingRequest{MappingID: pulid.MustNew("acctm_")}, valid: true},
		{name: "nothing named", req: SetAccountingMappingRequest{}},
		{name: "unknown target type", req: SetAccountingMappingRequest{TargetType: "Bogus", TrenovaKey: "ARAccount"}},
		{
			name:  "record target by id",
			req:   SetAccountingMappingRequest{TargetType: accountingsync.TargetCustomer, TrenovaObjectID: recordID},
			valid: true,
		},
		{name: "record target without id", req: SetAccountingMappingRequest{TargetType: accountingsync.TargetCustomer}},
		{
			name: "record target by key",
			req:  SetAccountingMappingRequest{TargetType: accountingsync.TargetCarrier, TrenovaKey: "Net30"},
		},
		{
			name: "record target with both",
			req: SetAccountingMappingRequest{
				TargetType:      accountingsync.TargetCustomer,
				TrenovaObjectID: recordID,
				TrenovaKey:      "ACH",
			},
		},
		{
			name: "keyed target by key",
			req: SetAccountingMappingRequest{
				TargetType: accountingsync.TargetAccountRole,
				TrenovaKey: accountingsync.AccountRoleAR,
			},
			valid: true,
		},
		{
			name: "keyed target with a foreign key",
			req:  SetAccountingMappingRequest{TargetType: accountingsync.TargetPaymentTerm, TrenovaKey: "ACH"},
		},
		{
			name: "keyed target with a record id",
			req: SetAccountingMappingRequest{
				TargetType:      accountingsync.TargetPaymentMethod,
				TrenovaKey:      "ACH",
				TrenovaObjectID: recordID,
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.req.ValidateTarget()
			if tc.valid {
				assert.NoError(t, err)
				return
			}
			assert.Error(t, err)
		})
	}
}
