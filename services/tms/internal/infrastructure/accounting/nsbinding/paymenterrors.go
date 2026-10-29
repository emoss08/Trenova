package nsbinding

import (
	"errors"
)

var ErrApplyTarget = errors.New("netsuite: an applied amount needs the document it pays")
