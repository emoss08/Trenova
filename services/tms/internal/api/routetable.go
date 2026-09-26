package api

import (
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/shared/reflectutils"
	"github.com/gin-gonic/gin"
)

func RouteTable() (routes gin.RoutesInfo, err error) {
	var params RouterParams
	if err = reflectutils.AllocatePointers(&params); err != nil {
		return nil, err
	}

	engine := gin.New()
	params.Server = &Server{router: engine}
	params.Config = &config.Config{}

	defer func() {
		if recovered := recover(); recovered != nil {
			routes = nil
			err = fmt.Errorf("registering routes without dependencies panicked: %v", recovered)
		}
	}()

	NewRouter(params).setupRoutes()

	return engine.Routes(), nil
}
