package invoiceservice

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"net/url"
	"strings"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	servicesports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/fileutils"
	"github.com/emoss08/trenova/shared/timeutils"
)

func (s *Service) DownloadSharedDocument(
	ctx context.Context,
	req *servicesports.DownloadInvoiceDocumentRequest,
) (*servicesports.DownloadInvoiceDocumentResult, error) {
	if s.documentService == nil {
		return nil, errortypes.NewBusinessError("Document service is not configured")
	}
	tokenHash := tokenHash(req.Token)
	share, err := s.repo.GetDocumentShareToken(
		ctx,
		repositories.GetInvoiceDocumentShareTokenRequest{
			TokenHash: tokenHash,
		},
	)
	if err != nil {
		return nil, err
	}
	now := timeutils.NowUnix()
	if share.RevokedAt != nil || share.ExpiresAt <= now {
		return nil, errortypes.NewNotFoundError("Document link is no longer available")
	}
	content, err := s.documentService.GetDownloadContent(ctx, repositories.GetDocumentByIDRequest{
		ID: share.DocumentID,
		TenantInfo: pagination.TenantInfo{
			OrgID: share.OrganizationID,
			BuID:  share.BusinessUnitID,
		},
	})
	if err != nil {
		return nil, err
	}
	body, err := readDocumentBody(content.Body, "shared document")
	if err != nil {
		return nil, err
	}

	share.DownloadedAt = &now
	if _, err = s.repo.UpdateDocumentShareToken(ctx, share); err != nil {
		return nil, err
	}

	return &servicesports.DownloadInvoiceDocumentResult{
		FileName:      content.Document.OriginalName,
		ContentType:   content.ContentType,
		ContentLength: int64(len(body)),
		ContentDisposition: fileutils.ContentDisposition(
			"attachment",
			content.Document.OriginalName,
		),
		Body: body,
	}, nil
}

func newShareToken() (string, string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}
	raw := base64.RawURLEncoding.EncodeToString(buf)
	return raw, tokenHash(raw), nil
}

func tokenHash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func signedDocumentURL(baseURL, token string) string {
	baseURL = strings.TrimRight(baseURL, "/")
	if baseURL == "" {
		return "/api/v1/billing/invoices/shared-documents/" + url.PathEscape(token) + "/download/"
	}
	return baseURL + "/api/v1/billing/invoices/shared-documents/" + url.PathEscape(
		token,
	) + "/download/"
}
