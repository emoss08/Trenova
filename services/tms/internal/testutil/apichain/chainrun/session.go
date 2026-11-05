package chainrun

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/testutil/apichain"
)

const (
	GraphQLPath  = "/graphql"
	loginPath    = "/api/v1/auth/login"
	csrfPath     = "/api/v1/auth/csrf"
	FixturesPass = "admin123!" // #nosec G101 -- the published development login the fixtures seed.
	requestLimit = 2 * time.Minute
)

type Result struct {
	Status int    `json:"status"`
	Code   string `json:"code,omitempty"`
	Body   any    `json:"body"`
}

func (r Result) Failed() bool {
	return r.Status >= http.StatusBadRequest || r.Code != ""
}

type Session struct {
	base       string
	client     *http.Client
	csrfToken  string
	csrfHeader string
}

func Anonymous(base string) *Session {
	jar, _ := cookiejar.New(nil)

	return &Session{
		base:   base,
		client: &http.Client{Jar: jar, Timeout: requestLimit},
	}
}

func Login(ctx context.Context, base, email string) (*Session, error) {
	session := Anonymous(base)
	payload := map[string]any{"emailAddress": email, "password": FixturesPass}
	result, err := session.send(ctx, http.MethodPost, loginPath, payload)
	if err != nil {
		return nil, err
	}
	if result.Failed() {
		return nil, fmt.Errorf("log in as %s: %d %s", email, result.Status, describe(result.Body))
	}

	if err = session.refreshCSRF(ctx); err != nil {
		return nil, fmt.Errorf("%s: %w", email, err)
	}

	return session, nil
}

func (s *Session) refreshCSRF(ctx context.Context) error {
	csrf, err := s.send(ctx, http.MethodGet, csrfPath, nil)
	if err != nil {
		return err
	}
	token, _ := apichain.Get(csrf.Body, "csrfToken")
	header, _ := apichain.Get(csrf.Body, "headerName")
	tokenText, tokenOK := token.(string)
	headerText, headerOK := header.(string)
	if !tokenOK || !headerOK || tokenText == "" || headerText == "" {
		return fmt.Errorf("read the CSRF token: %s", describe(csrf.Body))
	}
	s.csrfToken = tokenText
	s.csrfHeader = headerText

	return nil
}

func (s *Session) GraphQL(
	ctx context.Context,
	operation apichain.Operation,
	vars any,
) (Result, error) {
	payload := map[string]any{"operationName": operation.Name}
	if vars != nil {
		payload["variables"] = vars
	}
	if operation.Hash != "" {
		payload["extensions"] = map[string]any{
			"persistedQuery": map[string]any{
				"version":    1,
				"sha256Hash": strings.TrimPrefix(operation.Hash, "sha256:"),
			},
		}
	} else {
		payload["query"] = operation.Document
	}

	result, err := s.send(ctx, http.MethodPost, GraphQLPath, payload)
	if err != nil {
		return Result{}, err
	}
	if code, ok := apichain.Get(result.Body, "errors.0.extensions.code"); ok {
		result.Code = fmt.Sprint(code)
	} else if _, failed := apichain.Get(result.Body, "errors.0"); failed {
		result.Code = "GRAPHQL_ERROR"
	} else if result.Status >= http.StatusBadRequest {
		result.Code = statusCode(result)
	}

	return result, nil
}

func (s *Session) REST(ctx context.Context, method, path string, body any) (Result, error) {
	result, err := s.send(ctx, method, path, body)
	if err != nil {
		return Result{}, err
	}
	if result.Status >= http.StatusBadRequest {
		result.Code = statusCode(result)
	}

	return result, nil
}

func (s *Session) send(ctx context.Context, method, path string, body any) (Result, error) {
	var reader io.Reader
	if body != nil {
		raw, err := sonic.Marshal(body)
		if err != nil {
			return Result{}, fmt.Errorf("encode the %s %s body: %w", method, path, err)
		}
		reader = bytes.NewReader(raw)
	}

	request, err := http.NewRequestWithContext(ctx, method, s.base+path, reader)
	if err != nil {
		return Result{}, err
	}
	request.Header.Set("Accept", "application/json")
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if s.csrfToken != "" && method != http.MethodGet {
		request.Header.Set(s.csrfHeader, s.csrfToken)
	}

	response, err := s.client.Do(request)
	if err != nil {
		return Result{}, fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer response.Body.Close()

	raw, err := io.ReadAll(response.Body)
	if err != nil {
		return Result{}, fmt.Errorf("read the %s %s response: %w", method, path, err)
	}

	result := Result{Status: response.StatusCode}
	if len(bytes.TrimSpace(raw)) > 0 {
		var parsed any
		if err = sonic.Unmarshal(raw, &parsed); err != nil {
			result.Body = string(raw)
		} else {
			result.Body = parsed
		}
	}

	return result, nil
}

func statusCode(result Result) string {
	if problem, ok := apichain.Get(result.Body, "type"); ok {
		return problemCode(fmt.Sprint(problem))
	}

	return http.StatusText(result.Status)
}

func problemCode(problemType string) string {
	trimmed := strings.TrimRight(problemType, "/")
	if i := strings.LastIndex(trimmed, "/"); i >= 0 {
		return trimmed[i+1:]
	}

	return trimmed
}

func describe(body any) string {
	const limit = 600
	text := describeAll(body)
	if len(text) > limit {
		return text[:limit] + "…"
	}

	return text
}

var sortedJSON = sonic.Config{SortMapKeys: true}.Froze()

func describeAll(value any) string {
	raw, err := sortedJSON.Marshal(value)
	if err != nil {
		return fmt.Sprint(value)
	}

	return string(raw)
}
