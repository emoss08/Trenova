package deskbench

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/deskbench/benchscenario"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentupload"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	attachmentFormField  = "file"
	attachmentFormMemory = 64 << 20
	attachmentReadWithin = 2 * time.Minute
	attachmentReadEvery  = time.Second
)

var ErrAttachmentUnreadable = errors.New(
	"the server cannot read attached files, so a tool that reads one has nothing to read",
)

type attachmentSource struct {
	name        string
	contentType string
	body        []byte
}

func (c *Conversation) attach(ctx context.Context, refs []string) ([]pulid.ID, error) {
	if len(refs) == 0 {
		return nil, nil
	}

	ids := make([]pulid.ID, 0, len(refs))
	for _, ref := range refs {
		source, err := c.attachmentSource(ctx, strings.TrimSpace(ref))
		if err != nil {
			return nil, fmt.Errorf("attach %s: %w", ref, err)
		}
		id, err := c.upload(ctx, source)
		if err != nil {
			return nil, fmt.Errorf("attach %s: %w", ref, err)
		}
		if err = c.awaitReading(ctx, id); err != nil {
			return nil, fmt.Errorf("attach %s: %w", ref, err)
		}
		ids = append(ids, id)
	}

	return ids, nil
}

func (c *Conversation) attachmentSource(
	ctx context.Context,
	ref string,
) (*attachmentSource, error) {
	if strings.HasPrefix(ref, benchscenario.DocumentIDPrefix) {
		id, err := pulid.Parse(ref)
		if err != nil {
			return nil, err
		}

		return c.copyOfDocument(ctx, id)
	}

	body, err := os.ReadFile(ref)
	if err != nil {
		return nil, err
	}
	contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(ref)))
	if contentType == "" {
		contentType = http.DetectContentType(body)
	}

	return &attachmentSource{name: filepath.Base(ref), contentType: contentType, body: body}, nil
}

func (c *Conversation) copyOfDocument(ctx context.Context, id pulid.ID) (*attachmentSource, error) {
	content, err := c.session.bench.Documents.GetDownloadContent(ctx,
		documentports.GetDocumentByIDRequest{ID: id, TenantInfo: c.session.Tenant},
	)
	if err != nil {
		return nil, err
	}
	defer content.Body.Close()

	body, err := io.ReadAll(content.Body)
	if err != nil {
		return nil, err
	}

	return &attachmentSource{
		name:        content.Document.OriginalName,
		contentType: content.ContentType,
		body:        body,
	}, nil
}

func (c *Conversation) upload(ctx context.Context, source *attachmentSource) (pulid.ID, error) {
	file, err := formFile(source)
	if err != nil {
		return pulid.Nil, err
	}

	result, err := c.session.bench.Documents.Upload(ctx, &documentports.UploadRequest{
		TenantInfo:        c.session.Tenant,
		Actor:             c.session.Actor,
		File:              file,
		ResourceID:        c.Thread.ID.String(),
		ResourceType:      document.ConversationResourceType,
		ProcessingProfile: string(document.ProcessingProfileAssistantAttachment),
	})
	if err != nil {
		return pulid.Nil, err
	}

	return result.Document.ID, nil
}

func (c *Conversation) awaitReading(ctx context.Context, id pulid.ID) error {
	deadline := time.Now().Add(attachmentReadWithin)
	for {
		content, err := c.session.bench.Contents.GetContent(ctx, id, c.session.Tenant)
		if err == nil && content != nil && content.Status.Settled() {
			if !c.NeedsText {
				return nil
			}
			return unreadableHere(content)
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("document %s was not read within %s", id, attachmentReadWithin)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(attachmentReadEvery):
		}
	}
}

func formFile(source *attachmentSource) (*multipart.FileHeader, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", fmt.Sprintf(
		`form-data; name=%q; filename=%q`, attachmentFormField, source.name,
	))
	header.Set("Content-Type", source.contentType)
	part, err := writer.CreatePart(header)
	if err != nil {
		return nil, err
	}
	if _, err = part.Write(source.body); err != nil {
		return nil, err
	}
	if err = writer.Close(); err != nil {
		return nil, err
	}

	form, err := multipart.NewReader(&buf, writer.Boundary()).ReadForm(attachmentFormMemory)
	if err != nil {
		return nil, err
	}
	files := form.File[attachmentFormField]
	if len(files) == 0 {
		return nil, fmt.Errorf("the form carries no %s", attachmentFormField)
	}

	return files[0], nil
}

func unreadableHere(content *documentcontent.Content) error {
	switch documentupload.FailureCode(content.FailureCode) {
	case documentupload.FailureConverterUnavailable, documentupload.FailureExtractionUnavailable:
		return fmt.Errorf("%w: %s", ErrAttachmentUnreadable, content.FailureMessage)
	default:
		return nil
	}
}
