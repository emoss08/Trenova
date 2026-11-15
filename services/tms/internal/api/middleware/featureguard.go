package middleware

import (
	"fmt"
	"net/http"

	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/gin-gonic/gin"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type FeatureGuardParams struct {
	fx.In

	Features     platformcatalogports.FeatureGate
	ErrorHandler *helpers.ErrorHandler
	Logger       *zap.Logger
}

type FeatureGuard struct {
	features     platformcatalogports.FeatureGate
	errorHandler *helpers.ErrorHandler
	l            *zap.Logger
}

func NewFeatureGuard(p FeatureGuardParams) *FeatureGuard {
	return &FeatureGuard{
		features:     p.Features,
		errorHandler: p.ErrorHandler,
		l:            p.Logger.Named("feature-guard"),
	}
}

type OffAnswer struct {
	status  int
	err     func() error
	respond gin.HandlerFunc
}

func AnswerStatus(status int) OffAnswer {
	return OffAnswer{status: status}
}

func AnswerError(err func() error) OffAnswer {
	return OffAnswer{err: err}
}

func AnswerPlanRestricted(feature platformcatalog.FeatureKey) OffAnswer {
	return AnswerError(func() error {
		addOn, _ := platformcatalog.AddOnFor(feature)
		return errortypes.NewFeatureRestrictionError(string(feature), addOn.Name)
	})
}

func AnswerWith(respond gin.HandlerFunc) OffAnswer {
	return OffAnswer{respond: respond}
}

func (g *FeatureGuard) RequireFeature(
	feature platformcatalog.FeatureKey,
	off OffAnswer,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		on, err := g.enabled(c, feature)
		if err != nil {
			g.l.Error("could not ask whether a route's feature is on the plan",
				zap.String("feature", string(feature)),
				zap.String("route", c.FullPath()),
				zap.Error(err))
			c.AbortWithStatus(http.StatusServiceUnavailable)
			return
		}
		if on {
			c.Next()
			return
		}

		g.answer(c, off)
		c.Abort()
	}
}

func (g *FeatureGuard) enabled(c *gin.Context, feature platformcatalog.FeatureKey) (bool, error) {
	if _, known := platformcatalog.Lookup(feature); !known {
		return false, fmt.Errorf("%w: %q", platformcatalogports.ErrUnknownFeature, feature)
	}
	return g.features.Enabled(c.Request.Context(), feature)
}

func (g *FeatureGuard) answer(c *gin.Context, off OffAnswer) {
	switch {
	case off.respond != nil:
		off.respond(c)
	case off.err != nil:
		g.errorHandler.HandleError(c, off.err())
	default:
		c.Status(off.status)
	}
}
