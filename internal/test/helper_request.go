package test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api"
	"github.com/gofiber/fiber/v2"
)

type (
	GenericPayload      map[string]interface{}
	GenericArrayPayload []interface{}
)

func (g GenericPayload) Reader(t *testing.T) *bytes.Reader {
	t.Helper()

	b, err := sonic.Marshal(g)
	if err != nil {
		t.Fatalf("failed to serialize payload: %v", err)
	}

	return bytes.NewReader(b)
}

func (g GenericArrayPayload) Reader(t *testing.T) *bytes.Reader {
	t.Helper()

	b, err := sonic.Marshal(g)
	if err != nil {
		t.Fatalf("failed to serialize payload: %v", err)
	}

	return bytes.NewReader(b)
}

func PerformRequestWithArrayAndParams(t *testing.T, s *api.Server, method string, path string, body GenericArrayPayload, headers http.Header, queryParams map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	if body == nil {
		return PerformRequestWithRawBody(t, s, method, path, nil, headers, queryParams)
	}

	return PerformRequestWithRawBody(t, s, method, path, body.Reader(t), headers, queryParams)
}

func PerformRequestWithRawBody(
	t *testing.T, s *api.Server, method string, path string, body io.Reader, headers http.Header, queryParams map[string]string,
) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, path, body)

	if headers != nil {
		req.Header = headers
	}
	if body != nil && len(req.Header.Get(fiber.HeaderContentType)) == 0 {
		req.Header.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	}

	if queryParams != nil {
		q := req.URL.Query()
		for k, v := range queryParams {
			q.Add(k, v)
		}

		req.URL.RawQuery = q.Encode()
	}

	res := httptest.NewRecorder()

	resp, _ := s.Fiber.Test(req)

	res.Code = resp.StatusCode

	if resp.Body != nil {
		res.Body = new(bytes.Buffer)
		_, _ = res.Body.ReadFrom(resp.Body)
		resp.Body.Close() // Close the response body
	}

	return res
}

func PerformRequestWithParams(t *testing.T, s *api.Server, method string, path string, body GenericPayload, headers http.Header, queryParams map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	if body == nil {
		return PerformRequestWithRawBody(t, s, method, path, nil, headers, queryParams)
	}

	return PerformRequestWithRawBody(t, s, method, path, body.Reader(t), headers, queryParams)
}

func PerformRequest(t *testing.T, s *api.Server, method string, path string, body GenericPayload, headers http.Header) *httptest.ResponseRecorder {
	t.Helper()

	return PerformRequestWithParams(t, s, method, path, body, headers, nil)
}
