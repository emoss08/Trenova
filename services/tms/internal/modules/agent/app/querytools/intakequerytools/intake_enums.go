package intakequerytools

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratequoteports"
)

var (
	shopStrategies = agenttoolschema.Source("carrierShop.strategy", []ratequoteports.ShopStrategy{
		ratequoteports.ShopStrategyLeastCost,
		ratequoteports.ShopStrategyBestMargin,
		ratequoteports.ShopStrategyGuideRank,
		ratequoteports.ShopStrategyFastestAccept,
	})
)
