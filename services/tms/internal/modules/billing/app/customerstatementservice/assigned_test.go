package customerstatementservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/ports/customerledgerports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestAssignedToNamesWhoAnOlderInvoiceIsOwedTo(t *testing.T) {
	t.Parallel()

	current := &factoringcompany.FactoringCompany{ID: pulid.MustNew("fac_"), Name: "Current Capital"}
	previous := pulid.MustNew("fac_")
	factored := &prepared{factor: current}
	unfactored := &prepared{}

	assert.Empty(t, assignedTo(&customerledgerports.AROpenItem{
		FactoringCompanyID: current.ID,
		FactoringCompany:   current.Name,
	}, factored, "Acme"), "an invoice owed to the remit-to factor needs no note")
	assert.Equal(t, "Old Capital", assignedTo(&customerledgerports.AROpenItem{
		FactoringCompanyID: previous,
		FactoringCompany:   "Old Capital",
	}, factored, "Acme"))
	assert.Equal(t, "Acme", assignedTo(&customerledgerports.AROpenItem{}, factored, "Acme"),
		"an invoice posted before the customer was factored is owed to the carrier")
	assert.Empty(t, assignedTo(&customerledgerports.AROpenItem{}, unfactored, "Acme"))
	assert.Equal(t, "Old Capital", assignedTo(&customerledgerports.AROpenItem{
		FactoringCompanyID: previous,
		FactoringCompany:   "Old Capital",
	}, unfactored, "Acme"), "a customer no longer factored still owes the factor its older invoices")
}
