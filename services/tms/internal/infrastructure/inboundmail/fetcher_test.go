package inboundmail

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testEmailID = "4ef9a417-02e9-4d39-ad75-9611e0fcc33c"
	testAPIKey  = "re_test_123"
)

type resendStub struct {
	server      *httptest.Server
	emailStatus int
	emailBody   string
	listStatus  int
	fileStatus  int
	fileBody    []byte
	listCalls   atomic.Int32
	authHeaders []string
}

func newResendStub(t *testing.T) *resendStub {
	t.Helper()

	stub := &resendStub{
		emailStatus: http.StatusOK,
		listStatus:  http.StatusOK,
		fileStatus:  http.StatusOK,
		fileBody:    []byte("%PDF-1.7 tender"),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /emails/receiving/{id}", func(w http.ResponseWriter, r *http.Request) {
		stub.authHeaders = append(stub.authHeaders, r.Header.Get("Authorization"))
		w.WriteHeader(stub.emailStatus)
		_, _ = w.Write([]byte(stub.emailBody))
	})
	mux.HandleFunc("GET /emails/receiving/{id}/attachments", func(w http.ResponseWriter, r *http.Request) {
		call := stub.listCalls.Add(1)
		w.WriteHeader(stub.listStatus)
		if stub.listStatus != http.StatusOK {
			_, _ = w.Write([]byte(`{"statusCode":500,"message":"boom"}`))
			return
		}
		if call == 1 {
			assert.Equal(t, "100", r.URL.Query().Get("limit"))
			assert.Empty(t, r.URL.Query().Get("after"))
			_, _ = w.Write([]byte(`{"object":"list","has_more":true,"data":[` +
				fileJSON(stub.server.URL, "att_1", "tender.pdf", "application/pdf", 15) + `]}`))
			return
		}
		assert.Equal(t, "att_1", r.URL.Query().Get("after"))
		_, _ = w.Write([]byte(`{"object":"list","has_more":false,"data":[` +
			fileJSON(stub.server.URL, "att_2", "huge.zip", "application/zip", MaxAttachmentBytes+1) + `]}`))
	})
	mux.HandleFunc("GET /files/{id}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(stub.fileStatus)
		_, _ = w.Write(stub.fileBody)
	})

	stub.server = httptest.NewTLSServer(mux)
	t.Cleanup(stub.server.Close)

	stub.emailBody = `{
		"object": "email",
		"id": "` + testEmailID + `",
		"message_id": "<abc@bigshipper.com>",
		"subject": "Load 88213 tender",
		"html": "<p>Please confirm pickup Thursday.</p>",
		"text": null,
		"headers": {
			"in-reply-to": "<prev@bigshipper.com>",
			"references": ["<one@bigshipper.com>", "<prev@bigshipper.com>"]
		},
		"attachments": [{"id": "att_1"}, {"id": "att_2"}]
	}`

	return stub
}

func fileJSON(base, id, name, contentType string, size int) string {
	return `{"id":"` + id + `","filename":"` + name + `","content_type":"` + contentType +
		`","size":` + strconv.Itoa(size) + `,"download_url":"` + base + `/files/` + id + `?sig=x"}`
}

func (s *resendStub) fetcher() *Fetcher {
	return NewFetcher(&Options{
		ResendBaseURL:  s.server.URL,
		APIClient:      s.server.Client(),
		DownloadClient: s.server.Client(),
	})
}

func request() *services.InboundContentRequest {
	return &services.InboundContentRequest{
		Provider:          inboundmessage.ProviderResend,
		APIKey:            testAPIKey,
		ProviderMessageID: testEmailID,
	}
}

func TestFetchResendReadsBodyHeadersAndAttachments(t *testing.T) {
	t.Parallel()

	stub := newResendStub(t)

	content, err := stub.fetcher().Fetch(t.Context(), request())
	require.NoError(t, err)

	assert.Equal(t, []string{"Bearer " + testAPIKey}, stub.authHeaders)
	assert.Equal(t, "<p>Please confirm pickup Thursday.</p>", content.HTML)
	assert.Empty(t, content.Text)
	assert.Equal(t, "<abc@bigshipper.com>", content.MessageID)
	assert.Equal(t, "<prev@bigshipper.com>", content.InReplyTo)
	assert.Equal(t, []string{"<one@bigshipper.com>", "<prev@bigshipper.com>"}, content.References)

	require.Len(t, content.Attachments, 2)
	assert.Equal(t, "att_1", content.Attachments[0].ProviderID)
	assert.Equal(t, "tender.pdf", content.Attachments[0].FileName)
	assert.Equal(t, "application/pdf", content.Attachments[0].ContentType)
	assert.Equal(t, []byte("%PDF-1.7 tender"), content.Attachments[0].Content)
	assert.Empty(t, content.Attachments[0].FailureText)

	assert.Equal(t, "att_2", content.Attachments[1].ProviderID)
	assert.Nil(t, content.Attachments[1].Content)
	assert.Contains(t, content.Attachments[1].FailureText, "larger than the 32 MB")
	assert.Equal(t, int32(2), stub.listCalls.Load())
}

func TestFetchResendSkipsTheListWhenThereAreNoAttachments(t *testing.T) {
	t.Parallel()

	stub := newResendStub(t)
	stub.emailBody = `{"id":"` + testEmailID + `","text":"Plain only","html":null,"attachments":[]}`

	content, err := stub.fetcher().Fetch(t.Context(), request())
	require.NoError(t, err)

	assert.Equal(t, "Plain only", content.Text)
	assert.Empty(t, content.HTML)
	assert.Empty(t, content.Attachments)
	assert.Zero(t, stub.listCalls.Load())
}

func TestFetchResendClassifiesFailures(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		mutate   func(*resendStub)
		req      func(*services.InboundContentRequest)
		sentinel error
		contains string
	}{
		{
			name:     "missing key",
			req:      func(r *services.InboundContentRequest) { r.APIKey = "  " },
			sentinel: services.ErrInboundContentRejected,
			contains: "no Resend API key",
		},
		{
			name: "sending-only key",
			mutate: func(s *resendStub) {
				s.emailStatus = http.StatusUnauthorized
				s.emailBody = `{"statusCode":401,"message":"This API key is restricted to only send emails","name":"restricted_api_key"}`
			},
			sentinel: services.ErrInboundContentRejected,
			contains: "full access key",
		},
		{
			name: "unknown email",
			mutate: func(s *resendStub) {
				s.emailStatus = http.StatusNotFound
				s.emailBody = `{"statusCode":404,"message":"Email not found"}`
			},
			sentinel: services.ErrInboundContentRejected,
			contains: "Email not found",
		},
		{
			name:     "rate limited",
			mutate:   func(s *resendStub) { s.emailStatus = http.StatusTooManyRequests; s.emailBody = `{}` },
			sentinel: services.ErrInboundContentUnavailable,
			contains: "429",
		},
		{
			name:     "server error listing files",
			mutate:   func(s *resendStub) { s.listStatus = http.StatusBadGateway },
			sentinel: services.ErrInboundContentUnavailable,
			contains: "502",
		},
		{
			name:     "expired download",
			mutate:   func(s *resendStub) { s.fileStatus = http.StatusForbidden },
			sentinel: services.ErrInboundContentUnavailable,
			contains: "tender.pdf",
		},
		{
			name: "unsupported provider",
			req: func(r *services.InboundContentRequest) {
				r.Provider = inboundmessage.ProviderPostmark
			},
			sentinel: services.ErrInboundContentRejected,
			contains: "Postmark",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			stub := newResendStub(t)
			if tt.mutate != nil {
				tt.mutate(stub)
			}
			req := request()
			if tt.req != nil {
				tt.req(req)
			}

			content, err := stub.fetcher().Fetch(t.Context(), req)
			require.Error(t, err)
			assert.Nil(t, content)
			require.ErrorIs(t, err, tt.sentinel)
			assert.Contains(t, err.Error(), tt.contains)
		})
	}
}

func TestDownloadAttachmentRefusesNonHTTPSAddresses(t *testing.T) {
	t.Parallel()

	stub := newResendStub(t)

	attachment, err := stub.fetcher().downloadAttachment(t.Context(), &resendAttachmentFile{
		ID:          "att_9",
		Filename:    "rate.pdf",
		DownloadURL: strings.Replace(stub.server.URL, "https://", "http://", 1) + "/files/att_9",
	})
	require.NoError(t, err)
	assert.Nil(t, attachment.Content)
	assert.Equal(t, "Resend did not give a download address for the file.", attachment.FailureText)
}

func TestLimitRedirects(t *testing.T) {
	t.Parallel()

	httpsReq := httptest.NewRequest(http.MethodGet, "https://cdn.example.com/a", nil)
	httpReq := httptest.NewRequest(http.MethodGet, "http://cdn.example.com/a", nil)

	require.NoError(t, limitRedirects(httpsReq, nil))
	require.Error(t, limitRedirects(httpReq, nil))
	require.Error(t, limitRedirects(httpsReq, make([]*http.Request, maxRedirects)))
}
