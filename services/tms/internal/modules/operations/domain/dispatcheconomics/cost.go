package dispatcheconomics

type Component string

const (
	MinMultiplier = 0.25
	MaxMultiplier = 4.0
)

const (
	ComponentEmptyMiles         = Component("emptyMiles")
	ComponentLateness           = Component("lateness")
	ComponentIdleWait           = Component("idleWait")
	ComponentRest               = Component("rest")
	ComponentHomeTime           = Component("homeTime")
	ComponentFairness           = Component("fairness")
	ComponentCustomerExperience = Component("customerExperience")
	ComponentOnTimeHistory      = Component("onTimeHistory")
	ComponentSafety             = Component("safety")
	ComponentAcceptance         = Component("acceptance")
	ComponentTrailerContinuity  = Component("trailerContinuity")
	ComponentHaulFit            = Component("haulFit")
	ComponentTimeOffBuffer      = Component("timeOffBuffer")
	ComponentEndMarket          = Component("endMarket")
)

func SoftComponents() []Component {
	return []Component{
		ComponentHomeTime,
		ComponentFairness,
		ComponentCustomerExperience,
		ComponentOnTimeHistory,
		ComponentSafety,
		ComponentAcceptance,
		ComponentTrailerContinuity,
		ComponentHaulFit,
		ComponentTimeOffBuffer,
	}
}

func (c Component) Soft() bool {
	switch c {
	case ComponentHomeTime,
		ComponentFairness,
		ComponentCustomerExperience,
		ComponentOnTimeHistory,
		ComponentSafety,
		ComponentAcceptance,
		ComponentTrailerContinuity,
		ComponentHaulFit,
		ComponentTimeOffBuffer:
		return true
	case ComponentEmptyMiles,
		ComponentLateness,
		ComponentIdleWait,
		ComponentRest,
		ComponentEndMarket:
		return false
	default:
		return false
	}
}

type Line struct {
	Component Component `json:"component"`
	Cents     int64     `json:"cents"`
}

type Cost struct {
	TotalCents int64  `json:"totalCents"`
	Lines      []Line `json:"lines"`
}

func (c Component) Label() string {
	switch c {
	case ComponentEmptyMiles:
		return "empty miles"
	case ComponentLateness:
		return "lateness"
	case ComponentIdleWait:
		return "waiting"
	case ComponentRest:
		return "rest before driving"
	case ComponentHomeTime:
		return "time away from home"
	case ComponentFairness:
		return "miles above the fleet median"
	case ComponentCustomerExperience:
		return "new to the customer"
	case ComponentOnTimeHistory:
		return "on-time history"
	case ComponentSafety:
		return "safety history"
	case ComponentAcceptance:
		return "acceptance"
	case ComponentTrailerContinuity:
		return "trailer swap"
	case ComponentHaulFit:
		return "haul fit"
	case ComponentTimeOffBuffer:
		return "close to time off"
	case ComponentEndMarket:
		return "where the driver ends up"
	default:
		return string(c)
	}
}
