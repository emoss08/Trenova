package datausageports

import (
	"context"
	"errors"
	"time"
)

const SourceGroup = `group:"managed_data_sources"`

var ErrNotMeasured = errors.New("this instance does not measure this figure")

type Figure string

const (
	FigureDatabase  Figure = "database"
	FigureDocuments Figure = "documents"
	FigureSearch    Figure = "search"
)

func (f Figure) IsValid() bool {
	switch f {
	case FigureDatabase, FigureDocuments, FigureSearch:
		return true
	default:
		return false
	}
}

type Source interface {
	Figure() Figure
	Timeout() time.Duration
	Measure(ctx context.Context) (int64, error)
}

type Usage struct {
	MeasuredAt     int64
	DatabaseBytes  *int64
	DocumentsBytes *int64
	SearchBytes    *int64
}

type Reader interface {
	Usage(ctx context.Context) *Usage
}
