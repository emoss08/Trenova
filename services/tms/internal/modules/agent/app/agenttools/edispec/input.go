package edispec

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediconnectiondraftports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

var (
	ErrNoConversation = errors.New(
		"a partner's spec is read from a file attached to this conversation, and this run has none",
	)
	ErrStillReading = errors.New(
		"the spec is still being read; tell the person and draft it again in a moment",
	)
)

type Input struct {
	DocumentID pulid.ID
	Digest     string
	Draft      *ediconnectiondraftports.ConnectionDraftRequest
}

type rawInput struct {
	DocumentID   string                                          `json:"documentId"`
	PartnerID    string                                          `json:"ediPartnerId"`
	CustomerID   string                                          `json:"customerId"`
	Digest       string                                          `json:"documentDigest"`
	Values       []ediconnectiondraftports.DraftValueInput       `json:"values"`
	Requirements []ediconnectiondraftports.DraftRequirementInput `json:"requirements"`
	Mappings     []ediconnectiondraftports.DraftMappingInput     `json:"mappings"`
	Gaps         []ediconnectiondraftports.DraftGapInput         `json:"gaps"`
	Resolutions  []ediconnectiondraftports.DraftResolutionInput  `json:"resolutions"`
}

func Parse(params map[string]any, tenant pagination.TenantInfo) (*Input, error) {
	encoded, err := sonic.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("read the draft's parameters: %w", err)
	}
	var raw rawInput
	if err = sonic.Unmarshal(encoded, &raw); err != nil {
		return nil, fmt.Errorf("the draft's parameters do not match the schema: %w", err)
	}

	documentID, err := requiredID(ParamDocumentID, raw.DocumentID)
	if err != nil {
		return nil, err
	}
	partnerID, err := optionalID(ParamPartnerID, raw.PartnerID)
	if err != nil {
		return nil, err
	}
	customerID, err := optionalID(ParamCustomerID, raw.CustomerID)
	if err != nil {
		return nil, err
	}

	return &Input{
		DocumentID: documentID,
		Digest:     strings.TrimSpace(raw.Digest),
		Draft: &ediconnectiondraftports.ConnectionDraftRequest{
			TenantInfo:   tenant,
			PartnerID:    partnerID,
			CustomerID:   customerID,
			Values:       raw.Values,
			Requirements: raw.Requirements,
			Mappings:     raw.Mappings,
			Gaps:         raw.Gaps,
			Resolutions:  raw.Resolutions,
		},
	}, nil
}

func requiredID(key, value string) (pulid.ID, error) {
	id, err := optionalID(key, value)
	if err != nil {
		return pulid.Nil, err
	}
	if id.IsNil() {
		return pulid.Nil, fmt.Errorf("parameter %q is required", key)
	}
	return id, nil
}

func optionalID(key, value string) (pulid.ID, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return pulid.Nil, nil
	}
	id, err := pulid.Parse(value)
	if err != nil {
		return pulid.Nil, fmt.Errorf("parameter %q is not a valid id: %w", key, err)
	}
	return id, nil
}

type documentGetter interface {
	GetByID(
		ctx context.Context,
		req documentports.GetDocumentByIDRequest,
	) (*document.Document, error)
}

type contentReader interface {
	GetContent(
		ctx context.Context,
		documentID pulid.ID,
		tenantInfo pagination.TenantInfo,
	) (*documentcontent.Content, error)
}

type DocumentReader struct {
	documents documentGetter
	contents  contentReader
}

func NewDocumentReader(documents documentGetter, contents contentReader) *DocumentReader {
	return &DocumentReader{documents: documents, contents: contents}
}

type ReadRequest struct {
	Tenant     pagination.TenantInfo
	DocumentID pulid.ID
	ThreadID   pulid.ID
	AttachedBy pulid.ID
}

func (r *DocumentReader) Read(
	ctx context.Context,
	req *ReadRequest,
) (*ediconnectiondraftports.SpecDocument, error) {
	if req.ThreadID.IsNil() {
		return nil, ErrNoConversation
	}
	doc, err := r.documents.GetByID(ctx, documentports.GetDocumentByIDRequest{
		ID:         req.DocumentID,
		TenantInfo: req.Tenant,
	})
	if err != nil {
		if dberror.IsNotFoundError(err) || errortypes.IsNotFoundError(err) {
			return nil, fmt.Errorf("document %s does not exist", req.DocumentID)
		}
		return nil, err
	}
	attached := doc.AttachedTo(req.ThreadID)
	if req.AttachedBy.IsNotNil() {
		attached = doc.AttachedBy(req.ThreadID, req.AttachedBy)
	}
	if !attached {
		return nil, fmt.Errorf(
			"document %s was not attached to this conversation by the person you are working for",
			req.DocumentID,
		)
	}

	content, err := r.contents.GetContent(ctx, doc.ID, req.Tenant)
	if err != nil {
		if dberror.IsNotFoundError(err) || errortypes.IsNotFoundError(err) {
			return nil, ErrStillReading
		}
		return nil, err
	}
	switch {
	case !content.Status.Settled():
		return nil, ErrStillReading
	case content.Status == documentcontent.StatusFailed:
		return nil, fmt.Errorf(
			"the spec could not be read (%s); ask the person for a text PDF, a DOCX or the sample X12 as text",
			strings.TrimSpace(content.FailureMessage),
		)
	}

	pages := make([]ediconnectiondraftports.SpecPage, 0, len(content.Pages))
	for _, page := range content.Pages {
		if page == nil {
			continue
		}
		pages = append(
			pages,
			ediconnectiondraftports.SpecPage{Number: page.PageNumber, Text: page.ExtractedText},
		)
	}
	if len(pages) == 0 && strings.TrimSpace(content.ContentText) != "" {
		pages = append(
			pages,
			ediconnectiondraftports.SpecPage{Number: 1, Text: content.ContentText},
		)
	}
	return &ediconnectiondraftports.SpecDocument{
		ID:       doc.ID,
		FileName: doc.OriginalName,
		Pages:    pages,
	}, nil
}
