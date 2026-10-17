package documentextractionjobs

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/documents/domain/extractioneval"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/extractionevalports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

var _ extractionevalports.ExtractionCaseConverter = (*CaseConverter)(nil)

type CaseConverter struct {
	activities *Activities
}

func NewCaseConverter(a *Activities) extractionevalports.ExtractionCaseConverter {
	return &CaseConverter{activities: a}
}

func (c *CaseConverter) ConvertCaseDocument(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	documentID pulid.ID,
) (*extractionevalports.ConvertedCasePages, error) {
	a := c.activities
	doc, err := a.documentRepo.GetByID(ctx, documentports.GetDocumentByIDRequest{
		ID:         documentID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	control, err := a.getDocumentControl(ctx, doc.OrganizationID, doc.BusinessUnitID)
	if err != nil {
		return nil, err
	}

	switch conversionFormatOf(doc.FileType, doc.OriginalName) {
	case formatLegacyOffice, formatUnsupported:
		return nil, fmt.Errorf(
			"%w: %s",
			kernelports.ErrDocumentNotConvertible,
			unsupportedFormatMessage(
				conversionFormatOf(doc.FileType, doc.OriginalName),
				doc.FileType,
			),
		)
	case formatPlainText, formatPDF, formatImage, formatOffice, formatEmail:
	}

	data, failure := a.readDocumentBytes(ctx, doc)
	if failure != nil {
		return nil, fmt.Errorf("%w: %s", kernelports.ErrDocumentNotConvertible, failure.message)
	}

	converted, failure, err := a.convertDocumentBytes(ctx, doc, data, control.EnableOCR)
	if err != nil {
		return nil, err
	}
	if failure != nil {
		return nil, fmt.Errorf("%w: %s", kernelports.ErrDocumentNotConvertible, failure.message)
	}

	pages := make([]extractioneval.Page, 0, len(converted.pages))
	for _, page := range converted.pages {
		if strings.TrimSpace(page.Markdown) != "" {
			pages = append(pages, extractioneval.Page{
				Number:   page.PageNumber,
				Text:     page.Markdown,
				Markdown: true,
			})
			continue
		}
		if strings.TrimSpace(page.Text) != "" {
			pages = append(pages, extractioneval.Page{Number: page.PageNumber, Text: page.Text})
		}
	}

	return &extractionevalports.ConvertedCasePages{
		Pages:            pages,
		ConverterVersion: converted.converterVersion,
	}, nil
}
