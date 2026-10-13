package factoringcompany

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
)

func validFactor() *FactoringCompany {
	return &FactoringCompany{
		Status:       domaintypes.StatusActive,
		Code:         "RTS",
		Name:         "RTS Financial",
		AddressLine1: "PO Box 840267",
		City:         "Dallas",
		StateID:      "st_1",
		PostalCode:   "75284",
	}
}

func TestNoticeFallsBackToTheStandardWording(t *testing.T) {
	t.Parallel()

	factor := validFactor()
	assert.Contains(t, factor.Notice(), "payable only to, RTS Financial.")

	factor.NoticeOfAssignment = "  Pay RTS only.  "
	assert.Equal(t, "Pay RTS only.", factor.Notice())

	var none *FactoringCompany
	assert.Empty(t, none.Notice())
}

func TestRemitLinesPrintAddressInstructionsAndContact(t *testing.T) {
	t.Parallel()

	factor := validFactor()
	factor.State = &usstate.UsState{Abbreviation: "TX", CountryName: "United States"}
	factor.RemittanceInstructions = "ACH routing 111000614\n\nAccount 8842091"
	factor.Email = "pay@rts.test"

	assert.Equal(t, []string{
		"PO Box 840267",
		"Dallas, TX 75284",
		"United States",
		"ACH routing 111000614",
		"Account 8842091",
		"Email: pay@rts.test",
	}, factor.RemitLines())
	assert.True(t, strings.HasPrefix(factor.RemittanceText(), "Remit to RTS Financial\nPO Box 840267\n"))
}

func TestValidateRefusesABadEmailAndOverlongText(t *testing.T) {
	t.Parallel()

	factor := validFactor()
	multiErr := errortypes.NewMultiError()
	factor.Validate(multiErr)
	assert.False(t, multiErr.HasErrors(), "%v", multiErr)

	factor.Email = "not-an-email"
	factor.NoticeOfAssignment = string(make([]byte, MaxNoticeOfAssignment+1))
	multiErr = errortypes.NewMultiError()
	factor.Validate(multiErr)
	fields := map[string]bool{}
	for _, fieldErr := range multiErr.Errors {
		fields[fieldErr.Field] = true
	}
	assert.True(t, fields["email"])
	assert.True(t, fields["noticeOfAssignment"])
}

func TestIsActive(t *testing.T) {
	t.Parallel()

	factor := validFactor()
	assert.True(t, factor.IsActive())
	factor.Status = domaintypes.StatusInactive
	assert.False(t, factor.IsActive())
}
