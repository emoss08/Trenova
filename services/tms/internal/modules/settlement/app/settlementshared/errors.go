package settlementshared

import "errors"

var ErrPeriodBatchClosed = errors.New("the settlement batch for this pay period is no longer open")
