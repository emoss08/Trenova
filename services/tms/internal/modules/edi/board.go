package edi

import (
	"github.com/emoss08/trenova/internal/modules/edi/adapters/postgres/ediboardreader"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/boardreaderports"
	"go.uber.org/fx"
)

var boardReaders = fx.Options(
	fx.Provide(
		fx.Annotate(
			ediboardreader.NewInboundFiles,
			fx.As(new(boardreaderports.RecordReader)),
			fx.ResultTags(boardreaderports.RecordReaders),
		),
	),
	fx.Provide(
		fx.Annotate(
			ediboardreader.NewFileWaiting,
			fx.As(new(boardreaderports.MeasureReader)),
			fx.ResultTags(boardreaderports.MeasureReaders),
		),
	),
	fx.Provide(
		fx.Annotate(
			ediboardreader.NewMessages,
			fx.As(new(boardreaderports.RecordReader)),
			fx.ResultTags(boardreaderports.RecordReaders),
		),
	),
	fx.Provide(
		fx.Annotate(
			ediboardreader.NewMessageWaiting,
			fx.As(new(boardreaderports.MeasureReader)),
			fx.ResultTags(boardreaderports.MeasureReaders),
		),
	),
	fx.Provide(
		fx.Annotate(
			ediboardreader.NewPartnerDay,
			fx.As(new(boardreaderports.MetricReader)),
			fx.ResultTags(boardreaderports.MetricReaders),
		),
	),
)
