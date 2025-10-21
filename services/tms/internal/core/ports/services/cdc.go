package services

import (
	"context"

	"github.com/emoss08/trenova/pkg/cdctypes"
)

type CDCService interface {
	Start() error
	Stop() error
	IsRunning() bool
	RegisterHandler(table string, handler CDCEventHandler)
}

type CDCEventHandler interface {
	HandleEvent(ctx context.Context, event *cdctypes.CDCEvent) error
	GetTableName() string
}
