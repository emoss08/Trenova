package invoicehandler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type sharedDocumentStub struct {
	invoiceports.InvoiceSendService
}

func (sharedDocumentStub) DownloadSharedDocument(
	context.Context,
	*invoiceports.DownloadInvoiceDocumentRequest,
) (*invoiceports.DownloadInvoiceDocumentResult, error) {
	return &invoiceports.DownloadInvoiceDocumentResult{
		ContentType:        "application/pdf",
		ContentDisposition: `attachment; filename="pod.pdf"`,
		Body:               []byte("%PDF-1.7"),
	}, nil
}

func TestDownloadSharedDocument_IsNeverCachedOrReferred(t *testing.T) {
	t.Parallel()

	gin.SetMode(gin.TestMode)
	handler := &Handler{sending: sharedDocumentStub{}}
	router := gin.New()
	router.GET("/shared-documents/:token/download/", handler.downloadSharedDocument)

	req := httptest.NewRequest(http.MethodGet, "/shared-documents/a-token/download/", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
	assert.Equal(t, "no-referrer", rec.Header().Get("Referrer-Policy"))
	assert.Equal(t, `attachment; filename="pod.pdf"`, rec.Header().Get("Content-Disposition"))
	assert.Equal(t, "%PDF-1.7", rec.Body.String())
}
