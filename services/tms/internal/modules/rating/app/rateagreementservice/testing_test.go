package rateagreementservice

import (
	"github.com/emoss08/trenova/internal/modules/rating/ports/tariffports"
	"github.com/emoss08/trenova/shared/timeutils"
)

func NewTestValidator(resolver tariffports.Resolver, checker tariffports.Checker) *Validator {
	tariffs := &tariffInputs{resolver: resolver, checker: checker, now: timeutils.NowUnix}
	return &Validator{
		validator: newBuilder(nil, tariffs).Build(),
		tariffs:   tariffs,
	}
}
