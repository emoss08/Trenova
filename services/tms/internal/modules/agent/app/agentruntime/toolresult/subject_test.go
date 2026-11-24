package toolresult

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	iftaReturnID = "ifr_01M80ZFAMGBAG3MS1AQAV8EGSP"
	acmePartner  = "edip_01M7V48HPFRT5N51S07BK4QZ5C"
	otherPartner = "edip_01M7WF6QV7RTRADZ08AXDXG77E"
	novemberID   = "fp_01M7V48EJ3MZHT1XMECQED4H71"
)

func transferRow(id, partner string) string {
	return `{"id":"` + id + `","bol":"88244","direction":"Inbound","status":"Approved",` +
		`"counterparty":{"code":"ACMEMFG","id":"` + partner + `","name":"Acme Manufacturing EDI"}}`
}

func TestSubject_IsTheOneRecordAMetricReadsThatAResultIsAbout(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		content string
		subject string
	}{
		"the IFTA list holding one return (final-a, beyond-ifta-ready-to-file #2)": {
			content: Fence("fuel_list_ifta_returns", `{"count":1,"items":[{"id":"`+iftaReturnID+
				`","quarter":4,"year":2026,"status":"Draft"}],"total":1}`),
			subject: iftaReturnID,
		},
		"a day of tenders, every row the same partner's (final-a, beyond-edi-todays-tenders)": {
			content: Fence("edi_list_transfers", `{"count":2,"items":[`+
				transferRow("edilt_01M85JM3GBC6GG1DN85XW6W620", acmePartner)+","+
				transferRow("edilt_01M85JM3FY0CV3268MED8VB0GD", acmePartner)+`]}`),
			subject: acmePartner,
		},
		"the close blockers, about the period they name": {
			content: Fence("ledger_get_fiscal_close_blockers", `{"blockerCount":4,`+
				`"canClose":false,"fiscalPeriodId":"`+novemberID+`"}`),
			subject: novemberID,
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			subject, about := Subject(tc.content)
			assert.True(t, about)
			assert.Equal(t, tc.subject, subject.String())
		})
	}
}

func TestSubject_IsNoneWhenTheResultIsAboutSomethingElse(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"one tender is about that tender, not its partner's day (the EDI twin)": Fence(
			"edi_list_transfers",
			`{"count":1,"items":[`+transferRow("edilt_01M85JM3GBC6GG1DN85XW6W620", acmePartner)+`]}`,
		),
		"two partners' tenders": Fence("edi_list_transfers", `{"count":2,"items":[`+
			transferRow("edilt_01M85JM3GBC6GG1DN85XW6W620", acmePartner)+","+
			transferRow("edilt_01M85JM3FY0CV3268MED8VB0GD", otherPartner)+`]}`),
		"three returns": Fence("fuel_list_ifta_returns", `{"items":[`+
			`{"id":"ifr_01M80ZFAMGBAG3MS1AQAV8EGSP"},{"id":"ifr_01M80ZFAMGBAG3MS1AQAV8EGSQ"},`+
			`{"id":"ifr_01M80ZFAMGBAG3MS1AQAV8EGSR"}]}`),
		"one message, which has its own id": Fence("edi_get_message",
			`{"id":"edimsg_01M85JM3GFFF4WAYQ80RQVZ5BQ","partner":{"id":"`+acmePartner+`"}}`),
		"not a result": "the tool failed",
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, about := Subject(content)
			assert.False(t, about)
		})
	}
}
