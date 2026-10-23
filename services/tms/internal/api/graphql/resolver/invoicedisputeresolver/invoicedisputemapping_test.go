package invoicedisputeresolver

import (
	"testing"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDisputedInvoiceIDs_NamesOneInvoiceOrSeveralNeverBoth(t *testing.T) {
	t.Parallel()

	first, second := pulid.MustNew("inv_"), pulid.MustNew("inv_")
	one := first.String()

	ids, err := disputedInvoiceIDs(&one, nil)
	require.NoError(t, err)
	assert.Equal(t, []pulid.ID{first}, ids)

	ids, err = disputedInvoiceIDs(nil, []string{second.String(), first.String(), second.String()})
	require.NoError(t, err)
	assert.Equal(t, []pulid.ID{second, first}, ids, "in the order named, each once")

	_, err = disputedInvoiceIDs(&one, []string{second.String()})
	require.Error(t, err)

	_, err = disputedInvoiceIDs(nil, nil)
	require.Error(t, err)

	_, err = disputedInvoiceIDs(nil, []string{"INV-1042"})
	require.Error(t, err)

	tooMany := make([]string, 0, 101)
	for range 101 {
		tooMany = append(tooMany, pulid.MustNew("inv_").String())
	}
	_, err = disputedInvoiceIDs(nil, tooMany)
	require.Error(t, err)
}
