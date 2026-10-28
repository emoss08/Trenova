package factoringtransport

import (
	"context"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringsubmissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"go.uber.org/fx"
)

type EmailParams struct {
	fx.In

	Email emailports.EmailService
}

type Email struct {
	email emailports.EmailService
}

var _ factoringsubmissionports.FactoringTransport = (*Email)(nil)

func NewEmail(p EmailParams) factoringsubmissionports.FactoringTransport {
	return &Email{email: p.Email}
}

func (t *Email) Method() factoring.TransportMethod {
	return factoring.TransportMethodEmail
}

func (t *Email) Send(
	ctx context.Context,
	req *factoringsubmissionports.TransportSendRequest,
) (*factoringsubmissionports.TransportSendResult, error) {
	if err := checkSend(req); err != nil {
		return nil, err
	}
	if req.Render == nil {
		return nil, errortypes.NewBusinessError("The schedule message could not be prepared")
	}
	parts, err := PackAttachments(req.Files, emailports.MaxAttachmentBytes)
	if err != nil {
		return nil, err
	}

	references := make([]string, 0, len(parts))
	for idx, files := range parts {
		names := make([]string, 0, len(files))
		attachments := make([]emailports.EmailAttachment, 0, len(files))
		for _, file := range files {
			names = append(names, file.Name)
			attachments = append(attachments, emailports.EmailAttachment{
				FileName:    file.Name,
				ContentType: file.ContentType,
				Content:     file.Content,
				SizeBytes:   int64(len(file.Content)),
			})
		}
		message, renderErr := req.Render(idx+1, len(parts), names)
		if renderErr != nil {
			return nil, renderErr
		}
		sent, sendErr := t.email.Send(ctx, &emailports.SendEmailRequest{
			TenantInfo:     req.TenantInfo,
			Purpose:        email.PurposeBilling,
			To:             req.Transport.EmailRecipients,
			Subject:        message.Subject,
			HTML:           message.HTML,
			Text:           message.Text,
			Attachments:    attachments,
			IdempotencyKey: req.IdempotencyKey + "-part-" + strconv.Itoa(idx+1),
		})
		if sendErr != nil {
			return nil, sendErr
		}
		if sent != nil {
			references = append(references, sent.ID.String())
		}
	}

	return &factoringsubmissionports.TransportSendResult{
		Reference: strings.Join(references, ", "),
		Parts:     len(parts),
	}, nil
}

func (t *Email) Test(_ context.Context, req *factoringsubmissionports.TransportTestRequest) error {
	if req == nil || req.Transport == nil {
		return errortypes.NewBusinessError("There is no delivery setting to test")
	}
	multiErr := errortypes.NewMultiError()
	req.Transport.Validate(multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func PackAttachments(
	files []factoringsubmissionports.TransportFile,
	limit int64,
) ([][]factoringsubmissionports.TransportFile, error) {
	if len(files) == 0 {
		return nil, errortypes.NewBusinessError("The schedule has no files to send")
	}
	parts := make([][]factoringsubmissionports.TransportFile, 0, 1)
	current := make([]factoringsubmissionports.TransportFile, 0, len(files))
	var size int64
	for _, file := range files {
		fileSize := int64(len(file.Content))
		if fileSize > limit {
			return nil, errortypes.NewBusinessError(
				"{0} is too large to email ({1} bytes); send this factor's schedules by SFTP or without packets",
				file.Name,
				strconv.FormatInt(fileSize, 10),
			)
		}
		if size+fileSize > limit && len(current) > 0 {
			parts = append(parts, current)
			current = make([]factoringsubmissionports.TransportFile, 0, len(files))
			size = 0
		}
		current = append(current, file)
		size += fileSize
	}
	return append(parts, current), nil
}

func checkSend(req *factoringsubmissionports.TransportSendRequest) error {
	if req == nil || req.Submission == nil || req.Transport == nil {
		return errortypes.NewBusinessError("The schedule has nowhere to go")
	}
	if req.IdempotencyKey == "" {
		return errortypes.NewBusinessError("The schedule send has no idempotency key")
	}
	return nil
}
