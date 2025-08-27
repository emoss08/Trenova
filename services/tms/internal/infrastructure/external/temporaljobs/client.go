/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package temporaljobs

import (
	"github.com/emoss08/trenova/internal/pkg/config"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
)

type ClientParams struct {
	fx.In

	Config *config.Manager
}

func NewClient(p ClientParams) (client.Client, error) {
	cfg := p.Config.Temporal()

	return client.Dial(client.Options{
		HostPort:  cfg.HostPort,
		Namespace: cfg.Namespace,
	})
}
