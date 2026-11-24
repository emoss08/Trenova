package carriercoverrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/ports/carriercoverports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

const (
	testDay = int64(86400)
	testNow = int64(1_795_000_000)
)

func endingCover(name string, status carrier.Status, endsIn int64) *carriercoverports.Cover {
	return &carriercoverports.Cover{
		CarrierID: pulid.MustNew("car_"),
		Name:      name,
		Status:    status,
		Policy:    &carrier.CarrierInsurancePolicy{ExpirationDate: testNow + endsIn*testDay},
	}
}

func TestSoonestEndingKeepsActiveCarriersWhoseCoverEndsInTheWindowSoonestFirst(t *testing.T) {
	t.Parallel()

	blue := endingCover("Blue Line", carrier.StatusActive, 18)
	ridge := endingCover("Ridgeline", carrier.StatusActive, 6)
	lapsed := endingCover("Ironhorse", carrier.StatusActive, -30)
	longAgo := endingCover("Old", carrier.StatusActive, -120)
	renewed := endingCover("Renewed", carrier.StatusActive, 200)
	inactive := endingCover("Inactive", carrier.StatusInactive, 3)
	none := &carriercoverports.Cover{CarrierID: pulid.MustNew("car_"), Status: carrier.StatusActive}
	covers := map[pulid.ID]*carriercoverports.Cover{}
	for _, cover := range []*carriercoverports.Cover{
		blue, ridge, lapsed, longAgo, renewed, inactive, none,
	} {
		covers[cover.CarrierID] = cover
	}

	got := soonestEnding(covers, &carriercoverports.EndingRequest{
		Now: testNow, Until: testNow + 30*testDay, Limit: 10,
	})

	assert.Equal(t, []*carriercoverports.Cover{lapsed, ridge, blue}, got)

	first := soonestEnding(covers, &carriercoverports.EndingRequest{
		Now: testNow, Until: testNow + 30*testDay, Limit: 1,
	})
	assert.Equal(t, []*carriercoverports.Cover{lapsed}, first)
}
