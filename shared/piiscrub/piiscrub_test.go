package piiscrub_test

import (
	"testing"

	"github.com/emoss08/trenova/shared/piiscrub"
	"github.com/stretchr/testify/assert"
)

func TestScrub(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "masks a plain card number", in: "Card 4111111111111111 on file", want: "Card [CARD] on file"},
		{name: "masks a card grouped by spaces", in: "Card 4111 1111 1111 1111 exp 09/27", want: "Card [CARD] exp 09/27"},
		{name: "masks a card grouped by dashes", in: "4111-1111-1111-1111", want: "[CARD]"},
		{name: "masks an amex grouped four six five", in: "Amex 3782 822463 10005 billed", want: "Amex [CARD] billed"},
		{name: "masks every card in the text", in: "Mastercard 5500000000000004 and Discover 6011111111111117", want: "Mastercard [CARD] and Discover [CARD]"},
		{name: "keeps a number that fails luhn", in: "Card 4111111111111112 declined", want: "Card 4111111111111112 declined"},
		{name: "masks the card but keeps a trailing security code", in: "4111 1111 1111 1111 123", want: "[CARD] 123"},
		{name: "keeps phone numbers", in: "Call 555-123-4567 or 555-765-4321", want: "Call 555-123-4567 or 555-765-4321"},
		{name: "keeps adjacent phone numbers", in: "555-123-4567 555-765-4321", want: "555-123-4567 555-765-4321"},
		{name: "keeps a run of one repeated digit", in: "0000000000000000", want: "0000000000000000"},
		{name: "keeps a tracking number", in: "Tracking 1Z999AA10123456784", want: "Tracking 1Z999AA10123456784"},
		{name: "masks a dashed ssn", in: "SSN on file: 123-45-6789.", want: "SSN on file: [SSN]."},
		{name: "masks a spaced ssn without a label", in: "Employee 123 45 6789 hired", want: "Employee [SSN] hired"},
		{name: "keeps an ssn shape with mixed separators", in: "Split 123-45 6789 stays", want: "Split 123-45 6789 stays"},
		{name: "keeps numbers the ssa never issues", in: "000-12-3456 666-12-3456 912-34-5678", want: "000-12-3456 666-12-3456 912-34-5678"},
		{name: "masks a labelled undelimited ssn", in: "SSN: 123456789", want: "SSN: [SSN]"},
		{name: "masks a social security number by name", in: "social security number 078051120", want: "social security number [SSN]"},
		{name: "keeps an unlabelled nine digit reference", in: "PRO 123456789 delivered", want: "PRO 123456789 delivered"},
		{name: "masks a labelled routing number", in: "Routing number: 021000021", want: "Routing number: [ROUTING]"},
		{name: "keeps a routing number with a bad checksum", in: "Routing: 021000022", want: "Routing: 021000022"},
		{name: "masks an aba number", in: "ABA 011000015 wire", want: "ABA [ROUTING] wire"},
		{name: "keeps an unlabelled routing number", in: "Bank 021000021 unlabelled", want: "Bank 021000021 unlabelled"},
		{name: "masks a labelled account number", in: "Account number: 12345678", want: "Account number: [ACCOUNT]"},
		{name: "masks an abbreviated account number", in: "Acct #000123456789", want: "Acct #[ACCOUNT]"},
		{name: "masks a grouped account number", in: "a/c 9876 5432 10", want: "a/c [ACCOUNT]"},
		{name: "keeps accounts payable wording", in: "Accounts payable terms: net 30", want: "Accounts payable terms: net 30"},
		{name: "keeps account used as a verb", in: "Please account for 30 minutes", want: "Please account for 30 minutes"},
		{name: "masks routing and account together", in: "Wire to routing 021000021, account 123456789012.", want: "Wire to routing [ROUTING], account [ACCOUNT]."},
		{name: "masks a spaced iban", in: "IBAN DE89 3704 0044 0532 0130 00 for payment", want: "IBAN [ACCOUNT] for payment"},
		{name: "masks an iban and keeps the word after it", in: "GB82 WEST 1234 5698 7654 32 EUR", want: "[ACCOUNT] EUR"},
		{name: "masks a compact iban", in: "Pay GB82WEST12345698765432 today", want: "Pay [ACCOUNT] today"},
		{name: "keeps ordinary freight numbers", in: "Order 12 shipped 2026-09-24 weighing 40000 lbs", want: "Order 12 shipped 2026-09-24 weighing 40000 lbs"},
		{name: "keeps text without digits", in: "Deliver to the north dock", want: "Deliver to the north dock"},
		{name: "keeps empty text", in: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := piiscrub.Scrub(tt.in)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, got, piiscrub.Scrub(got), "scrubbing twice changes nothing more")
		})
	}
}

func TestScrubAllKeepsOrderAndLength(t *testing.T) {
	t.Parallel()

	in := []string{"SSN 123-45-6789", "no digits here", "Card 4111111111111111"}
	got := piiscrub.ScrubAll(in)

	assert.Equal(t, []string{"SSN [SSN]", "no digits here", "Card [CARD]"}, got)
	assert.Equal(t, "SSN 123-45-6789", in[0], "the input slice is left alone")
}

func TestScrubAllHandlesNoTexts(t *testing.T) {
	t.Parallel()

	assert.Empty(t, piiscrub.ScrubAll(nil))
}

func TestScrubLeavesMasksWithoutDigits(t *testing.T) {
	t.Parallel()

	for _, mask := range []string{
		piiscrub.MaskSSN,
		piiscrub.MaskCard,
		piiscrub.MaskRouting,
		piiscrub.MaskAccount,
	} {
		assert.NotRegexp(t, `[0-9]`, mask)
	}
}
