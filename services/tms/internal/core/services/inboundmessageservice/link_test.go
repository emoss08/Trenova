package inboundmessageservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type linkStubMessageRepo struct {
	repositories.InboundMessageRepository

	message *inboundmessage.InboundMessage
	updated *inboundmessage.InboundMessage
}

func (s *linkStubMessageRepo) GetByID(
	_ context.Context,
	_ repositories.GetInboundMessageByIDRequest,
) (*inboundmessage.InboundMessage, error) {
	return s.message, nil
}

func (s *linkStubMessageRepo) Update(
	_ context.Context,
	message *inboundmessage.InboundMessage,
) (*inboundmessage.InboundMessage, error) {
	s.updated = message
	return message, nil
}

func linker(repo *linkStubMessageRepo, existing ...pulid.ID) *Service {
	known := make(map[pulid.ID]bool, len(existing))
	for _, id := range existing {
		known[id] = true
	}

	return &Service{
		l:           zap.NewNop(),
		messageRepo: repo,
		shipments:   &stubShipments{existing: known},
		parties:     &stubParties{existing: known},
	}
}

func linkRequest() LinkRequest {
	return LinkRequest{
		MessageID:  pulid.MustNew("imsg_"),
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		Reason:     "The PRO in the subject is this load's.",
	}
}

func TestLink_StoresRecordsThatExistWithTheReason(t *testing.T) {
	t.Parallel()

	shipmentID := pulid.MustNew("shp_")
	customerID := pulid.MustNew("cus_")
	repo := &linkStubMessageRepo{message: &inboundmessage.InboundMessage{}}
	req := linkRequest()
	req.ShipmentID = shipmentID
	req.CustomerID = customerID

	_, err := linker(repo, shipmentID, customerID).Link(t.Context(), req)
	require.NoError(t, err)

	require.NotNil(t, repo.updated)
	assert.Equal(t, shipmentID, repo.updated.MatchedShipmentID)
	assert.Equal(t, customerID, repo.updated.MatchedCustomerID)
	assert.Equal(t, req.Reason, repo.updated.MatchReason)
}

/*
The matched columns carry no foreign key, so without this check a link could
name a shipment from another tenant, or one that never existed, and the inbox
would store it and show a match to nothing.
*/
func TestLink_RefusesARecordTheTenantDoesNotHave(t *testing.T) {
	t.Parallel()

	cases := []struct {
		field string
		set   func(*LinkRequest, pulid.ID)
	}{
		{field: "shipmentId", set: func(r *LinkRequest, id pulid.ID) { r.ShipmentID = id }},
		{field: "customerId", set: func(r *LinkRequest, id pulid.ID) { r.CustomerID = id }},
		{field: "carrierId", set: func(r *LinkRequest, id pulid.ID) { r.CarrierID = id }},
	}

	for _, tc := range cases {
		t.Run(tc.field, func(t *testing.T) {
			t.Parallel()

			repo := &linkStubMessageRepo{message: &inboundmessage.InboundMessage{}}
			req := linkRequest()
			tc.set(&req, pulid.MustNew("xyz_"))

			_, err := linker(repo).Link(t.Context(), req)

			var multiErr *errortypes.MultiError
			require.ErrorAs(t, err, &multiErr)
			assert.Equal(t, tc.field, multiErr.Errors[0].Field)
			assert.Nil(t, repo.updated, "nothing may be written for a record that is not there")
		})
	}
}

func TestLink_RefusesRatherThanTrustsWhenItCannotCheck(t *testing.T) {
	t.Parallel()

	repo := &linkStubMessageRepo{message: &inboundmessage.InboundMessage{}}
	svc := &Service{l: zap.NewNop(), messageRepo: repo}
	req := linkRequest()
	req.ShipmentID = pulid.MustNew("shp_")

	_, err := svc.Link(t.Context(), req)

	require.Error(t, err)
	assert.Nil(t, repo.updated)
}
