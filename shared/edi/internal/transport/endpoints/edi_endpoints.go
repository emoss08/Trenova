package endpoints

import (
	"context"

	"github.com/emoss08/trenova/shared/edi/internal/core/domain"
	"github.com/emoss08/trenova/shared/edi/internal/core/services"
	"github.com/go-kit/kit/endpoint"
)

// ProcessEDIRequest represents the request for processing an EDI document
type ProcessEDIRequest struct {
	PartnerID  string `json:"partner_id"`
	EDIContent string `json:"edi_content"`
}

// ProcessEDIResponse represents the response from processing an EDI document
type ProcessEDIResponse struct {
	Document *domain.EDIDocument `json:"document,omitempty"`
	Error    string              `json:"error,omitempty"`
}

// MakeProcessEDIEndpoint creates an endpoint for processing EDI documents
func MakeProcessEDIEndpoint(svc *services.EDIProcessorService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(ProcessEDIRequest)
		doc, err := svc.ProcessEDIDocument(ctx, req.PartnerID, req.EDIContent)
		if err != nil {
			return ProcessEDIResponse{Error: err.Error()}, nil
		}
		return ProcessEDIResponse{Document: doc}, nil
	}
}

// GetDocumentRequest represents the request for getting an EDI document
type GetDocumentRequest struct {
	DocumentID string `json:"document_id"`
}

// GetDocumentResponse represents the response for getting an EDI document
type GetDocumentResponse struct {
	Document *domain.EDIDocument `json:"document,omitempty"`
	Error    string              `json:"error,omitempty"`
}

// MakeGetDocumentEndpoint creates an endpoint for retrieving EDI documents
func MakeGetDocumentEndpoint(svc *services.EDIProcessorService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GetDocumentRequest)
		doc, err := svc.GetDocumentByID(ctx, req.DocumentID)
		if err != nil {
			return GetDocumentResponse{Error: err.Error()}, nil
		}
		return GetDocumentResponse{Document: doc}, nil
	}
}

// ListDocumentsRequest represents the request for listing EDI documents
type ListDocumentsRequest struct {
	PartnerID string `json:"partner_id"`
	Limit     int    `json:"limit"`
	Offset    int    `json:"offset"`
}

// ListDocumentsResponse represents the response for listing EDI documents
type ListDocumentsResponse struct {
	Documents []*domain.EDIDocument `json:"documents,omitempty"`
	Count     int                   `json:"count"`
	Error     string                `json:"error,omitempty"`
}

// MakeListDocumentsEndpoint creates an endpoint for listing EDI documents
func MakeListDocumentsEndpoint(svc *services.EDIProcessorService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(ListDocumentsRequest)
		docs, err := svc.ListDocuments(ctx, req.PartnerID, req.Limit, req.Offset)
		if err != nil {
			return ListDocumentsResponse{Error: err.Error()}, nil
		}
		return ListDocumentsResponse{Documents: docs, Count: len(docs)}, nil
	}
}

// Endpoints collects all of the endpoints that compose an EDI service
type Endpoints struct {
	ProcessEDIEndpoint    endpoint.Endpoint
	GetDocumentEndpoint   endpoint.Endpoint
	ListDocumentsEndpoint endpoint.Endpoint
}

// NewEndpoints returns an Endpoints struct where each endpoint invokes
// the corresponding method on the provided service
func NewEndpoints(svc *services.EDIProcessorService) Endpoints {
	return Endpoints{
		ProcessEDIEndpoint:    MakeProcessEDIEndpoint(svc),
		GetDocumentEndpoint:   MakeGetDocumentEndpoint(svc),
		ListDocumentsEndpoint: MakeListDocumentsEndpoint(svc),
	}
}

// Chain applies a list of middlewares to all endpoints
func (e Endpoints) Chain(mw ...endpoint.Middleware) Endpoints {
	for _, m := range mw {
		e.ProcessEDIEndpoint = m(e.ProcessEDIEndpoint)
		e.GetDocumentEndpoint = m(e.GetDocumentEndpoint)
		e.ListDocumentsEndpoint = m(e.ListDocumentsEndpoint)
	}
	return e
}

