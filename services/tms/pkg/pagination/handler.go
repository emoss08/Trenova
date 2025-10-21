package pagination

import (
	authctx "github.com/emoss08/trenova/internal/api/context"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Handler[T any] struct {
	ctx          *gin.Context
	authCtx      *authctx.AuthContext
	errorHandler *helpers.ErrorHandler
	queryOpts    *QueryOptions
	extraParams  any
	debug        bool
	logger       *zap.Logger
}

func Handle[T any](c *gin.Context, authCtx *authctx.AuthContext) *Handler[T] {
	return &Handler[T]{
		ctx:     c,
		authCtx: authCtx,
		logger:  zap.L().Named("pagination-handler"),
		queryOpts: &QueryOptions{
			TenantOpts: TenantOptions{
				OrgID:  authCtx.OrganizationID,
				BuID:   authCtx.BusinessUnitID,
				UserID: authCtx.UserID,
			},
			Limit:  20, // Default limit
			Offset: 0,  // Default offset
		},
	}
}

func (ph *Handler[T]) WithDebug(debug bool) *Handler[T] {
	ph.debug = debug
	return ph
}

func (ph *Handler[T]) WithErrorHandler(eh *helpers.ErrorHandler) *Handler[T] {
	ph.errorHandler = eh
	return ph
}

func (ph *Handler[T]) WithExtraParams(params any) *Handler[T] {
	ph.extraParams = params
	if err := ph.ctx.ShouldBindQuery(params); err != nil && ph.errorHandler != nil {
		ph.errorHandler.HandleError(ph.ctx, err)
	}
	return ph
}

func (ph *Handler[T]) Execute(
	handler func(c *gin.Context, opts *QueryOptions) (*ListResult[T], error),
) {
	if err := ph.ctx.ShouldBindQuery(ph.queryOpts); err != nil {
		if ph.errorHandler != nil {
			ph.errorHandler.HandleError(ph.ctx, err)
			return
		}

		return
	}

	parseFilters(ph.ctx, ph.queryOpts)
	parseSort(ph.ctx, ph.queryOpts)

	result, err := handler(ph.ctx, ph.queryOpts)
	if err != nil {
		if ph.errorHandler != nil {
			ph.errorHandler.HandleError(ph.ctx, err)
		}
		return
	}

	nextURL := GetNextPageURL(ph.ctx, ph.queryOpts.Limit, ph.queryOpts.Offset, result.Total)
	prevURL := GetPrevPageURL(ph.ctx, ph.queryOpts.Limit, ph.queryOpts.Offset)

	if ph.debug {
		ph.logger.Debug(
			"Final Results",
			zap.Any("query options", ph.queryOpts),
			zap.Any("extra params", ph.extraParams),
			zap.String("next URL", nextURL),
			zap.String("prev URL", prevURL),
		)
	}

	ph.ctx.JSON(200, Response[[]T]{
		Count:   result.Total,
		Results: result.Items,
		Next:    nextURL,
		Prev:    prevURL,
	})
}

func (ph *Handler[T]) ExecuteWithHandler(handler PageableHandler[T]) {
	ph.Execute(func(c *gin.Context, opts *QueryOptions) (*ListResult[T], error) {
		return handler(c, opts)
	})
}
