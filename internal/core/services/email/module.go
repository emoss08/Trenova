/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package email

import (
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/email/providers"
	"go.uber.org/fx"
)

var Module = fx.Module("email",
	// Include the providers module
	providers.Module,

	// Provide internal components
	fx.Provide(
		NewClientFactory,
		NewMessageBuilder,
		NewSender,
		NewQueueProcessor,
		NewAttachmentHandler,
		NewBackgroundEmailService,
	),

	// Provide service interfaces
	fx.Provide(
		fx.Annotate(
			NewService,
			fx.As(new(services.EmailService)),
		),
		fx.Annotate(
			NewProfileService,
			fx.As(new(services.EmailProfileService)),
		),
		fx.Annotate(
			NewTemplateService,
			fx.As(new(services.EmailTemplateService)),
		),
		fx.Annotate(
			NewQueueService,
			fx.As(new(services.EmailQueueService)),
		),
		fx.Annotate(
			NewLogService,
			fx.As(new(services.EmailLogService)),
		),
	),
)
