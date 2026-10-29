package accountingconnectioncert

import (
	"regexp"
	"strings"

	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingconnectionkit"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/pkg/errortypes"
)

const (
	certificateSubjectPrefix  = "Trenova "
	maxCertificateSubjectRune = 64
)

var certificateIDPattern = regexp.MustCompile(`^[A-Za-z0-9._~+/=-]{1,100}$`)

func validateCertificateID(field, certificateID string) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	if strings.TrimSpace(certificateID) == "" {
		multiErr.Add(field, errortypes.ErrRequired, "Certificate ID is required")
	} else if !certificateIDPattern.MatchString(certificateID) {
		multiErr.Add(
			field,
			errortypes.ErrInvalid,
			"Certificate ID is not valid. Copy it from the certificate's row in the mapping.",
		)
	}
	return multiErr
}

func errCertificateRefused(provider string, err error) error {
	return errortypes.NewBusinessError(
		"{0} did not accept the certificate. Check that it is uploaded to the mapping for Trenova's application, that the mapping's role can use REST web services, and that the certificate ID and account ID are right.",
		provider,
	).WithInternal(err)
}

type certificateRenewal struct {
	req   *accountingconnectionports.RenewAccountingCertificateRequest
	name  string
	held  *accountingconnectionkit.HeldCertificate
	grant *accountingconnectorports.AccountingTokenGrant
	now   int64
}

type renewedCertificate struct {
	conn     *accountingconnection.AccountingConnection
	previous map[string]any
	before   accountingconnection.ConnectionStatus
}
