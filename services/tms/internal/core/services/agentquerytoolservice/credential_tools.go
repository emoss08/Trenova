package agentquerytoolservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/pagination"
)

const (
	defaultExpiryHorizonDays = 30
	maxExpiryHorizonDays     = 365
	// expiredGraceDays is how far back "already expired" reaches. A card that
	// lapsed last quarter is still the fleet's problem, and someone asking who
	// is out of date means them too.
	expiredGraceDays = 90
	maxExpiringRows  = 100
)

// expiringCredentialRow is one credential in the words someone chasing renewals
// would use.
//
// The stored entity carries the worker's whole profile, the scanned document
// and the user who verified it. None of that helps answer who needs a new
// medical card, and all of it costs context the model needs for the answer.
type expiringCredentialRow struct {
	ID              string       `json:"id"`
	WorkerID        string       `json:"workerId"`
	WorkerName      string       `json:"workerName"`
	CredentialType  string       `json:"credentialType"`
	CredentialCode  string       `json:"credentialCode"`
	ExpiresAt       optionalDate `json:"expiresAt"`
	DaysUntilExpiry int64        `json:"daysUntilExpiry"`
	// Expired is stated rather than left to be derived from a negative day
	// count, because the difference between "renew this" and "this driver
	// should not be dispatched" is the whole point of the question.
	Expired bool `json:"expired"`
}

type listExpiringCredentialsTool struct {
	repo repositories.WorkerCredentialRepository
}

func newListExpiringCredentialsTool(
	repo repositories.WorkerCredentialRepository,
) serviceports.AgentQueryTool {
	return &listExpiringCredentialsTool{repo: repo}
}

func (t *listExpiringCredentialsTool) Name() string { return "list_expiring_credentials" }

func (t *listExpiringCredentialsTool) Description() string {
	return "List worker (driver) credentials falling due, newest expiry first — " +
		"medical cards, licences, hazmat endorsements and anything else the " +
		"organization tracks. This is the tool for any question about who is " +
		"expiring, lapsed, or out of compliance by a date; search_worker cannot " +
		"filter on dates. Narrow to one kind with credentialTypeCode, such as " +
		"MED_CARD for a medical card or CDL for a licence."
}

func (t *listExpiringCredentialsTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"withinDays": map[string]any{
				"type": "integer",
				"description": fmt.Sprintf(
					"How far ahead to look, in days. Defaults to %d, at most %d.",
					defaultExpiryHorizonDays, maxExpiryHorizonDays,
				),
			},
			"credentialTypeCode": map[string]any{
				"type": "string",
				"description": "Optional credential type code, such as MED_CARD or CDL. " +
					"Omit to include every kind the organization tracks.",
			},
			"includeExpired": map[string]any{
				"type": "boolean",
				"description": "Also return credentials that have already lapsed. " +
					"Defaults to false, which returns only what is still valid but due.",
			},
			"requiredOnly": map[string]any{
				"type":        "boolean",
				"description": "Only credential types the organization marks required.",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": fmt.Sprintf("How many to return, at most %d.", maxExpiringRows),
			},
		},
		"additionalProperties": false,
	}
}

func (t *listExpiringCredentialsTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{
		resource: permission.ResourceWorkerCredential,
	})
}

func (t *listExpiringCredentialsTool) Query(
	ctx context.Context,
	params serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	horizon := optionalInt(params.Params, "withinDays", defaultExpiryHorizonDays)
	if horizon <= 0 || horizon > maxExpiryHorizonDays {
		horizon = defaultExpiryHorizonDays
	}

	limit := optionalInt(params.Params, "limit", defaultSearchLimit)
	if limit <= 0 || limit > maxExpiringRows {
		limit = maxExpiringRows
	}

	grace := 0
	includeExpired := optionalBool(params.Params, "includeExpired")
	if includeExpired {
		grace = expiredGraceDays
	}

	requiredOnly := optionalBool(params.Params, "requiredOnly")

	// Codes are stored upper-case; a model writing "med_card" means the same
	// credential and should not get an empty list for the difference.
	var codes []string
	if code := optionalString(params.Params, "credentialTypeCode"); code != "" {
		codes = []string{strings.ToUpper(code)}
	}

	criteria := filtercatalog.NewCriteria("credentials").At(clockFor(params))
	criteria.Field("expiring within", fmt.Sprintf("%d days", horizon))
	if len(codes) > 0 {
		criteria.Field("credential type", codes[0])
	}
	if includeExpired {
		criteria.Field("including", "already expired")
	}
	if requiredOnly {
		criteria.Field("limited to", "required credential types")
	}

	// The tenant comes from the actor, never from the model. ListExpiring walks
	// every tenant when handed a zero TenantInfo — that is what the nightly
	// compliance sweep wants and exactly what a model-driven caller must never
	// be able to ask for.
	credentials, err := t.repo.ListExpiring(ctx, &repositories.ListExpiringWorkerCredentialsRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  params.OrganizationID,
			BuID:   params.BusinessUnitID,
			UserID: params.Actor.UserID,
		},
		HorizonDays:         horizon,
		GraceDays:           grace,
		AsOf:                criteria.Clock.Today(),
		RequiredOnly:        requiredOnly,
		CredentialTypeCodes: codes,
		Limit:               limit,
	})
	if err != nil {
		return nil, err
	}

	rows := make([]expiringCredentialRow, 0, len(credentials))
	for _, credential := range credentials {
		rows = append(rows, toExpiringRow(credential, criteria.Clock))
	}

	return searchResult(criteria, rows, len(rows)), nil
}

func toExpiringRow(credential *worker.WorkerCredential, clk clock) expiringCredentialRow {
	// The credential's number is not on the row. Who needs a new medical
	// card is answered without it, and a licence or TWIC number in a chat
	// transcript is a leak with nobody to blame.
	row := expiringCredentialRow{
		ID:       credential.ID.String(),
		WorkerID: credential.WorkerID.String(),
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
