package ratezone

type ZoneKind string

const (
	ZoneKindCustom   = ZoneKind("Custom")
	ZoneKindKMA      = ZoneKind("KMA")
	ZoneKindRegional = ZoneKind("Regional")
	ZoneKindMetro    = ZoneKind("Metro")
	ZoneKindCountry  = ZoneKind("Country")
)

func (zk ZoneKind) String() string {
	return string(zk)
}

func (zk ZoneKind) IsValid() bool {
	switch zk {
	case ZoneKindCustom,
		ZoneKindKMA,
		ZoneKindRegional,
		ZoneKindMetro,
		ZoneKindCountry:
		return true
	default:
		return false
	}
}
