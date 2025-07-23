// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package api

import (
	"github.com/emoss08/trenova/internal/api/server"
	"go.uber.org/fx"
)

var ServerModule = fx.Module("api.Server", fx.Provide(server.NewServer))
