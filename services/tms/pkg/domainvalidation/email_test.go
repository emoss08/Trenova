package domainvalidation_test

import (
	"testing"

	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/stretchr/testify/assert"
)

func TestEmailListAcceptsEveryAddressOfAList(t *testing.T) {
	t.Parallel()

	rule := domainvalidation.EmailList("bad")
	for _, value := range []string{
		"",
		"ap@customer.example.com",
		"ap@customer.example.com, cfo@customer.example.com",
		"ap@customer.example.com;cfo@customer.example.com",
	} {
		assert.NoError(t, rule.Validate(value), value)
	}
}

func TestEmailListRefusesAListWithOneBadAddress(t *testing.T) {
	t.Parallel()

	rule := domainvalidation.EmailList("bad")
	for _, value := range []any{
		"ap@customer.example.com, not-an-address",
		"not-an-address",
		" , ",
		42,
	} {
		err := rule.Validate(value)
		if assert.Error(t, err, value) {
			assert.Equal(t, "bad", err.Error())
		}
	}
}
