package planservice

import "errors"

var ErrTenantRequired = errors.New("plan resolution requires an organization and business unit")
