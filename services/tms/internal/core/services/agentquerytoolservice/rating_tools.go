package agentquerytoolservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/fuelsurcharge"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/domain/rategeo"
	"github.com/emoss08/trenova/internal/core/domain/ratematrix"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/fuelsurchargeservice"
	"github.com/emoss08/trenova/internal/core/services/rateagreementservice"
	"github.com/emoss08/trenova/internal/core/services/ratematrixservice"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const (
	maxAgreementRules        = 50
	maxAgreementBreaks       = 20
	maxAgreementAccessorials = 50
	maxMatrixCells           = 200
	matrixCellPage           = 100
	maxFuelPrograms          = 50
)

var (
	rateAgreementStatuses = []string{
		string(rateagreement.StatusDraft),
		string(rateagreement.StatusInReview),
		string(rateagreement.StatusActive),
		string(rateagreement.StatusSuspended),
		string(rateagreement.StatusExpired),
		string(rateagreement.StatusArchived),
	}
	rateAgreementParties = []string{
		string(rateagreement.PartyTypeCustomer),
		string(rateagreement.PartyTypeCarrier),
	}
)

func ratingToolProviders() []any {
	return []any{
		provideListRateAgreementsTool,
		provideGetRateAgreementTool,
		provideGetRateMatrixTool,
		provideGetFuelSurchargeRatesTool,
	}
}

type rateAgreementReader interface {
	List(
		ctx context.Context,
		req *repositories.ListRateAgreementRequest,
	) (*pagination.ListResult[*rateagreement.RateAgreement], error)
	GetByID(
		ctx context.Context,
		req *repositories.GetRateAgreementByIDRequest,
	) (*rateagreement.RateAgreement, error)
	ListVersions(
		ctx context.Context,
		req *repositories.ListRateAgreementVersionsRequest,
	) (*pagination.ListResult[*rateagreement.RateAgreementVersion], error)
}

type rateMatrixReader interface {
	GetByID(
		ctx context.Context,
		req *repositories.GetRateMatrixByIDRequest,
	) (*ratematrix.RateMatrix, error)
	ListCells(
		ctx context.Context,
		req *repositories.ListRateMatrixCellsRequest,
	) (*pagination.ListResult[*ratematrix.RateMatrixCell], error)
}

type fuelRateReader interface {
	ProgramCurrentRates(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) ([]*fuelsurchargeservice.ProgramCurrentRate, error)
}

func provideListRateAgreementsTool(
	agreements *rateagreementservice.Service,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return newListRateAgreementsTool(agreements, permissions)
}

func provideGetRateAgreementTool(
	agreements *rateagreementservice.Service,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return newGetRateAgreementTool(agreements, permissions)
}

func provideGetRateMatrixTool(
	matrices *ratematrixservice.Service,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return newGetRateMatrixTool(matrices, permissions)
}

func provideGetFuelSurchargeRatesTool(
	fuel *fuelsurchargeservice.Service,
) serviceports.AgentQueryTool {
	return newGetFuelSurchargeRatesTool(fuel)
}

func nullDecimalText(value decimal.NullDecimal) string {
	if !value.Valid {
		return ""
	}

	return value.Decimal.String()
}

func decimalPointerText(value *decimal.Decimal) string {
	if value == nil {
		return ""
	}

	return value.String()
}

func laneEnd(scopeType rategeo.ScopeType, value, city string) string {
	parts := make([]string, 0, 3)
	if scopeType != "" {
		parts = append(parts, string(scopeType))
	}
	if city != "" {
		parts = append(parts, city)
	}
	if value != "" && value != city {
		parts = append(parts, value)
	}

	return strings.Join(parts, " ")
}

type rateAgreementRow struct {
	ID            string       `json:"id"`
	Code          string       `json:"code"`
	Name          string       `json:"name"`
	PartyType     string       `json:"partyType"`
	CustomerID    string       `json:"customerId,omitempty"`
	CarrierID     string       `json:"carrierId,omitempty"`
	Party         string       `json:"party,omitempty"`
	AgreementType string       `json:"agreementType"`
	Status        string       `json:"status"`
	Priority      int16        `json:"priority"`
	ContractRef   string       `json:"contractRef,omitempty"`
	EffectiveFrom optionalDate `json:"effectiveFrom"`
	EffectiveTo   optionalDate `json:"effectiveTo"`
	AutoRenew     bool         `json:"autoRenew"`
	Currency      string       `json:"currency"`
	MinCharge     string       `json:"defaultMinCharge,omitempty"`
	MaxCharge     string       `json:"defaultMaxCharge,omitempty"`
}

func rateAgreementRowFrom(entity *rateagreement.RateAgreement, gate *fieldGate) rateAgreementRow {
	row := rateAgreementRow{
		ID:            entity.ID.String(),
		Code:          entity.Code,
		Name:          entity.Name,
		PartyType:     string(entity.PartyType),
		CustomerID:    pointerIDString(entity.CustomerID),
		CarrierID:     pointerIDString(entity.CarrierID),
		AgreementType: string(entity.AgreementType),
		Status:        string(entity.Status),
		Priority:      entity.Priority,
		ContractRef:   entity.ContractRef,
		EffectiveFrom: recordedDate(entity.EffectiveFrom),
		EffectiveTo:   expectedDate(derefInt64(entity.EffectiveTo), "open-ended"),
		AutoRenew:     entity.AutoRenew,
		Currency:      entity.Currency,
	}
	switch {
	case entity.Customer != nil:
		row.Party = entity.Customer.Name
	case entity.Carrier != nil:
		row.Party = entity.Carrier.Name
	}
	if entity.DefaultMinCharge.Valid && gate.show("defaultMinCharge", "defaultMinCharge") {
		row.MinCharge = nullDecimalText(entity.DefaultMinCharge)
	}
	if entity.DefaultMaxCharge.Valid && gate.show("defaultMaxCharge", "defaultMaxCharge") {
		row.MaxCharge = nullDecimalText(entity.DefaultMaxCharge)
	}

	return row
}

type listRateAgreementsTool struct {
	agreements rateAgreementReader
	access     fieldAccess
}

func newListRateAgreementsTool(
	agreements rateAgreementReader,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return &listRateAgreementsTool{agreements: agreements, access: newFieldAccess(permissions)}
}

func (t *listRateAgreementsTool) Name() string { return "list_rate_agreements" }

func (t *listRateAgreementsTool) Description() string {
	return "List rate agreements, the contracts and tariffs that price freight for a " +
		"customer or pay a carrier. Each has its code, name, party, type, status and " +
		"effective dates. Narrow to a customer, a carrier, a status or text; " +
		"get_rate_agreement opens one with its lane rules."
}

func (t *listRateAgreementsTool) ParamSchema() map[string]any {
	return objectSchema(withPaging(map[string]any{
		"query": stringParam("Words to look for in the agreement's code or name."),
		"partyType": enumParam("Customer agreements price what a customer is billed; "+
			"Carrier agreements price what a carrier is paid.", rateAgreementParties),
		"customerId": stringParam("Only this customer's agreements, by id from " +
			"list_customers."),
		"carrierId": stringParam("Only this carrier's agreements, by id from list_carriers."),
		"status":    enumParam("Only agreements in this status.", rateAgreementStatuses),
	}, defaultListLimit, maxListLimit))
}

func (t *listRateAgreementsTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceRateAgreement})
}

func (t *listRateAgreementsTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	party, err := validEnum(params.Params, "partyType", rateAgreementParties)
	if err != nil {
		return nil, err
	}
	status, err := validEnum(params.Params, "status", rateAgreementStatuses)
	if err != nil {
		return nil, err
	}
	customerID, err := optionalID(params.Params, "customerId")
	if err != nil {
		return nil, err
	}
	carrierID, err := optionalID(params.Params, "carrierId")
	if err != nil {
		return nil, err
	}
	window := readPage(params.Params, defaultListLimit, maxListLimit)

	req := &repositories.ListRateAgreementRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: tenantOf(params),
			Pagination: pagination.Info{Limit: window.fetch(), Offset: window.offset},
			Query:      optionalString(params.Params, "query"),
		},
		PartyType: rateagreement.PartyType(party),
		Status:    rateagreement.Status(status),
	}

	criteria := filtercatalog.NewCriteria("rate agreements").At(clockFor(params))
	criteria.Text(req.Filter.Query)
	if party != "" {
		criteria.Field("party type", party)
	}
	if status != "" {
		criteria.Field("status", status)
	}
	if customerID.IsNotNil() {
		req.CustomerID = &customerID
		criteria.Field("customer", customerID.String())
	}
	if carrierID.IsNotNil() {
		req.CarrierID = &carrierID
		criteria.Field("carrier", carrierID.String())
	}

	result, err := t.agreements.List(ctx, req)
	if err != nil {
		return nil, err
	}

	gate := t.access.gate(ctx, params, permission.ResourceRateAgreement)
	rows := make([]rateAgreementRow, 0, len(result.Items))
	for _, entity := range result.Items {
		if entity != nil {
			rows = append(rows, rateAgreementRowFrom(entity, gate))
		}
	}
	rows, more := trim(window, rows)

	return gatedResult(searchResult(criteria, rows, len(rows)).paged(window, more), gate), nil
}

type getRateAgreementTool struct {
	agreements rateAgreementReader
	access     fieldAccess
}

func newGetRateAgreementTool(
	agreements rateAgreementReader,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return &getRateAgreementTool{agreements: agreements, access: newFieldAccess(permissions)}
}

func (t *getRateAgreementTool) Name() string { return "get_rate_agreement" }

func (t *getRateAgreementTool) Description() string {
	return "Retrieve one rate agreement, a contract or tariff, by id with its lane rules, " +
		"rates, minimums and weight breaks. It also gives the current version, the " +
		"accessorial schedule and the fuel surcharge binding. explain_rate says how it " +
		"priced one shipment. Take the id from list_rate_agreements."
}

func (t *getRateAgreementTool) ParamSchema() map[string]any {
	return idSchema("rateAgreementId", "The rate agreement's id, from list_rate_agreements, "+
		"explain_rate or the page you are on.")
}

func (t *getRateAgreementTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceRateAgreement})
}

type rateBreakRow struct {
	FromWeight string `json:"fromWeight"`
	ToWeight   string `json:"toWeight,omitempty"`
	Label      string `json:"label,omitempty"`
	Rate       string `json:"rate,omitempty"`
	MinCharge  string `json:"minCharge,omitempty"`
}

type rateRuleRow struct {
	ID            string         `json:"id"`
	Label         string         `json:"label,omitempty"`
	Origin        string         `json:"origin"`
	Destination   string         `json:"destination"`
	Direction     string         `json:"direction"`
	Priority      int16          `json:"priority"`
	EffectiveFrom optionalDate   `json:"effectiveFrom"`
	EffectiveTo   optionalDate   `json:"effectiveTo"`
	HazmatOnly    bool           `json:"hazmatOnly,omitempty"`
	TempOnly      bool           `json:"tempControlOnly,omitempty"`
	FormulaID     string         `json:"formulaTemplateId,omitempty"`
	RateMatrixID  string         `json:"rateMatrixId,omitempty"`
	Currency      string         `json:"currency,omitempty"`
	Rate          string         `json:"rate,omitempty"`
	MinCharge     string         `json:"minCharge,omitempty"`
	MaxCharge     string         `json:"maxCharge,omitempty"`
	Discount      string         `json:"discountPercent,omitempty"`
	Breaks        []rateBreakRow `json:"weightBreaks,omitempty"`
	BreaksOmitted int            `json:"weightBreaksOmitted,omitempty"`
}

type rateAccessorialRow struct {
	AccessorialChargeID string       `json:"accessorialChargeId"`
	Method              string       `json:"method"`
	RateUnit            string       `json:"rateUnit,omitempty"`
	Waived              bool         `json:"waived"`
	AutoApply           bool         `json:"autoApply"`
	FreeUnits           *int16       `json:"freeUnits,omitempty"`
	EffectiveFrom       optionalDate `json:"effectiveFrom"`
	EffectiveTo         optionalDate `json:"effectiveTo"`
	Amount              string       `json:"amount,omitempty"`
	MaxAmount           string       `json:"maxAmount,omitempty"`
	ApplyCondition      string       `json:"applyCondition,omitempty"`
}

type rateFuelRow struct {
	FuelSurchargeProgramID string `json:"fuelSurchargeProgramId"`
	Waived                 bool   `json:"waived"`
}

type rateVersionRow struct {
	VersionNumber int64        `json:"versionNumber"`
	EffectiveFrom optionalDate `json:"effectiveFrom"`
	EffectiveTo   optionalDate `json:"effectiveTo"`
	ChangeMessage string       `json:"changeMessage,omitempty"`
}

type rateAgreementView struct {
	rateAgreementRow

	Description         string               `json:"description,omitempty"`
	BillToCustomerID    string               `json:"billToCustomerId,omitempty"`
	RenewalNoticeDays   int16                `json:"renewalNoticeDays"`
	RoundingMode        string               `json:"roundingMode"`
	SubmittedAt         optionalDate         `json:"submittedAt"`
	ApprovedAt          optionalDate         `json:"approvedAt"`
	ReviewComment       string               `json:"reviewComment,omitempty"`
	CurrentVersion      *rateVersionRow      `json:"currentVersion,omitempty"`
	RuleCount           int                  `json:"ruleCount"`
	Rules               []rateRuleRow        `json:"rules"`
	RulesOmitted        int                  `json:"rulesOmitted,omitempty"`
	AccessorialCount    int                  `json:"accessorialCount"`
	Accessorials        []rateAccessorialRow `json:"accessorials"`
	AccessorialsOmitted int                  `json:"accessorialsOmitted,omitempty"`
	Fuel                *rateFuelRow         `json:"fuel,omitempty"`
	Withheld            []string             `json:"withheldByAccess,omitempty"`
}

func (t *getRateAgreementTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	id, err := requirePulid(params.Params, "rateAgreementId")
	if err != nil {
		return nil, err
	}

	tenant := tenantOf(params)
	entity, err := t.agreements.GetByID(ctx, &repositories.GetRateAgreementByIDRequest{
		RateAgreementID: id,
		TenantInfo:      tenant,
		IncludeChildren: true,
		AsOf:            clockFor(params).Instant(),
	})
	if err != nil {
		return nil, err
	}

	versions, err := t.agreements.ListVersions(ctx, &repositories.ListRateAgreementVersionsRequest{
		TenantInfo:      tenant,
		RateAgreementID: id,
		Limit:           1,
	})
	if err != nil {
		return nil, err
	}

	gate := t.access.gate(ctx, params, permission.ResourceRateAgreement)
	view := rateAgreementView{
		rateAgreementRow:  rateAgreementRowFrom(entity, gate),
		Description:       entity.Description,
		BillToCustomerID:  pointerIDString(entity.BillToCustomerID),
		RenewalNoticeDays: entity.RenewalNoticeDays,
		RoundingMode:      string(entity.RoundingMode),
		SubmittedAt:       expectedDate(derefInt64(entity.SubmittedAt), "not submitted"),
		ApprovedAt:        expectedDate(derefInt64(entity.ApprovedAt), "not approved"),
		RuleCount:         len(entity.Rules),
		AccessorialCount:  len(entity.Accessorials),
	}
	if entity.ReviewComment != "" && gate.show("reviewComment", "reviewComment") {
		view.ReviewComment = entity.ReviewComment
	}
	if len(versions.Items) > 0 && versions.Items[0] != nil {
		version := versions.Items[0]
		view.CurrentVersion = &rateVersionRow{
			VersionNumber: version.VersionNumber,
			EffectiveFrom: recordedDate(version.EffectiveFrom),
			EffectiveTo:   expectedDate(derefInt64(version.EffectiveTo), "current"),
		}
		if version.ChangeMessage != "" && gate.show("changeMessage", "changeMessage") {
			view.CurrentVersion.ChangeMessage = version.ChangeMessage
		}
	}

	view.Rules = t.rules(entity.Rules, gate)
	view.RulesOmitted = max(len(entity.Rules)-maxAgreementRules, 0)
	view.Accessorials = t.accessorials(entity.Accessorials, gate)
	view.AccessorialsOmitted = max(len(entity.Accessorials)-maxAgreementAccessorials, 0)
	if entity.FuelBinding != nil {
		view.Fuel = &rateFuelRow{
			FuelSurchargeProgramID: entity.FuelBinding.FuelSurchargeProgramID.String(),
			Waived:                 entity.FuelBinding.Waived,
		}
	}
	view.Withheld = gate.Withheld()

	return view, nil
}

func (t *getRateAgreementTool) rules(
	rules []*rateagreement.RateAgreementRule,
	gate *fieldGate,
) []rateRuleRow {
	showRate := gate.show("rate", "rules.rate")
	showMin := gate.show("minCharge", "rules.minCharge")
	showMax := gate.show("maxCharge", "rules.maxCharge")
	showDiscount := gate.show("discountPercent", "rules.discountPercent")

	rows := make([]rateRuleRow, 0, min(len(rules), maxAgreementRules))
	for _, rule := range rules {
		if rule == nil {
			continue
		}
		if len(rows) == maxAgreementRules {
			break
		}
		row := rateRuleRow{
			ID:    rule.ID.String(),
			Label: rule.Label,
			Origin: laneEnd(
				rule.OriginScopeType, rule.OriginScopeValue, rule.OriginCity,
			),
			Destination: laneEnd(
				rule.DestinationScopeType, rule.DestinationScopeValue, rule.DestinationCity,
			),
			Direction:     string(rule.Direction),
			Priority:      rule.Priority,
			EffectiveFrom: recordedDate(rule.EffectiveFrom),
			EffectiveTo:   expectedDate(derefInt64(rule.EffectiveTo), "open-ended"),
			HazmatOnly:    rule.HazmatOnly,
			TempOnly:      rule.TempControlOnly,
			FormulaID:     pointerIDString(rule.FormulaTemplateID),
			RateMatrixID:  pointerIDString(rule.RateMatrixID),
			Currency:      rule.Currency,
		}
		if showRate {
			row.Rate = nullDecimalText(rule.Rate)
		}
		if showMin {
			row.MinCharge = nullDecimalText(rule.MinCharge)
		}
		if showMax {
			row.MaxCharge = nullDecimalText(rule.MaxCharge)
		}
		if showDiscount {
			row.Discount = nullDecimalText(rule.DiscountPercent)
		}
		for _, weightBreak := range rule.Breaks {
			if weightBreak == nil {
				continue
			}
			if len(row.Breaks) == maxAgreementBreaks {
				row.BreaksOmitted++
				continue
			}
			entry := rateBreakRow{
				FromWeight: weightBreak.FromWeight.String(),
				ToWeight:   nullDecimalText(weightBreak.ToWeight),
				Label:      weightBreak.Label,
			}
			if showRate {
				entry.Rate = weightBreak.Rate.String()
			}
			if showMin {
				entry.MinCharge = nullDecimalText(weightBreak.MinCharge)
			}
			row.Breaks = append(row.Breaks, entry)
		}
		rows = append(rows, row)
	}

	return rows
}

func (t *getRateAgreementTool) accessorials(
	accessorials []*rateagreement.RateAgreementAccessorial,
	gate *fieldGate,
) []rateAccessorialRow {
	showAmount := gate.show("amount", "accessorials.amount")
	showMax := gate.show("maxAmount", "accessorials.maxAmount")
	showCondition := gate.show("applyCondition", "accessorials.applyCondition")

	rows := make([]rateAccessorialRow, 0, min(len(accessorials), maxAgreementAccessorials))
	for _, accessorial := range accessorials {
		if accessorial == nil {
			continue
		}
		if len(rows) == maxAgreementAccessorials {
			break
		}
		row := rateAccessorialRow{
			AccessorialChargeID: accessorial.AccessorialChargeID.String(),
			Method:              string(accessorial.Method),
			RateUnit:            string(accessorial.RateUnit),
			Waived:              accessorial.Waived,
			AutoApply:           accessorial.AutoApply,
			FreeUnits:           accessorial.FreeUnits,
			EffectiveFrom:       pointerDate(accessorial.EffectiveFrom),
			EffectiveTo:         expectedDate(derefInt64(accessorial.EffectiveTo), "open-ended"),
		}
		if showAmount {
			row.Amount = accessorial.Amount.String()
		}
		if showMax {
			row.MaxAmount = nullDecimalText(accessorial.MaxAmount)
		}
		if showCondition {
			row.ApplyCondition = accessorial.ApplyCondition
		}
		rows = append(rows, row)
	}

	return rows
}

type getRateMatrixTool struct {
	matrices rateMatrixReader
	access   fieldAccess
}

func newGetRateMatrixTool(
	matrices rateMatrixReader,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return &getRateMatrixTool{matrices: matrices, access: newFieldAccess(permissions)}
}

func (t *getRateMatrixTool) Name() string { return "get_rate_matrix" }

func (t *getRateMatrixTool) Description() string {
	return "Retrieve one rate matrix by id: the grid of rates a lane rule prices from. " +
		"It gives the axes (zone, zip, weight break, freight class and so on) and up to " +
		"200 cells with their keys, ranges and values. Take the id from a rule's " +
		"rateMatrixId in get_rate_agreement."
}

func (t *getRateMatrixTool) ParamSchema() map[string]any {
	return objectSchema(map[string]any{
		"rateMatrixId": stringParam("The rate matrix's id, from a rule's rateMatrixId in " +
			"get_rate_agreement or the page you are on."),
		"cellOffset": intParam("How many cells to skip. When a result says cellsHasMore, " +
			"call again with its nextCellOffset."),
	}, "rateMatrixId")
}

func (t *getRateMatrixTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceRateMatrix})
}

type matrixDimensionRow struct {
	Position         int16  `json:"position"`
	Kind             string `json:"kind"`
	MatchMode        string `json:"matchMode"`
	Label            string `json:"label,omitempty"`
	KeyNormalization string `json:"keyNormalization,omitempty"`
	RangeOverflow    string `json:"rangeOverflow,omitempty"`
}

type matrixCellRow struct {
	Keys            []string `json:"keys,omitempty"`
	Ranges          []string `json:"ranges,omitempty"`
	DeficitEligible bool     `json:"deficitEligible,omitempty"`
	Value           string   `json:"value,omitempty"`
	MinCharge       string   `json:"minCharge,omitempty"`
}

type rateMatrixView struct {
	ID                string               `json:"id"`
	Code              string               `json:"code"`
	Name              string               `json:"name"`
	Description       string               `json:"description,omitempty"`
	Status            string               `json:"status"`
	Currency          string               `json:"currency"`
	FormulaTemplateID string               `json:"formulaTemplateId"`
	RoundingMode      string               `json:"roundingMode"`
	Dimensions        []matrixDimensionRow `json:"dimensions"`
	CellCount         int                  `json:"cellCount"`
	CellOffset        int                  `json:"cellOffset,omitempty"`
	Cells             []matrixCellRow      `json:"cells"`
	CellsHasMore      bool                 `json:"cellsHasMore"`
	NextCellOffset    *int                 `json:"nextCellOffset,omitempty"`
	Withheld          []string             `json:"withheldByAccess,omitempty"`
}

func (t *getRateMatrixTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	id, err := requirePulid(params.Params, "rateMatrixId")
	if err != nil {
		return nil, err
	}
	offset := max(optionalInt(params.Params, "cellOffset", 0), 0)

	tenant := tenantOf(params)
	matrix, err := t.matrices.GetByID(ctx, &repositories.GetRateMatrixByIDRequest{
		RateMatrixID:      id,
		TenantInfo:        tenant,
		IncludeDimensions: true,
	})
	if err != nil {
		return nil, err
	}

	cells, total, err := t.cells(ctx, tenant, id, offset)
	if err != nil {
		return nil, err
	}

	gate := t.access.gate(ctx, params, permission.ResourceRateMatrix)
	view := rateMatrixView{
		ID:                matrix.ID.String(),
		Code:              matrix.Code,
		Name:              matrix.Name,
		Description:       matrix.Description,
		Status:            string(matrix.Status),
		Currency:          matrix.Currency,
		FormulaTemplateID: pulidString(matrix.FormulaTemplateID),
		RoundingMode:      string(matrix.RoundingMode),
		Dimensions:        make([]matrixDimensionRow, 0, len(matrix.Dimensions)),
		CellCount:         total,
		CellOffset:        offset,
		Cells:             make([]matrixCellRow, 0, len(cells)),
	}
	axes := 0
	for _, dimension := range matrix.Dimensions {
		if dimension == nil {
			continue
		}
		axes = max(axes, int(dimension.Position)+1)
		view.Dimensions = append(view.Dimensions, matrixDimensionRow{
			Position:         dimension.Position,
			Kind:             string(dimension.Kind),
			MatchMode:        string(dimension.MatchMode),
			Label:            dimension.Label,
			KeyNormalization: string(dimension.KeyNormalization),
			RangeOverflow:    string(dimension.RangeOverflow),
		})
	}

	showValue := gate.show("value", "cells.value")
	showMin := gate.show("minCharge", "cells.minCharge")
	for _, cell := range cells {
		if cell == nil {
			continue
		}
		row := matrixCellRow{
			Keys:            cellKeys(cell, axes),
			Ranges:          cellRanges(cell, axes),
			DeficitEligible: cell.DeficitEligible,
		}
		if showValue {
			row.Value = cell.Value.String()
		}
		if showMin {
			row.MinCharge = nullDecimalText(cell.MinCharge)
		}
		view.Cells = append(view.Cells, row)
	}
	if offset+len(view.Cells) < total {
		view.CellsHasMore = true
		next := offset + len(view.Cells)
		view.NextCellOffset = &next
	}
	view.Withheld = gate.Withheld()

	return view, nil
}

func (t *getRateMatrixTool) cells(
	ctx context.Context,
	tenant pagination.TenantInfo,
	matrixID pulid.ID,
	offset int,
) ([]*ratematrix.RateMatrixCell, int, error) {
	cells := make([]*ratematrix.RateMatrixCell, 0, matrixCellPage)
	total := 0
	for len(cells) < maxMatrixCells {
		page, err := t.matrices.ListCells(ctx, &repositories.ListRateMatrixCellsRequest{
			TenantInfo:   tenant,
			RateMatrixID: matrixID,
			Filter: &pagination.QueryOptions{
				TenantInfo: tenant,
				Pagination: pagination.Info{
					Limit:  min(matrixCellPage, maxMatrixCells-len(cells)),
					Offset: offset + len(cells),
				},
			},
		})
		if err != nil {
			return nil, 0, err
		}
		total = page.Total
		cells = append(cells, page.Items...)
		if len(page.Items) < matrixCellPage || offset+len(cells) >= total {
			break
		}
	}

	return cells, total, nil
}

func cellKeys(cell *ratematrix.RateMatrixCell, axes int) []string {
	keys := []string{cell.D0Key, cell.D1Key, cell.D2Key, cell.D3Key}[:min(max(axes, 0), 4)]
	for _, key := range keys {
		if key != "" {
			return keys
		}
	}

	return nil
}

func cellRanges(cell *ratematrix.RateMatrixCell, axes int) []string {
	bounds := [][2]decimal.NullDecimal{
		{cell.D0Min, cell.D0Max},
		{cell.D1Min, cell.D1Max},
		{cell.D2Min, cell.D2Max},
		{cell.D3Min, cell.D3Max},
	}[:min(max(axes, 0), 4)]

	ranges := make([]string, 0, len(bounds))
	found := false
	for _, bound := range bounds {
		low, high := nullDecimalText(bound[0]), nullDecimalText(bound[1])
		if low == "" && high == "" {
			ranges = append(ranges, "")
			continue
		}
		found = true
		if high == "" {
			high = "and up"
		}
		ranges = append(ranges, low+" to "+high)
	}
	if !found {
		return nil
	}

	return ranges
}

type getFuelSurchargeRatesTool struct {
	fuel fuelRateReader
}

func newGetFuelSurchargeRatesTool(fuel fuelRateReader) serviceports.AgentQueryTool {
	return &getFuelSurchargeRatesTool{fuel: fuel}
}

func (t *getFuelSurchargeRatesTool) Name() string { return "get_fuel_surcharge_rates" }

func (t *getFuelSurchargeRatesTool) Description() string {
	return "Read this week's fuel surcharge rate for every active fuel program. It gives " +
		"the fuel index price and its date, and the surcharge as a rate per mile, a " +
		"percent of linehaul or a flat amount. It says when a fallback price was used " +
		"because this week's was missing."
}

func (t *getFuelSurchargeRatesTool) ParamSchema() map[string]any {
	return objectSchema(map[string]any{
		"query": stringParam("Words to look for in the program's code or name. Omit for " +
			"every active program."),
	})
}

func (t *getFuelSurchargeRatesTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceFuelSurchargeProgram})
}

type fuelRateRow struct {
	ProgramID    string `json:"programId"`
	Code         string `json:"code"`
	Name         string `json:"name"`
	Method       string `json:"method"`
	FuelIndexID  string `json:"fuelIndexId"`
	FuelIndex    string `json:"fuelIndex,omitempty"`
	PriceDate    string `json:"priceDate,omitempty"`
	IndexPrice   string `json:"indexPrice,omitempty"`
	RatePerMile  string `json:"ratePerMile,omitempty"`
	Percent      string `json:"percent,omitempty"`
	FlatAmount   string `json:"flatAmount,omitempty"`
	UsedFallback bool   `json:"usedFallback"`
	Note         string `json:"note,omitempty"`
}

func fuelRateRowFrom(entry *fuelsurchargeservice.ProgramCurrentRate) fuelRateRow {
	program := entry.Program
	row := fuelRateRow{
		ProgramID:    program.ID.String(),
		Code:         program.Code,
		Name:         program.Name,
		Method:       string(program.Method),
		FuelIndexID:  program.FuelIndexID.String(),
		RatePerMile:  decimalPointerText(entry.RatePerMile),
		Percent:      decimalPointerText(entry.Percent),
		FlatAmount:   decimalPointerText(entry.FlatAmount),
		UsedFallback: entry.UsedFallback,
	}
	if program.FuelIndex != nil {
		row.FuelIndex = program.FuelIndex.Name
	}
	if entry.Price != nil {
		row.PriceDate = entry.Price.PriceDate
		row.IndexPrice = entry.Price.Price.String()
	}
	switch {
	case entry.Price == nil:
		row.Note = "No index price is on file for this week or the fallback window, so " +
			"no surcharge can be computed."
	case entry.RatePerMile == nil && entry.Percent == nil && entry.FlatAmount == nil:
		row.Note = "The index price falls outside every row of this program's table."
	}

	return row
}

func (t *getFuelSurchargeRatesTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	query := strings.ToLower(optionalString(params.Params, "query"))
	criteria := filtercatalog.NewCriteria("active fuel surcharge programs").At(clockFor(params))
	criteria.Text(query)

	rates, err := t.fuel.ProgramCurrentRates(ctx, tenantOf(params))
	if err != nil {
		return nil, err
	}

	rows := make([]fuelRateRow, 0, min(len(rates), maxFuelPrograms))
	matched := 0
	for _, entry := range rates {
		if entry == nil || entry.Program == nil || !programMatches(entry.Program, query) {
			continue
		}
		matched++
		if len(rows) < maxFuelPrograms {
			rows = append(rows, fuelRateRowFrom(entry))
		}
	}

	return searchResult(criteria, rows, matched), nil
}

func programMatches(program *fuelsurcharge.FuelSurchargeProgram, query string) bool {
	if query == "" {
		return true
	}

	return strings.Contains(strings.ToLower(program.Code), query) ||
		strings.Contains(strings.ToLower(program.Name), query)
}
