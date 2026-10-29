package documenttemplate

import (
	"github.com/emoss08/trenova/pkg/templateengine"
)

// Channel names one output slot of a template version.
type Channel string

const (
	ChannelPDF               = Channel("PDF")
	ChannelEmailHTML         = Channel("EmailHTML")
	ChannelEmailText         = Channel("EmailText")
	ChannelSubject           = Channel("Subject")
	ChannelNotificationTitle = Channel("NotificationTitle")
	ChannelNotificationBody  = Channel("NotificationBody")
)

func (c Channel) String() string { return string(c) }

// Engine maps a domain channel to the engine's, so the engine stays free of
// domain imports and the domain owns the enum the database and GraphQL see.
func (c Channel) Engine() templateengine.Channel {
	return templateengine.Channel(c)
}

// Field names the editor pane and the database column a channel is stored in.
func (c Channel) Field() string {
	switch c {
	case ChannelPDF, ChannelEmailHTML:
		return "bodyHtml"
	case ChannelEmailText:
		return "bodyText"
	case ChannelSubject:
		return "subject"
	case ChannelNotificationTitle:
		return "notificationTitle"
	case ChannelNotificationBody:
		return "notificationMessage"
	default:
		return string(c)
	}
}
