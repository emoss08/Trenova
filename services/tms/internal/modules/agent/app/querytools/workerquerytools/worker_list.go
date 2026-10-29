package workerquerytools

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/fieldgate"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/listtool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/textsim"
)

var (
	workerTypes      = []string{"Employee", "Contractor"}
	driverTypes      = []string{"Local", "Regional", "OTR", "Team"}
	endorsementCodes = []string{"O", "N", "H", "X", "P", "T"}
	complianceStates = []string{"Compliant", "NonCompliant", "Pending"}
	cdlClasses       = []string{"A", "B", "C"}
)

const (
	workerProfilePath = "profile."
	workerCityField   = "city"
)

// endorsementNote spells the codes out.
//
// The column stores a single letter, and X is the one that matters: it means
// tanker AND hazmat, so a hazmat question answered with endorsement = 'H' alone
// undercounts the fleet. No model infers that, and one that guesses "Hazmat"
// gets an empty page it will report as nobody holding one.
const endorsementNote = "O none, N tanker, H hazmat, X tanker+hazmat, " +
	"P passenger, T doubles/triples; hazmat is H or X"

type workerRow struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	Type       string `json:"type"`
	DriverType string `json:"driverType"`
	City       string `json:"city,omitempty"`
	FleetCode  string `json:"fleetCode,omitempty"`
	// CanBeAssigned folds the compliance block into the stored flag, because
	// the stored flag alone answers a narrower question than its name does.
	// The column means "no employment-level block"; the domain additionally
	// refuses to dispatch a worker whose credentials are NonCompliant
	// (workercompliance.Standing). A row that reported the column verbatim said a driver
	// whose medical card had lapsed four days earlier was dispatchable.
	CanBeAssigned     bool   `json:"canBeAssigned"`
	AssignmentBlocked string `json:"assignmentBlocked,omitempty"`
	// The compliance fields travel with the row because they are the reason
	// the row was asked for. A roster filtered on an endorsement that then
	// comes back without it leaves the reader to trust the filter blindly.
	Endorsement      string `json:"endorsement,omitempty"`
	CDLClass         string `json:"cdlClass,omitempty"`
	ComplianceStatus string `json:"complianceStatus,omitempty"`
	Qualified        *bool  `json:"qualified,omitempty"`
	// Always present, never omitted: a missing credential date is the finding,
	// and a key that disappears when nothing is on file reads as nothing to
	// worry about.
	HazmatExpiry      querytool.OptionalDate `json:"hazmatExpiry"`
	LicenseExpiry     querytool.OptionalDate `json:"licenseExpiry"`
	MedicalCardExpiry querytool.OptionalDate `json:"medicalCardExpiry"`
}

func NewListWorkersTool(
	repo workerports.WorkerRepository,
	permissions permissionports.PermissionEngine,
) agenttoolports.AgentQueryTool {
	return listtool.New(listtool.Spec{
		Name:         querytool.ToolWorkerList,
		EntityPlural: "workers",
		Summary:      workersSummary,
		SearchTerms:  workerSearchTerms,
		QueryNote:    workersQueryNote,
		Resource:     permission.ResourceWorker,
		Config:       listquery.Describe((*worker.Worker)(nil), workerdb.WorkersSchema()),
		Fields: []filtercatalog.Field{
			{
				Name:   "status",
				Kind:   filtercatalog.KindEnum,
				Values: querytool.StatusValues,
				Note:   "not availability; see canBeAssigned, worker_list_time_off",
			},
			{Name: "type", Kind: filtercatalog.KindEnum, Values: workerTypes},
			{Name: "driverType", Kind: filtercatalog.KindEnum, Values: driverTypes},
			{Name: "city", Kind: filtercatalog.KindText, Sortable: true},
			{Name: "lastName", Kind: filtercatalog.KindText, Sortable: true},
			{Name: "firstName", Kind: filtercatalog.KindText},
			{
				Name: "canBeAssigned",
				Kind: filtercatalog.KindBoolean,
				Note: "false = dispatch blocked",
			},
			{Name: "createdAt", Kind: filtercatalog.KindDate, Sortable: true},
			{
				Name:   "profile.endorsement",
				Kind:   filtercatalog.KindEnum,
				Values: endorsementCodes,
				Note:   endorsementNote,
			},
			{Name: "profile.cdlClass", Kind: filtercatalog.KindEnum, Values: cdlClasses},
			{
				Name:   "profile.complianceStatus",
				Kind:   filtercatalog.KindEnum,
				Values: complianceStates,
			},
			{
				Name: "profile.isQualified",
				Kind: filtercatalog.KindBoolean,
				Note: "false = lapsed or missing",
			},
			{Name: "profile.hazmatExpiry", Kind: filtercatalog.KindDate},
			{Name: "profile.licenseExpiry", Kind: filtercatalog.KindDate},
			{Name: "profile.medicalCardExpiry", Kind: filtercatalog.KindDate},
			{Name: "profile.twicExpiry", Kind: filtercatalog.KindDate},
		},
		Access:        fieldgate.NewAccess(permissions),
		GatedField:    workerGatedField,
		PagesByCursor: true,
		FetchGated: func(
			ctx context.Context,
			opts *pagination.QueryOptions,
			cursor pagination.CursorInfo,
			gate *fieldgate.Gate,
		) (listtool.Page, error) {
			result, err := repo.List(
				ctx,
				&workerports.ListWorkersRequest{Filter: opts, Cursor: cursor},
			)
			if err != nil {
				return listtool.Page{}, err
			}
			showCity := gate.Show(workerCityField, workerCityField)

			listed, err := listtool.CursorPage(result, func(item *worker.Worker) any {
				return toGatedWorkerRow(item, showCity)
			})
			if err != nil {
				return listtool.Page{}, err
			}
			if len(opts.Sort) == 0 {
				listed.Rows = rankWorkersByName(opts.Query, result.Items, listed.Rows)
			}

			return listed, nil
		},
	})
}

func rankWorkersByName(query string, items []*worker.Worker, rows []any) []any {
	if strings.TrimSpace(query) == "" || len(items) < 2 || len(items) != len(rows) {
		return rows
	}

	type scored struct {
		row   any
		score float64
	}
	ranked := make([]scored, 0, len(items))
	for idx, item := range items {
		ranked = append(ranked, scored{row: rows[idx], score: workerNameScore(query, item)})
	}
	slices.SortStableFunc(ranked, func(a, b scored) int {
		return cmp.Compare(b.score, a.score)
	})
	out := make([]any, 0, len(ranked))
	for idx := range ranked {
		out = append(out, ranked[idx].row)
	}

	return out
}

func workerNameScore(query string, item *worker.Worker) float64 {
	return namesScore(query, item.FirstName, item.LastName)
}

// namesScore is how closely a query matches a person's name, either way
// round.
func namesScore(query, first, last string) float64 {
	return max(
		textsim.NameSimilarity(query, strings.TrimSpace(first+" "+last)),
		textsim.NameSimilarity(query, last+" "+first),
	)
}

func workerGatedField(field string) string {
	return strings.TrimPrefix(field, workerProfilePath)
}

// applyProfile copies the qualification roll-up onto the row. A worker without a
// profile is a record mid-onboarding, not an error: the fields stay empty rather
// than reporting a lapsed licence nobody has yet entered.
// toWorkerRow is the one worker projection. The worker name search used to
// return the stored entity instead, which cost six times the bytes for the
// same answer — eight drivers overran the tool-result cap and were truncated
// mid-record — and reported an unrecorded credential as a bare null, which is
// the reading that caused the incident these rows exist to prevent.
func toWorkerRow(item *worker.Worker) workerRow {
	row := workerRow{
		ID:                item.ID.String(),
		Name:              querytool.WorkerName(item),
		Status:            string(item.Status),
		Type:              string(item.Type),
		DriverType:        string(item.DriverType),
		City:              item.City,
		CanBeAssigned:     item.CanBeAssigned,
		AssignmentBlocked: item.AssignmentBlocked,
	}
	if item.FleetCode != nil {
		row.FleetCode = item.FleetCode.Code
	}
	applyProfile(&row, item.Profile)

	return row
}

func toGatedWorkerRow(item *worker.Worker, showCity bool) workerRow {
	row := toWorkerRow(item)
	if !showCity {
		row.City = ""
	}

	return row
}

// dispatchable folds a worker's credential compliance into the stored
// dispatch flag, as workercompliance.Standing does: a NonCompliant worker cannot be
// assigned whatever the column says.
func dispatchable(
	canBeAssigned bool,
	blocked string,
	compliance worker.ComplianceStatus,
) (assignable bool, reason string) {
	if compliance != worker.ComplianceStatusNonCompliant {
		return canBeAssigned, blocked
	}
	if blocked == "" {
		blocked = "Credentials are not compliant"
	}

	return false, blocked
}

func applyProfile(row *workerRow, profile *worker.WorkerProfile) {
	if profile == nil {
		return
	}

	row.Endorsement = string(profile.Endorsement)
	row.CDLClass = string(profile.CDLClass)
	row.ComplianceStatus = string(profile.ComplianceStatus)
	row.Qualified = &profile.IsQualified

	// The same rule workercompliance.Standing applies, at the only point this projection
	// can apply it. Standing also weighs training, safety and checklists, which
	// a list row does not load — so this can still be too generous, never too
	// strict, and a caller deciding whether to dispatch should open the record.
	row.CanBeAssigned, row.AssignmentBlocked = dispatchable(
		row.CanBeAssigned, row.AssignmentBlocked, profile.ComplianceStatus,
	)

	row.HazmatExpiry = querytool.PointerDate(profile.HazmatExpiry)
	row.LicenseExpiry = querytool.RecordedDate(profile.LicenseExpiry)
	row.MedicalCardExpiry = querytool.PointerDate(profile.MedicalCardExpiry)
}
