package factoringledgerservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringledgerports"
	"github.com/stretchr/testify/assert"
)

const historyDay = int64(86400)

func TestWalkBackEndsEachDayWithTheReserveAfterItsMovements(t *testing.T) {
	t.Parallel()

	first := int64(1_780_000_000) - int64(1_780_000_000)%historyDay
	read := &factoringledgerports.ReserveHistoryRead{
		CurrentMinor: 1_000,
		Changes: []*factoringledgerports.ReserveDayChange{
			{Day: first + historyDay, ChangeMinor: 300},
			{Day: first + 2*historyDay + 3600, ChangeMinor: -100},
			{Day: first + 2*historyDay, ChangeMinor: 50},
		},
	}

	assert.Equal(t, []*factoringledgerports.ReserveDay{
		{Day: first, ReserveMinor: 750},
		{Day: first + historyDay, ReserveMinor: 1_050},
		{Day: first + 2*historyDay, ReserveMinor: 1_000},
	}, walkBack(read, first, 3))
}

func TestWalkBackLeavesOutMovementsDatedAfterTheWindow(t *testing.T) {
	t.Parallel()

	first := int64(1_780_000_000) - int64(1_780_000_000)%historyDay
	read := &factoringledgerports.ReserveHistoryRead{
		CurrentMinor: 500,
		Changes: []*factoringledgerports.ReserveDayChange{
			{Day: first + 5*historyDay, ChangeMinor: 200},
		},
	}

	assert.Equal(t, []*factoringledgerports.ReserveDay{
		{Day: first, ReserveMinor: 300},
		{Day: first + historyDay, ReserveMinor: 300},
	}, walkBack(read, first, 2), "a movement dated past today is not in today's figure")
}
