package homequerytools

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/querytool"
	"github.com/emoss08/trenova/internal/modules/agent/app/querytools/readspec"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/pkg/filtercatalog"
)

/*
The records a Desk page opens on.

A person asks about what is in front of them: a detention clock, a carrier
alert, a run that failed, a service failure, a credential about to lapse. Each
of those pages names its record in the page context, and each needs a get tool
so the model can read the record rather than the page title.
*/

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

const (
	fieldStatusUpdatePreference = "statusUpdatePreference"
	fieldStatusUpdateRecipients = "statusUpdateRecipients"
	fieldEmailProfile           = "emailProfile"
	fieldToRecipients           = "toRecipients"
)

// CustomerUpdatePreferencesGetSpec answers the one question the customer
// update desk has to ask before it writes anything: does this customer want
// to be told, and who is on the list. It reads DeskCustomerUpdatePreferences.
//
// It is deliberately narrow. customer_get returns the whole record, and a
// model reading a whole customer to find one field reads thirty others it
// does not need — including the billing addresses, which are not who asked
// for status updates.
func CustomerUpdatePreferencesGetSpec() readspec.Get {
	return readspec.Get{
		Name:     "customer_get_update_preferences",
		Entity:   querytool.LabelCustomer,
		Resource: permission.ResourceCustomer,
		Summary: "Retrieve what a customer asked to be told as their freight moves: " +
			"whether they want arrival notices, departure notices, both or neither, and " +
			"who receives them. A customer set to None gets no routine arrival or " +
			"departure notice; that does not stop an update the person asks you to send, " +
			"such as an arrival notice, a delay or a new ETA, which goes out on their " +
			"approval. When the " +
			"recipient list is empty the notice profile's own recipients are the fallback. " +
			"Call this before writing any status update.",
		ParamName: querytool.ParamCustomerID,
		IDSource:  "from customer_list or the customerId on shipment_get",
		Shape:     shapeUpdatePreferences,
	}
}

// shapeUpdatePreferences words what the preference means for the agent:
// whether arrivals and departures are wanted, the warning a customer set to
// None needs, and the notice profile's recipients when the customer names
// none. The profile itself is only the fallback, so it leaves the record.
func shapeUpdatePreferences(
	record map[string]any,
	withheld map[string]struct{},
	_ filtercatalog.Clock,
) {
	profile, _ := record[fieldEmailProfile].(map[string]any)
	delete(record, fieldEmailProfile)
	if _, hidden := withheld[fieldStatusUpdatePreference]; hidden {
		return
	}
	raw, _ := record[fieldStatusUpdatePreference].(string)
	preference := customer.StatusUpdatePreference(raw)
	record["wantsArrivals"] = preference.WantsArrivals()
	record["wantsDepartures"] = preference.WantsDepartures()
	if warning := updatePreferenceWarning(preference); warning != "" {
		record["warning"] = warning
	}
	if _, hidden := withheld[fieldStatusUpdateRecipients]; hidden {
		return
	}
	if recipients, _ := record[fieldStatusUpdateRecipients].(string); recipients == "" {
		if fallback, ok := profile[fieldToRecipients].(string); ok && fallback != "" {
			record["fallbackRecipients"] = fallback
		}
	}
}
