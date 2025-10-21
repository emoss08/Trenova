package email

import "errors"

var (
	ErrTemplateNotFound            = errors.New("template not found")
	ErrHTMLTemplateNil             = errors.New("HTML template is nil")
	ErrTextTemplateNil             = errors.New("text template is nil")
	ErrTooManyAttachments          = errors.New("too many attachments")
	ErrAttachmentFileNameEmpty     = errors.New("attachment file name is empty")
	ErrTotalAttachmentSizeExceeded = errors.New("total attachment size exceeds limit")
)
