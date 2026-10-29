package nsbinding

import (
	"errors"
)

var ErrCertificateRevoked = errors.New(
	"netsuite refused the certificate: the client credentials mapping is missing or was " +
		"cleared, for example by a sandbox refresh",
)
