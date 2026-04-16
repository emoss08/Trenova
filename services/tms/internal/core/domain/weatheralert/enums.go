package weatheralert

type AlertCategory string

const (
	AlertCategoryWinterWeather          = AlertCategory("winter_weather")
	AlertCategoryWindStorm              = AlertCategory("wind_storm")
	AlertCategoryFloodWater             = AlertCategory("flood_water")
	AlertCategoryFire                   = AlertCategory("fire")
	AlertCategoryHeat                   = AlertCategory("heat")
	AlertCategoryTornadoSevereStorm     = AlertCategory("tornado_severe_storm")
	AlertCategoryTropicalStormHurricane = AlertCategory("tropical_storm_hurricane")
	AlertCategoryOther                  = AlertCategory("other")
)

var validAlertCategories = map[AlertCategory]struct{}{
	AlertCategoryWinterWeather:          {},
	AlertCategoryWindStorm:              {},
	AlertCategoryFloodWater:             {},
	AlertCategoryFire:                   {},
	AlertCategoryHeat:                   {},
	AlertCategoryTornadoSevereStorm:     {},
	AlertCategoryTropicalStormHurricane: {},
	AlertCategoryOther:                  {},
}

func (a AlertCategory) String() string {
	return string(a)
}

func (a AlertCategory) IsValid() bool {
	_, ok := validAlertCategories[a]
	return ok
}

type ActivityType string

const (
	ActivityTypeIssued    = ActivityType("issued")
	ActivityTypeUpdated   = ActivityType("updated")
	ActivityTypeExpired   = ActivityType("expired")
	ActivityTypeCancelled = ActivityType("cancelled")
)

func (a ActivityType) String() string {
	return string(a)
}
