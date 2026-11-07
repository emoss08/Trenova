package invoicetools

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/app/invoicepostingservice"
	"github.com/stretchr/testify/assert"
)

/*
An agent that posted an invoice said "I don't know whether this customer is
sent invoices automatically" and offered to send one that may already have
gone. The post preview now says whether posting emails it, and to whom.
*/
func TestEmailSentence_SaysWhetherPostingEmailsTheInvoice(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		plan *invoicepostingservice.EmailPlan
		want string
	}{
		"nothing planned": {want: ""},
		"no email at all": {
			plan: &invoicepostingservice.EmailPlan{AutoSend: true},
			want: "The customer does not take invoices by email, so nobody is emailed.",
		},
		"sent by hand": {
			plan: &invoicepostingservice.EmailPlan{Allowed: true},
			want: "Nobody is emailed: this customer is not sent invoices automatically, so it " +
				"goes out only when someone sends it (invoice_send).",
		},
		"automatic with nobody to send to": {
			plan: &invoicepostingservice.EmailPlan{Allowed: true, AutoSend: true},
			want: "The customer is sent invoices automatically but has no billing email, so " +
				"nobody is emailed.",
		},
		"automatic to one address": {
			plan: &invoicepostingservice.EmailPlan{
				Allowed: true, AutoSend: true, Recipients: []string{"ap@crown.example"},
			},
			want: "It is emailed to ap@crown.example on its own once its PDF is made.",
		},
		"automatic to several": {
			plan: &invoicepostingservice.EmailPlan{
				Allowed: true, AutoSend: true,
				Recipients: []string{"ap@crown.example", "billing@crown.example"},
			},
			want: "It is emailed to ap@crown.example and 1 more on its own once its PDF is made.",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, emailSentence(tc.plan))
		})
	}
}
