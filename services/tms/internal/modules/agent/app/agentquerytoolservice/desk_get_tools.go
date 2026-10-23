package agentquerytoolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
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

func carrierIntelEventGetSpec() getSpec {
	return getSpec{
		name:   "carrier_get_intelligence_event",
		entity: "carrier intelligence event",
		searchTerms: []string{
			"carrier risk alert", "authority change", "insurance lapse", "safety rating change",
		},
		resource: permission.ResourceCarrierIntelligence,
		kinds:    []permission.RecordKind{permission.KindCarrierIntelligenceEvent},
		summary: "Retrieve one carrier intelligence event by id: what changed on the " +
			"carrier's authority, insurance or safety record, and its severity. It also " +
			"gives the prior and current values and whether anyone has acknowledged or " +
			"resolved it. Use carrier_list_intelligence_events first when you do not have an id, " +
			"and carrier_get for the carrier itself.",
		paramName: "eventId",
		idSource: "from carrier_list_intelligence_events, the page you are on, this run's subject " +
			"or a mentioned record",
	}
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
// It is deliberately narrow. customer_get returns the whole record, and a
// model reading a whole customer to find one field reads thirty others it
// does not need — including the billing addresses, which are not who asked
// for status updates.
func newGetCustomerUpdatePreferencesTool(
	repo customerports.CustomerRepository,
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return newGetTool(&getSpec{
		name:     "customer_get_update_preferences",
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
		idSource:  "from customer_list or the customerId on shipment_get",
		fetch: func(ctx context.Context, id pulid.ID, tenant pagination.TenantInfo) (any, error) {
			entity, err := repo.GetByID(ctx, customerports.GetCustomerByIDRequest{
				ID:         id,
				TenantInfo: tenant,
				CustomerFilterOptions: customerports.CustomerFilterOptions{
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
