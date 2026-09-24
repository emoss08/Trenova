package agentquerytoolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

/*
The records a Desk page opens on.

A person asks about what is in front of them: a detention clock, a carrier
alert, a run that failed, a service failure, a credential about to lapse. Each
of those pages names its record in the page context, and each needs a get tool
so the model can read the record rather than the page title.
*/

func newGetDetentionOccurrenceTool(
	repo repositories.DetentionOccurrenceRepository,
) serviceports.AgentQueryTool {
	return newGetTool(getSpec{
		name:     "get_detention_occurrence",
		entity:   "detention occurrence",
		resource: permission.ResourceDetentionPolicy,
		summary: "Retrieve one detention occurrence by id: the stop, the clock, free time, " +
			"billable minutes, amounts, notice status, and the evidence and notices on file. " +
			"Use list_detention_desk first when you do not have an id.",
		paramName: "occurrenceId",
		idSource:  "from list_detention_desk, the page you are on, or this run's subject",
		fetch: func(ctx context.Context, id pulid.ID, tenant pagination.TenantInfo) (any, error) {
			return repo.GetByID(ctx, &repositories.GetDetentionOccurrenceByIDRequest{
				OccurrenceID:    id,
				TenantInfo:      tenant,
				IncludeEvidence: true,
				IncludeNotices:  true,
			})
		},
	})
}

func newGetCarrierIntelEventTool(
	repo repositories.CarrierIntelEventRepository,
) serviceports.AgentQueryTool {
	return newGetTool(getSpec{
		name:     "get_carrier_intel_event",
		entity:   "carrier intelligence event",
		resource: permission.ResourceCarrierIntelligence,
		summary: "Retrieve one carrier intelligence event by id: what changed on the " +
			"carrier's authority, insurance or safety record, and its severity. It also " +
			"gives the prior and current values and whether anyone has acknowledged or " +
			"resolved it. No tool lists these events; use get_carrier for the carrier itself.",
		paramName: "eventId",
		idSource: "from the page you are on, this run's subject, or a mentioned record " +
			"(no tool lists these events)",
		fetch: func(ctx context.Context, id pulid.ID, tenant pagination.TenantInfo) (any, error) {
			events, err := repo.GetByIDs(ctx, tenant, []pulid.ID{id})
			if err != nil {
				return nil, err
			}
			if len(events) == 0 {
				return nil, errortypes.NewNotFoundError("Carrier intelligence event not found")
			}

			return events[0], nil
		},
	})
}

func newGetServiceFailureTool(
	repo repositories.ServiceFailureRepository,
) serviceports.AgentQueryTool {
	return newGetTool(getSpec{
		name:     "get_service_failure",
		entity:   "service failure",
		resource: permission.ResourceServiceFailure,
		summary: "Retrieve one service failure by id: the late or missed stop, how late, " +
			"the reason code, the notes, and who reviewed, resolved or voided it. Use " +
			"list_service_failures first when you do not have an id.",
		paramName: "serviceFailureId",
		idSource:  "from list_service_failures or the page you are on",
		fetch: func(ctx context.Context, id pulid.ID, tenant pagination.TenantInfo) (any, error) {
			return repo.GetByID(ctx, &repositories.GetServiceFailureByIDRequest{
				ID:         id,
				TenantInfo: tenant,
			})
		},
	})
}

// getWorkerCredentialTool reads one credential under the person's field
// access. A credential number is a licence or medical card number, which the
// resource keeps at the Restricted tier: it reaches a model only for a person
// whose role reaches it, and an agent principal never sees it.
type getWorkerCredentialTool struct {
	repo   repositories.WorkerCredentialRepository
	access fieldAccess
}

func newGetWorkerCredentialTool(
	repo repositories.WorkerCredentialRepository,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return &getWorkerCredentialTool{repo: repo, access: newFieldAccess(permissions)}
}

func (t *getWorkerCredentialTool) Name() string { return "get_worker_credential" }

func (t *getWorkerCredentialTool) Description() string {
	return "Retrieve one worker credential by id: its type, the worker who holds it, its " +
		"status, and when it was issued and expires. It also says whether it was verified " +
		"and whether a document is on file. Use list_expiring_credentials first when you " +
		"do not have an id."
}

func (t *getWorkerCredentialTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"credentialId": map[string]any{
				"type": "string",
				"description": "The worker credential's id, from list_expiring_credentials " +
					"or the page you are on.",
			},
		},
		"required":             []string{"credentialId"},
		"additionalProperties": false,
	}
}

func (t *getWorkerCredentialTool) Policy() serviceports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{
		resource: permission.ResourceWorkerCredential,
	})
}

func (t *getWorkerCredentialTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	credentialID, err := requirePulid(params.Params, "credentialId")
	if err != nil {
		return nil, err
	}

	entity, err := t.repo.GetByID(ctx, &repositories.GetWorkerCredentialByIDRequest{
		ID: credentialID,
		TenantInfo: pagination.TenantInfo{
			OrgID:  params.OrganizationID,
			BuID:   params.BusinessUnitID,
			UserID: params.Actor.UserID,
		},
		IncludeType:   true,
		IncludeWorker: true,
	})
	if err != nil {
		return nil, err
	}

	ceiling := t.access.ceiling(ctx, params, permission.ResourceWorkerCredential)

	return workerCredentialDetailFrom(entity, t.access, ceiling), nil
}

// workerCredentialDetail is one credential as a model may see it. The number
// is present only under a ceiling that reaches it and is named as withheld
// otherwise, so its absence reads as withheld rather than as not on file.
type workerCredentialDetail struct {
	ID               string       `json:"id"`
	WorkerID         string       `json:"workerId"`
	WorkerName       string       `json:"workerName,omitempty"`
	CredentialTypeID string       `json:"credentialTypeId"`
	CredentialType   string       `json:"credentialType,omitempty"`
	Category         string       `json:"category,omitempty"`
	Status           string       `json:"status"`
	Number           string       `json:"number,omitempty"`
	IssuingAuthority string       `json:"issuingAuthority,omitempty"`
	IssuedAt         optionalDate `json:"issuedAt"`
	ExpiresAt        optionalDate `json:"expiresAt"`
	Verified         bool         `json:"verified"`
	VerifiedAt       optionalDate `json:"verifiedAt"`
	DocumentID       string       `json:"documentId,omitempty"`
	HasDocument      bool         `json:"hasDocument"`
	Notes            string       `json:"notes,omitempty"`
	Archived         bool         `json:"archived"`
	ArchiveReason    string       `json:"archiveReason,omitempty"`
	Version          int64        `json:"version"`
	Withheld         []string     `json:"withheldByAccess,omitempty"`
}

func workerCredentialDetailFrom(
	entity *worker.WorkerCredential,
	access fieldAccess,
	ceiling permission.FieldSensitivity,
) workerCredentialDetail {
	detail := workerCredentialDetail{
		ID:               entity.ID.String(),
		WorkerID:         entity.WorkerID.String(),
		CredentialTypeID: entity.CredentialTypeID.String(),
		Status:           string(entity.Status),
		IssuingAuthority: entity.IssuingAuthority,
		IssuedAt:         pointerDate(entity.IssuedAt),
		ExpiresAt:        pointerDate(entity.ExpiresAt),
		Verified:         entity.VerifiedAt != nil,
		VerifiedAt:       pointerDate(entity.VerifiedAt),
		HasDocument:      !entity.DocumentID.IsNil(),
		Notes:            entity.Notes,
		Archived:         entity.ArchivedAt != nil,
		ArchiveReason:    entity.ArchiveReason,
		Version:          entity.Version,
	}
	if !entity.DocumentID.IsNil() {
		detail.DocumentID = entity.DocumentID.String()
	}
	if entity.CredentialType != nil {
		detail.CredentialType = entity.CredentialType.Name
		detail.Category = string(entity.CredentialType.Category)
	}
	if entity.Worker != nil {
		detail.WorkerName = entity.Worker.FirstName + " " + entity.Worker.LastName
	}
	if access.visible(permission.ResourceWorkerCredential, "number", ceiling) {
		detail.Number = entity.Number
	} else if entity.Number != "" {
		detail.Withheld = append(detail.Withheld, "number")
	}

	return detail
}

// newGetCustomerUpdatePreferencesTool answers the one question the customer
// update desk has to ask before it writes anything: does this customer want
// to be told, and who is on the list.
//
// It is deliberately narrow. get_customer returns the whole record, and a
// model reading a whole customer to find one field reads thirty others it
// does not need — including the billing addresses, which are not who asked
// for status updates.
func newGetCustomerUpdatePreferencesTool(
	repo repositories.CustomerRepository,
) serviceports.AgentQueryTool {
	return newGetTool(getSpec{
		name:     "get_customer_update_preferences",
		entity:   "customer",
		resource: permission.ResourceCustomer,
		summary: "Retrieve what a customer asked to be told as their freight moves: " +
			"whether they want arrivals, departures, both or nothing, and who receives " +
			"them. A customer set to None is not to be emailed about a stop at all. " +
			"When the recipient list is empty the notice profile's own recipients are " +
			"the fallback. Call this before writing any status update.",
		paramName: "customerId",
		idSource:  "from list_customers or the customerId on get_shipment",
		fetch: func(ctx context.Context, id pulid.ID, tenant pagination.TenantInfo) (any, error) {
			entity, err := repo.GetByID(ctx, repositories.GetCustomerByIDRequest{
				ID:         id,
				TenantInfo: tenant,
				CustomerFilterOptions: repositories.CustomerFilterOptions{
					IncludeEmailProfile: true,
				},
			})
			if err != nil {
				return nil, err
			}

			out := map[string]any{
				"customerId":             entity.ID.String(),
				"customerName":           entity.Name,
				"statusUpdatePreference": entity.StatusUpdatePreference,
				"wantsArrivals":          entity.StatusUpdatePreference.WantsArrivals(),
				"wantsDepartures":        entity.StatusUpdatePreference.WantsDepartures(),
				"statusUpdateRecipients": entity.StatusUpdateRecipients,
			}
			if entity.StatusUpdatePreference == customer.StatusUpdateNone {
				out["warning"] = "This customer has not asked for status updates; do not email them about a stop."
			}
			if entity.StatusUpdateRecipients == "" && entity.EmailProfile != nil {
				out["fallbackRecipients"] = entity.EmailProfile.ToRecipients
			}

			return out, nil
		},
	})
}
