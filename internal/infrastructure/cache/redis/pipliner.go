/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package redis

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports/infra"
	"github.com/redis/go-redis/v9"
)

type Pipeliner struct {
	pipe redis.Pipeliner
}

func (p *Pipeliner) Exec(ctx context.Context) error {
	_, err := p.pipe.Exec(ctx)
	return err
}

func (p *Pipeliner) Queue(cmd infra.CacheCommand) {
	p.pipe.Do(context.Background(), cmd.Name(), cmd.Args())
}
