package factoringsql

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const itemJoin = "EXISTS (SELECT 1 FROM factoring_submission_items AS facsit WHERE " +
	"facsit.organization_id = inv.organization_id AND " +
	"facsit.business_unit_id = inv.business_unit_id AND " +
	"facsit.invoice_id = inv.id AND facsit.status IN "

func TestChargedBackInvoice(t *testing.T) {
	t.Parallel()

	assert.Equal(t, itemJoin+"('ChargedBack'))", ChargedBackInvoice())
}

func TestLiveInvoice(t *testing.T) {
	t.Parallel()

	assert.Equal(t, itemJoin+"('Pending', 'Funded'))", LiveInvoice())
}
