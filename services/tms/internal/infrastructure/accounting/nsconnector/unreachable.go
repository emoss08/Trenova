package nsconnector

import (
	"errors"
	"net"

	"github.com/emoss08/trenova/shared/netsuiteerrors"
)

func isUnreachable(err error) bool {
	var dnsErr *net.DNSError
	return netsuiteerrors.IsNotFound(err) || errors.As(err, &dnsErr)
}
