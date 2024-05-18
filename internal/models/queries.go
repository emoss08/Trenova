package models

import (
	"github.com/emoss08/trenova/internal/ent"
	"github.com/rs/zerolog"
)

type QueryService struct {
	Client *ent.Client
	Logger *zerolog.Logger
}

func NewQueryService(c *ent.Client, l *zerolog.Logger) *QueryService {
	return &QueryService{
		Client: c,
		Logger: l,
	}
}
