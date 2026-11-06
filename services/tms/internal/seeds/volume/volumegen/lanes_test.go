package volumegen_test

import (
	"fmt"
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
	"github.com/emoss08/trenova/internal/seeds/volume/volumegen"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	laneNow       = int64(1_793_577_600)
	laneWeeks     = 52
	laneCustomers = 40
)

func laneParams() *volumegen.LaneProgramParams {
	cities := volumegen.Cities
	shippers := make([]*volumegen.LocationRecord, 0, len(cities))
	consignees := make([]*volumegen.LocationRecord, 0, len(cities))
	for i := range cities {
		shippers = append(shippers, &volumegen.LocationRecord{
			ID:      pulid.ID(fmt.Sprintf("loc_s%03d", i)),
			City:    &cities[i],
			Shipper: true,
		})
		consignees = append(consignees, &volumegen.LocationRecord{
			ID:        pulid.ID(fmt.Sprintf("loc_c%03d", i)),
			City:      &cities[i],
			Consignee: true,
		})
	}
	customers := make([]*volumegen.CustomerRecord, 0, laneCustomers)
	for i := range laneCustomers {
		home := i * 7 % len(cities)
		customer := &volumegen.CustomerRecord{
			ID:     pulid.ID(fmt.Sprintf("cus_%03d", i)),
			Name:   fmt.Sprintf("Customer %d", i),
			Weight: max(1, 900/(i+1)),
			Home:   &cities[home],
		}
		if i%3 == 0 {
			customer.Facilities = []*volumegen.LocationRecord{shippers[home]}
		}
		customers = append(customers, customer)
	}

	return &volumegen.LaneProgramParams{
		Customers:  customers,
		Shippers:   shippers,
		Consignees: consignees,
		Begin:      laneNow - laneWeeks*dispatchmarket.WeekSeconds,
		End:        laneNow + 14*volumegen.Day,
		Now:        laneNow,
		Loads:      8_000,
	}
}

func leadBand(bands []volumegen.LeadBand, lead int64) int {
	for i, band := range bands {
		if lead >= band.Min && lead <= band.Max {
			return i
		}
	}

	return -1
}

func TestPlanLanePrograms_IsReproducibleFromTheSeed(t *testing.T) {
	t.Parallel()

	params := laneParams()
	first := volumegen.PlanLanePrograms(volumegen.NewFaker(42), params)
	second := volumegen.PlanLanePrograms(volumegen.NewFaker(42), params)
	other := volumegen.PlanLanePrograms(volumegen.NewFaker(43), params)

	require.Len(t, first.Programs, 5)
	assert.Equal(t, first, second)
	assert.NotEqual(t, first.Loads, other.Loads)
}

func TestPlanLanePrograms_GivesEachProgramItsOwnMapArea(t *testing.T) {
	t.Parallel()

	plan := volumegen.PlanLanePrograms(volumegen.NewFaker(42), laneParams())

	seen := map[int64]string{}
	for _, program := range plan.Programs {
		cell, ok := dispatchmarket.CellOf(dispatchmarket.Point{
			Latitude:  program.Origin.City.Lat,
			Longitude: program.Origin.City.Lon,
		})
		require.True(t, ok)
		assert.NotContains(t, seen, cell, "%s shares an area with %s", program.Name, seen[cell])
		seen[cell] = program.Name
	}
}

func TestPlanLanePrograms_LoadsFollowTheirProgramsRhythmAndLeadMix(t *testing.T) {
	t.Parallel()

	params := laneParams()
	plan := volumegen.PlanLanePrograms(volumegen.NewFaker(7), params)
	require.Len(t, plan.Programs, 5)

	for _, program := range plan.Programs {
		heavySlots := make([]bool, len(program.SlotWeights))
		heavyShare := 0.0
		for slot, weight := range program.SlotWeights {
			if weight > 1/float64(len(program.SlotWeights)) {
				heavySlots[slot] = true
				heavyShare += weight
			}
		}
		weights := 0
		for _, band := range program.Leads {
			weights += band.Weight
		}

		loads, inHeavy, booked := 0.0, 0.0, 0.0
		bands := make([]float64, len(program.Leads))
		for _, load := range plan.Loads {
			if load.Program != program {
				continue
			}
			loads++
			assert.Less(t, load.CreatedAt, params.Now,
				"%s books nothing in the future", program.Name)
			if heavySlots[dispatchmarket.SlotOf(load.PickupAt)] {
				inHeavy++
			}
			if load.PickupAt >= params.Now {
				continue
			}
			booked++
			band := leadBand(program.Leads, load.PickupAt-load.CreatedAt)
			require.GreaterOrEqual(t, band, 0, "%s lead outside its bands", program.Name)
			bands[band]++
		}

		require.Greater(t, loads, 500.0, program.Name)
		assert.InDelta(t, heavyShare, inHeavy/loads, 0.05,
			"%s loads in its heavy slots", program.Name)
		for i, band := range program.Leads {
			assert.InDelta(t, float64(band.Weight)/float64(weights), bands[i]/booked, 0.05,
				"%s booked %d–%d s ahead", program.Name, band.Min, band.Max)
		}
	}
}
