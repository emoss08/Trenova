package auditrollup

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/keel/analytics/rollup"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/rollupread"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit/auditdb"
)

const (
	HourlyName     = "audit_hourly"
	DailyName      = "audit_daily"
	ActorDailyName = "audit_actor_daily"

	Resource      = "resource"
	Category      = "category"
	PrincipalID   = "principal_id"
	PrincipalType = "principal_type"

	Entries        = "row_count"
	People         = "people"
	AgentsAsked    = "agents_asked"
	AgentsOwn      = "agents_own"
	Integrations   = "integrations"
	Jobs           = "jobs"
	Sensitive      = "sensitive"
	Security       = "security"
	DeletesExports = "deletes_exports"
	PeopleOffHours = "people_off_hours"
)

var FigureColumns = []string{
	Entries, People, AgentsAsked, AgentsOwn, Integrations, Jobs,
	Sensitive, Security, DeletesExports, PeopleOffHours,
}

var Hourly = figuresRollup(HourlyName, rollup.Hour(auditdb.Entries.Timestamp).Epoch())

var Daily = figuresRollup(DailyName, rollup.Day(auditdb.Entries.Timestamp).Epoch())

var ActorDaily = rollup.MustDefine(ActorDailyName, auditdb.EntriesSchema(),
	rollup.Day(auditdb.Entries.Timestamp).Epoch(),
	rollup.Dims(
		auditdb.Entries.PrincipalID,
		auditdb.Entries.PrincipalType,
		auditdb.Entries.Resource,
		auditdb.Entries.Category,
	),
	rollup.Measures(rollup.Count(), rollup.CountWhere(Sensitive, sensitive())),
	rollup.TenantZone(OrganizationZone()),
	rollup.WithoutSourceIndex(),
)

var (
	HourlyTable     = rollupread.Of(Hourly)
	DailyTable      = rollupread.Of(Daily)
	ActorDailyTable = rollupread.Of(ActorDaily)
)

func All() []*rollup.Rollup {
	return []*rollup.Rollup{Hourly, Daily, ActorDaily}
}

func OrganizationZone() keel.StringExpr {
	return rollupread.OrganizationZone(auditdb.Entries.OrganizationID)
}

func figuresRollup(name string, grain rollup.Grain) *rollup.Rollup {
	e := auditdb.Entries
	return rollup.MustDefine(name, auditdb.EntriesSchema(), grain,
		rollup.Dims(e.Resource, e.Category),
		figures(),
		rollup.TenantZone(OrganizationZone()),
		rollup.WithoutSourceIndex(),
	)
}

func figures() []rollup.Measure {
	e := auditdb.Entries
	people := e.Channel.In(audit.PeopleChannels...)
	hour := rollupread.LocalHour(e.Timestamp, OrganizationZone())

	return rollup.Measures(
		rollup.Count(),
		rollup.CountWhere(People, people),
		rollup.CountWhere(AgentsAsked, e.Channel.Eq(audit.ChannelAgentConversation)),
		rollup.CountWhere(AgentsOwn, e.Channel.Eq(audit.ChannelAgentRun)),
		rollup.CountWhere(Integrations, e.Channel.In(audit.IntegrationChannels...)),
		rollup.CountWhere(Jobs, e.Channel.In(audit.JobChannels...)),
		rollup.CountWhere(Sensitive, sensitive()),
		rollup.CountWhere(Security, e.Category.Eq(audit.CategorySecurity)),
		rollup.CountWhere(DeletesExports, e.Operation.In(audit.SensitiveOperations...)),
		rollup.CountWhere(PeopleOffHours, keel.And(
			people,
			keel.Or(hour.Gte(audit.OffHoursFrom), hour.Lt(audit.OffHoursTo)),
		)),
	)
}

func sensitive() keel.Pred {
	e := auditdb.Entries
	return keel.Or(
		e.Category.Eq(audit.CategorySecurity),
		e.Critical.IsTrue(),
		e.Operation.In(audit.SensitiveOperations...),
	)
}
