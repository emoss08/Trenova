package agentquerytoolservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workercredentialservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/typeutils"
)

const (
	toolWorkerListCredentials = "worker_list_credentials"
	defaultExpiryHorizonDays  = 30
	maxExpiryHorizonDays      = 365
	// expiredGraceDays is how far back "already expired" reaches. A card that
	// lapsed last quarter is still the fleet's problem, and someone asking who
	// is out of date means them too.
	expiredGraceDays        = 90
	maxCredentialRows       = 100
	paramWithinDays         = "withinDays"
	paramIncludeExpired     = "includeExpired"
	paramCredentialTypeCode = "credentialTypeCode" //nolint:gosec // G101: a parameter name, not a credential
	paramRequiredOnly       = "requiredOnly"
	credentialEntity        = "credentials"
)

// expiringCredentialRow is one credential in the words someone chasing renewals
// would use.
//
// The stored entity carries the worker's whole profile, the scanned document
// and the user who verified it. None of that helps answer who needs a new
// medical card, and all of it costs context the model needs for the answer.
type expiringCredentialRow struct {
	ID               string       `json:"id"`
	WorkerID         string       `json:"workerId"`
	WorkerName       string       `json:"workerName"`
	CredentialTypeID string       `json:"credentialTypeId"`
	CredentialType   string       `json:"credentialType"`
	CredentialCode   string       `json:"credentialCode"`
	ExpiresAt        optionalDate `json:"expiresAt"`
	DaysUntilExpiry  int64        `json:"daysUntilExpiry"`
	// Expired is stated rather than left to be derived from a negative day
	// count, because the difference between "renew this" and "this driver
	// should not be dispatched" is the whole point of the question.
	Expired bool `json:"expired"`
}

type credentialRow struct {
	ID               string       `json:"id"`
	CredentialTypeID string       `json:"credentialTypeId"`
	CredentialType   string       `json:"credentialType,omitempty"`
	CredentialCode   string       `json:"credentialCode,omitempty"`
	Status           string       `json:"status"`
	Number           string       `json:"number,omitempty"`
	IssuingAuthority string       `json:"issuingAuthority,omitempty"`
	IssuedAt         optionalDate `json:"issuedAt"`
	ExpiresAt        optionalDate `json:"expiresAt"`
	Verified         bool         `json:"verified"`
	HasDocument      bool         `json:"hasDocument"`
}

type credentialTypeRow struct {
	ID         string `json:"id"`
	Code       string `json:"code"`
	Name       string `json:"name"`
	Category   string `json:"category"`
	IsRequired bool   `json:"isRequired"`
}

type workerCredentialsOutcome struct {
	gatedOutcome

	Types []credentialTypeRow `json:"credentialTypes,omitempty"`
}

type credentialExpiryReader interface {
	ListExpiring(
		ctx context.Context,
		req *workerports.ListExpiringWorkerCredentialsRequest,
	) ([]*worker.WorkerCredential, error)
}

type credentialFilter struct {
	workerID       pulid.ID
	byExpiry       bool
	horizon        int
	includeExpired bool
	codes          []string
	requiredOnly   bool
}

func readCredentialFilter(params map[string]any) (credentialFilter, error) {
	workerID, err := optionalWorker(params)
	if err != nil {
		return credentialFilter{}, err
	}

	requested := optionalInt(params, paramWithinDays, 0)
	horizon := defaultExpiryHorizonDays
	if requested > 0 {
		horizon = min(requested, maxExpiryHorizonDays)
	}
	includeExpired := optionalBool(params, paramIncludeExpired)

	// Codes are stored upper-case; a model writing "med_card" means the same
	// credential and should not get an empty list for the difference.
	var codes []string
	if code := strings.TrimSpace(optionalString(params, paramCredentialTypeCode)); code != "" {
		codes = []string{strings.ToUpper(code)}
	}

	return credentialFilter{
		workerID:       workerID,
		byExpiry:       workerID.IsNil() || requested > 0 || includeExpired,
		horizon:        horizon,
		includeExpired: includeExpired,
		codes:          codes,
		requiredOnly:   optionalBool(params, paramRequiredOnly),
	}, nil
}

func (f credentialFilter) admits(credential *worker.WorkerCredential) bool {
	kind := credential.CredentialType
	if f.requiredOnly && (kind == nil || !kind.IsRequired) {
		return false
	}
	if len(f.codes) > 0 && (kind == nil || !strings.EqualFold(kind.Code, f.codes[0])) {
		return false
	}

	return true
}

func (f credentialFilter) describe(criteria *filtercatalog.Criteria) {
	if !f.workerID.IsNil() {
		criteria.Field("worker", f.workerID.String())
	}
	if f.byExpiry {
		criteria.Field("expiring within", fmt.Sprintf("%d days", f.horizon))
	}
	if len(f.codes) > 0 {
		criteria.Field("credential type", f.codes[0])
	}
	if f.includeExpired {
		criteria.Field("including", "already expired")
	}
	if f.requiredOnly {
		criteria.Field("limited to", "required credential types")
	}
}

type listWorkerCredentialsTool struct {
	credentials wfCredentialReader
	expiring    credentialExpiryReader
	access      fieldAccess
}

func newListWorkerCredentialsTool(
	credentials wfCredentialReader,
	expiring credentialExpiryReader,
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return &listWorkerCredentialsTool{
		credentials: credentials,
		expiring:    expiring,
		access:      newFieldAccess(permissions),
	}
}

func provideListWorkerCredentialsTool(
	s *workercredentialservice.Service,
	repo workerports.WorkerCredentialRepository,
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return newListWorkerCredentialsTool(s, repo, permissions)
}

func (t *listWorkerCredentialsTool) Name() string { return toolWorkerListCredentials }

func (t *listWorkerCredentialsTool) SearchTerms() []string {
	return []string{"expiring licenses", "medical card expiry", "credentials due", "lapsed"}
}

func (t *listWorkerCredentialsTool) Description() string {
	return "List worker (driver) credentials: one worker's licences, medical cards and " +
		"endorsements, or those falling due across the organization, soonest expiry first. " +
		"With workerId and no expiry filter it returns every credential that worker holds, " +
		"with its dates and whether it is verified, and the credential types the " +
		"organization tracks; a required type with nothing on file was never recorded, " +
		"which is worse than expiring soon. Without workerId, or with withinDays or " +
		"includeExpired, it lists active credentials expiring within withinDays, soonest " +
		"first, for one worker when workerId is given: the tool for any question about who " +
		"is expiring, lapsed or out of compliance by a date. Narrow to one kind with " +
		"credentialTypeCode, such as MED_CARD for a medical card or CDL for a licence. It " +
		"yields the credentialId and credentialTypeId the credential tools take."
}

func (t *listWorkerCredentialsTool) ParamSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: withPaging(map[string]any{
			paramWorkerID: wfWorkerProperty("Narrow to one worker"),
			paramWithinDays: toolschema.Clamped(map[string]any{
				toolschema.KeyType:    toolschema.TypeInteger,
				toolschema.KeyMinimum: 1,
				toolschema.KeyMaximum: maxExpiryHorizonDays,
				toolschema.KeyDescription: fmt.Sprintf(
					"List only credentials expiring within this many days, soonest first. "+
						"Defaults to %d when no workerId is given, at most %d.",
					defaultExpiryHorizonDays, maxExpiryHorizonDays,
				),
			}),
			paramCredentialTypeCode: map[string]any{
				toolschema.KeyType: toolschema.TypeString,
				toolschema.KeyDescription: "Optional credential type code, such as MED_CARD " +
					"or CDL. Omit to include every kind the organization tracks.",
			},
			paramIncludeExpired: wfBoolProperty(fmt.Sprintf(
				"List what falls due including credentials that lapsed in the last %d days. "+
					"Defaults to false, which returns only what is still valid but due.",
				expiredGraceDays,
			)),
			paramRequiredOnly: wfBoolProperty(
				"Only credential types the organization marks required.",
			),
			paramIncludeArchived: wfBoolProperty("With workerId and no expiry filter, also " +
				"return the worker's archived credentials. What falls due is always active " +
				"credentials."),
		}, defaultListLimit, maxCredentialRows),
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *listWorkerCredentialsTool) Policy() agentports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{
		resource: permission.ResourceWorkerCredential,
	})
}

func (t *listWorkerCredentialsTool) Query(
	ctx context.Context,
	params *agentports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	filter, err := readCredentialFilter(params.Params)
	if err != nil {
		return nil, err
	}

	window := readPage(params.Params, defaultListLimit, maxCredentialRows)
	if filter.byExpiry {
		return t.listFallingDue(ctx, params, filter, window)
	}

	return t.listForWorker(ctx, params, filter, window)
}

func (t *listWorkerCredentialsTool) listFallingDue(
	ctx context.Context,
	params *agentports.QueryToolParams,
	filter credentialFilter,
	window page,
) (any, error) {
	criteria := filtercatalog.NewCriteria(credentialEntity).At(clockFor(params))
	filter.describe(criteria)

	grace := 0
	if filter.includeExpired {
		grace = expiredGraceDays
	}

	// The tenant comes from the actor, never from the model. ListExpiring walks
	// every tenant when handed a zero TenantInfo — that is what the nightly
	// compliance sweep wants and exactly what a model-driven caller must never
	// be able to ask for.
	request := &workerports.ListExpiringWorkerCredentialsRequest{
		TenantInfo:          tenantOf(params),
		HorizonDays:         filter.horizon,
		GraceDays:           grace,
		AsOf:                criteria.Clock.Today(),
		RequiredOnly:        filter.requiredOnly,
		CredentialTypeCodes: filter.codes,
		WorkerID:            filter.workerID,
		SoonestFirst:        true,
		Offset:              window.offset,
		Limit:               window.fetch(),
	}
	credentials, err := t.expiring.ListExpiring(ctx, request)
	if err != nil {
		return nil, err
	}

	rows := make([]expiringCredentialRow, 0, len(credentials))
	for _, credential := range credentials {
		rows = append(rows, toExpiringRow(credential, criteria.Clock))
	}
	rows, more := trim(window, rows)

	return searchResult(criteria, rows, len(rows)).paged(window, more), nil
}

func (t *listWorkerCredentialsTool) listForWorker(
	ctx context.Context,
	params *agentports.QueryToolParams,
	filter credentialFilter,
	window page,
) (any, error) {
	includeArchived := optionalBool(params.Params, paramIncludeArchived)
	found, err := t.credentials.ListForWorker(ctx, tenantOf(params), filter.workerID,
		includeArchived)
	if err != nil {
		return nil, err
	}

	gate := t.access.gate(ctx, params, permission.ResourceWorkerCredential)
	rows := make([]credentialRow, 0, len(found))
	for _, credential := range found {
		if filter.admits(credential) {
			rows = append(rows, toCredentialRow(credential, gate))
		}
	}

	criteria := filtercatalog.NewCriteria(credentialEntity).At(clockFor(params))
	filter.describe(criteria)
	if includeArchived {
		criteria.Field("including", "archived")
	}

	shown, more := slicePage(window, rows)
	if shown == nil {
		shown = []credentialRow{}
	}
	outcome := searchResult(criteria, shown, len(shown)).paged(window, more)
	if len(rows) == 0 && len(filter.codes) == 0 && !filter.requiredOnly {
		outcome.Note = "No credentials are on file for this worker. A credential never " +
			"recorded is worse than one expiring soon: compare credentialTypes for what " +
			"the organization requires."
	}

	result := &workerCredentialsOutcome{gatedOutcome: *gatedResult(&outcome, gate)}
	if t.access.mayRead(ctx, params, permission.ResourceWorkerCredentialType) {
		types, typeErr := t.credentials.ListTypes(ctx, &workerports.ListCredentialTypesRequest{
			Filter: catalogFilter(params),
			Cursor: pagination.CursorInfo{Limit: maxCatalogRows},
			Status: activeStatus,
		})
		if typeErr != nil {
			return nil, typeErr
		}
		result.Types = make([]credentialTypeRow, 0, len(types.Items))
		for _, kind := range types.Items {
			result.Types = append(result.Types, credentialTypeRow{
				ID:         kind.ID.String(),
				Code:       kind.Code,
				Name:       kind.Name,
				Category:   string(kind.Category),
				IsRequired: kind.IsRequired,
			})
		}
	}

	return result, nil
}

func toCredentialRow(credential *worker.WorkerCredential, gate *fieldGate) credentialRow {
	row := credentialRow{
		ID:               credential.ID.String(),
		CredentialTypeID: credential.CredentialTypeID.String(),
		Status:           string(credential.Status),
		Number:           gatedText(gate, "number", credential.Number),
		IssuingAuthority: credential.IssuingAuthority,
		IssuedAt:         pointerDate(credential.IssuedAt),
		ExpiresAt: expectedDate(
			typeutils.ValueOrZero(credential.ExpiresAt),
			absentNotExpiring,
		),
		Verified:    credential.IsVerified(),
		HasDocument: credential.DocumentID.IsNotNil(),
	}
	if credential.CredentialType != nil {
		row.CredentialType = credential.CredentialType.Name
		row.CredentialCode = credential.CredentialType.Code
	}

	return row
}

func toExpiringRow(credential *worker.WorkerCredential, clk clock) expiringCredentialRow {
	// The credential's number is not on the row. Who needs a new medical
	// card is answered without it, and a licence or TWIC number in a chat
	// transcript is a leak with nobody to blame.
	row := expiringCredentialRow{
		ID:               credential.ID.String(),
		WorkerID:         credential.WorkerID.String(),
		CredentialTypeID: credential.CredentialTypeID.String(),
	}

	row.ExpiresAt = pointerDate(credential.ExpiresAt)
	if credential.ExpiresAt != nil {
		// Calendar days in the organization's zone, so a card expiring at
		// 00:30 tomorrow is one day out at any hour tonight — not zero.
		row.DaysUntilExpiry = clk.DaysBetween(clk.Instant(), *credential.ExpiresAt)
		row.Expired = *credential.ExpiresAt < clk.Instant()
	}

	if credential.CredentialType != nil {
		row.CredentialType = credential.CredentialType.Name
		row.CredentialCode = credential.CredentialType.Code
	}

	row.WorkerName = workerName(credential.Worker)

	return row
}
