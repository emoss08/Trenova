package agentquerytoolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolschema"
	"github.com/emoss08/trenova/internal/core/services/detentionservice"
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

type openClockPricer interface {
	PriceOpenClock(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		occurrence *detention.DetentionOccurrence,
	) (*detentionservice.ClockPrice, error)
}

type pricedOccurrence struct {
	*detention.DetentionOccurrence

	CurrentlyBillable *currentlyBillable `json:"currentlyBillable,omitempty"`
}

type currentlyBillable struct {
	BillableMinutes int32        `json:"billableMinutes"`
	BillableAmount  string       `json:"billableAmount"`
	AsOf            optionalDate `json:"asOf"`
	Note            string       `json:"note"`
}

func provideGetDetentionOccurrenceTool(
	repo repositories.DetentionOccurrenceRepository,
	permissions serviceports.PermissionEngine,
	detentions *detentionservice.Service,
) serviceports.AgentQueryTool {
	return newGetDetentionOccurrenceTool(repo, permissions, detentions)
}

func newGetDetentionOccurrenceTool(
	repo repositories.DetentionOccurrenceRepository,
	permissions serviceports.PermissionEngine,
	pricer openClockPricer,
) serviceports.AgentQueryTool {
	return newGetTool(&getSpec{
		name:     "get_detention_occurrence",
		entity:   "detention occurrence",
		resource: permission.ResourceDetentionPolicy,
		kinds:    []permission.RecordKind{permission.KindDetentionOccurrence},
		summary: "Retrieve one detention occurrence by id: the stop, the clock, free time, " +
			"billable minutes, amounts, notice status, and the evidence and notices on file. " +
			"A clock still running also carries currentlyBillable, priced as of now; quote " +
			"that rather than the stored amounts, which date from the last recalculation. " +
			"Use list_detention_desk first when you do not have an id.",
		paramName: "occurrenceId",
		idSource:  "from list_detention_desk, the page you are on, or this run's subject",
		fetch: func(ctx context.Context, id pulid.ID, tenant pagination.TenantInfo) (any, error) {
			occurrence, err := repo.GetByID(ctx, &repositories.GetDetentionOccurrenceByIDRequest{
				OccurrenceID:    id,
				TenantInfo:      tenant,
				IncludeEvidence: true,
				IncludeNotices:  true,
			})
			if err != nil || pricer == nil {
				return occurrence, err
			}

			price, err := pricer.PriceOpenClock(ctx, tenant, occurrence)
			if err != nil || price == nil {
				return occurrence, err
			}

			return pricedOccurrence{
				DetentionOccurrence: occurrence,
				CurrentlyBillable: &currentlyBillable{
					BillableMinutes: price.BillableMinutes,
					BillableAmount:  price.BillableAmount.StringFixed(2),
					AsOf:            recordedDate(price.AsOf),
					Note: "The clock is still running. These are what it would bill if " +
						"the truck left now; the stored billableMinutes and billableAmount " +
						"are from the last recalculation.",
				},
			}, nil
		},
	}, permissions)
}

func newGetCarrierIntelEventTool(
	repo repositories.CarrierIntelEventRepository,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return newGetTool(&getSpec{
		name:   "get_carrier_intel_event",
		entity: "carrier intelligence event",
		searchTerms: []string{
			"carrier risk alert", "authority change", "insurance lapse", "safety rating change",
		},
		resource: permission.ResourceCarrierIntelligence,
		kinds:    []permission.RecordKind{permission.KindCarrierIntelligenceEvent},
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
	}, permissions)
}

func newGetServiceFailureTool(
	repo repositories.ServiceFailureRepository,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return newGetTool(&getSpec{
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
	}, permissions)
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
			"credentialId": agenttoolschema.RecordIDText(
				permission.ResourceWorkerCredential,
				"The worker credential's id, from list_expiring_credentials "+
					"or the page you are on.",
			),
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

// updatePreferenceWarning is what a customer set to None means for the agent.
// It used to name only a delay and a new ETA as what the person may still ask
// for, and Haiku, asked to send FreshHaul's arrival notice, read that notice as
// routine and set no wait.
func updatePreferenceWarning(preference customer.StatusUpdatePreference) string {
	if preference != customer.StatusUpdateNone {
		return ""
	}

	return "This customer gets no routine arrival or departure notices, so never send one " +
		"on your own. An arrival or departure notice the person asks for is not routine: " +
		"it still goes out with their approval, and when it waits on the truck, wait for " +
		"it as asked. So does any other update they ask for, such as a delay or a new ETA."
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
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return newGetTool(&getSpec{
		name:     "get_customer_update_preferences",
		entity:   "customer",
		resource: permission.ResourceCustomer,
		summary: "Retrieve what a customer asked to be told as their freight moves: " +
			"whether they want arrival notices, departure notices, both or neither, and " +
			"who receives them. A customer set to None gets no routine arrival or " +
			"departure notice; that does not stop an update the person asks you to send, " +
			"such as an arrival notice, a delay or a new ETA, which goes out on their " +
			"approval. When the " +
			"recipient list is empty the notice profile's own recipients are the fallback. " +
			"Call this before writing any status update.",
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
			if warning := updatePreferenceWarning(entity.StatusUpdatePreference); warning != "" {
				out["warning"] = warning
			}
			if entity.StatusUpdateRecipients == "" && entity.EmailProfile != nil {
				out["fallbackRecipients"] = entity.EmailProfile.ToRecipients
			}

			return out, nil
		},
	}, permissions)
}
