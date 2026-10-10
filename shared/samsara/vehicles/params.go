package vehicles

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/shared/samsara/internal/httpx"
	samsaraspec "github.com/emoss08/trenova/shared/samsara/internal/samsaraspec"
)

const (
	maxStatsTypes       = 3
	maxStatsDecorations = 2
	auxInputPrefix      = "auxInput"
	groupedAuxInputMin  = 3
	groupedAuxInputMax  = 10
)

type StatsParams struct {
	After        string
	Time         *time.Time
	ParentTagIDs []string
	TagIDs       []string
	VehicleIDs   []string
	Types        []string
}

//nolint:gocritic // value receiver is kept for ergonomic immutable call sites.
func (p StatsParams) Validate() error {
	return validateTypes(p.Types, func(value string) bool {
		return samsaraspec.GetVehicleStatsParamsTypes(value).Valid()
	})
}

//nolint:gocritic // value receiver is kept for ergonomic immutable call sites.
func (p StatsParams) Query() url.Values {
	values := url.Values{}
	httpx.SetString(values, "after", p.After)
	httpx.SetTime(values, "time", p.Time)
	httpx.SetStringsCSV(values, "parentTagIds", p.ParentTagIDs)
	httpx.SetStringsCSV(values, "tagIds", p.TagIDs)
	httpx.SetStringsCSV(values, "vehicleIds", p.VehicleIDs)
	httpx.SetStringsCSV(values, "types", p.Types)
	return values
}

type StatsFeedParams struct {
	After        string
	ParentTagIDs []string
	TagIDs       []string
	VehicleIDs   []string
	Types        []string
	Decorations  []string
}

//nolint:gocritic // value receiver is kept for ergonomic immutable call sites.
func (p StatsFeedParams) Validate() error {
	if err := validateTypes(p.Types, func(value string) bool {
		return samsaraspec.GetVehicleStatsFeedParamsTypes(value).Valid()
	}); err != nil {
		return err
	}
	return validateDecorations(p.Decorations, func(value string) bool {
		return samsaraspec.GetVehicleStatsFeedParamsDecorations(value).Valid()
	})
}

//nolint:gocritic // value receiver is kept for ergonomic immutable call sites.
func (p StatsFeedParams) Query() url.Values {
	values := url.Values{}
	httpx.SetString(values, "after", p.After)
	httpx.SetStringsCSV(values, "parentTagIds", p.ParentTagIDs)
	httpx.SetStringsCSV(values, "tagIds", p.TagIDs)
	httpx.SetStringsCSV(values, "vehicleIds", p.VehicleIDs)
	httpx.SetStringsCSV(values, "types", p.Types)
	httpx.SetStringsCSV(values, "decorations", p.Decorations)
	return values
}

type StatsHistoryParams struct {
	After        string
	StartTime    time.Time
	EndTime      time.Time
	ParentTagIDs []string
	TagIDs       []string
	VehicleIDs   []string
	Types        []string
	Decorations  []string
}

//nolint:gocritic // value receiver is kept for ergonomic immutable call sites.
func (p StatsHistoryParams) Validate() error {
	if p.StartTime.IsZero() || p.EndTime.IsZero() {
		return ErrStatsTimeRangeRequired
	}
	if p.EndTime.Before(p.StartTime) {
		return ErrStatsTimeRangeInvalid
	}
	if err := validateTypes(p.Types, func(value string) bool {
		return samsaraspec.GetVehicleStatsHistoryParamsTypes(value).Valid()
	}); err != nil {
		return err
	}
	return validateDecorations(p.Decorations, func(value string) bool {
		return samsaraspec.GetVehicleStatsHistoryParamsDecorations(value).Valid()
	})
}

//nolint:gocritic // value receiver is kept for ergonomic immutable call sites.
func (p StatsHistoryParams) Query() url.Values {
	values := url.Values{}
	httpx.SetString(values, "after", p.After)
	httpx.SetTime(values, "startTime", &p.StartTime)
	httpx.SetTime(values, "endTime", &p.EndTime)
	httpx.SetStringsCSV(values, "parentTagIds", p.ParentTagIDs)
	httpx.SetStringsCSV(values, "tagIds", p.TagIDs)
	httpx.SetStringsCSV(values, "vehicleIds", p.VehicleIDs)
	httpx.SetStringsCSV(values, "types", p.Types)
	httpx.SetStringsCSV(values, "decorations", p.Decorations)
	return values
}

func validateTypes(types []string, valid func(string) bool) error {
	if len(types) == 0 {
		return ErrStatsTypesRequired
	}
	for _, statType := range types {
		if !valid(statType) {
			return fmt.Errorf("%w: %q", ErrStatsTypeInvalid, statType)
		}
	}
	if countedStatTypes(types) > maxStatsTypes {
		return ErrStatsTypesTooMany
	}
	return nil
}

func validateDecorations(decorations []string, valid func(string) bool) error {
	if len(decorations) > maxStatsDecorations {
		return ErrStatsDecorationsTooMany
	}
	for _, decoration := range decorations {
		if !valid(decoration) {
			return fmt.Errorf("%w: %q", ErrStatsDecorationInvalid, decoration)
		}
	}
	return nil
}

func countedStatTypes(types []string) int {
	count := 0
	groupedAuxCounted := false
	for _, statType := range types {
		if isGroupedAuxInput(statType) {
			if !groupedAuxCounted {
				groupedAuxCounted = true
				count++
			}
			continue
		}
		count++
	}
	return count
}

func isGroupedAuxInput(statType string) bool {
	suffix, ok := strings.CutPrefix(statType, auxInputPrefix)
	if !ok {
		return false
	}
	index, err := strconv.Atoi(suffix)
	if err != nil {
		return false
	}
	return index >= groupedAuxInputMin && index <= groupedAuxInputMax
}
