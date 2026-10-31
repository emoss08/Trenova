package billingqueueservice

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateagreementports"
	"github.com/emoss08/trenova/internal/testutil/mocks/rateagreementportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

func ratedItem(agreementID *pulid.ID, template *formulatemplate.FormulaTemplate) *billingqueue.BillingQueueItem {
	return &billingqueue.BillingQueueItem{
		ID: pulid.MustNew("bqi_"),
		Shipment: &shipment.Shipment{
			RateAgreementID: agreementID,
			FormulaTemplate: template,
		},
	}
}

func TestRatedBy_NamesTheContractThatPricedTheShipment(t *testing.T) {
	t.Parallel()

	agreementID := pulid.MustNew("ra_")
	agreements := rateagreementportsmock.NewMockRateAgreementRepository(t)
	agreements.EXPECT().GetByID(mock.Anything, mock.MatchedBy(
		func(req *rateagreementports.GetRateAgreementByIDRequest) bool {
			return req.RateAgreementID == agreementID && !req.IncludeChildren
		},
	)).Return(&rateagreement.RateAgreement{ID: agreementID, Code: "RA-2041", Name: "Acme 2026"}, nil)
	s := &Service{l: zap.NewNop(), agreements: agreements}
	template := &formulatemplate.FormulaTemplate{ID: pulid.MustNew("ft_"), Name: "Per mile"}

	got := s.ratedBy(t.Context(), ratedItem(&agreementID, template), pagination.TenantInfo{})

	assert.Equal(t, &billingqueue.RatedBy{
		Kind: billingqueue.RatedByAgreement,
		ID:   agreementID,
		Code: "RA-2041",
		Name: "Acme 2026",
	}, got)
}

func TestRatedBy_NamesTheFormulaTemplateWithoutAContract(t *testing.T) {
	t.Parallel()

	template := &formulatemplate.FormulaTemplate{ID: pulid.MustNew("ft_"), Name: "Per mile"}
	s := &Service{l: zap.NewNop(), agreements: rateagreementportsmock.NewMockRateAgreementRepository(t)}

	got := s.ratedBy(t.Context(), ratedItem(nil, template), pagination.TenantInfo{})

	assert.Equal(t, &billingqueue.RatedBy{
		Kind: billingqueue.RatedByTemplate,
		ID:   template.ID,
		Name: "Per mile",
	}, got)
}

func TestRatedBy_FallsBackToTheTemplateWhenTheContractCannotBeRead(t *testing.T) {
	t.Parallel()

	agreementID := pulid.MustNew("ra_")
	agreements := rateagreementportsmock.NewMockRateAgreementRepository(t)
	agreements.EXPECT().GetByID(mock.Anything, mock.Anything).Return(nil, errors.New("gone"))
	s := &Service{l: zap.NewNop(), agreements: agreements}
	template := &formulatemplate.FormulaTemplate{ID: pulid.MustNew("ft_"), Name: "Per mile"}

	got := s.ratedBy(t.Context(), ratedItem(&agreementID, template), pagination.TenantInfo{})

	assert.Equal(t, billingqueue.RatedByTemplate, got.Kind)
	assert.Nil(t, s.ratedBy(t.Context(), ratedItem(nil, nil), pagination.TenantInfo{}), "nothing priced it")
}
