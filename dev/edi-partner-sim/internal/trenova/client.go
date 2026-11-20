package trenova

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/jsonutils"
)

const (
	requestTimeout = 60 * time.Second
	maxResponse    = 32 << 20
	csrfHeader     = "X-CSRF-Token"
)

var ErrTrenova = errors.New("trenova refused the request")

type Problem struct {
	Method string
	Path   string
	Status int
	Title  string
	Detail string
	Errors []FieldError
}

type FieldError struct {
	Path    jsonutils.RawJSON `json:"path"`
	Message string            `json:"message"`
	Code    string            `json:"code"`
}

func (p *Problem) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s answered %d", p.Method, p.Path, p.Status)
	if p.Title != "" {
		b.WriteString(": " + p.Title)
	}
	if p.Detail != "" && p.Detail != p.Title {
		b.WriteString(": " + p.Detail)
	}
	for _, field := range p.Errors {
		path := strings.Trim(string(field.Path), `"`)
		if path != "" && path != "null" {
			fmt.Fprintf(&b, "; %s: %s", path, field.Message)
		} else {
			fmt.Fprintf(&b, "; %s", field.Message)
		}
	}
	return b.String()
}

func (p *Problem) Unwrap() error {
	return ErrTrenova
}

func IsNotFound(err error) bool {
	var problem *Problem
	return errors.As(err, &problem) && problem.Status == http.StatusNotFound
}

type Client struct {
	api  string
	host string
	http *http.Client
	csrf string
}

func New(api string, client *http.Client) (*Client, error) {
	parsed, err := url.Parse(strings.TrimRight(api, "/"))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("--api %q is not an absolute URL", api)
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	if client == nil {
		client = &http.Client{Timeout: requestTimeout}
	}
	copied := *client
	copied.Jar = jar
	return &Client{
		api:  strings.TrimRight(api, "/"),
		host: parsed.Scheme + "://" + parsed.Host,
		http: &copied,
	}, nil
}

func (c *Client) Host() string {
	return c.host
}

type loginResponse struct {
	CSRFToken              string   `json:"csrfToken"`
	MFARequired            bool     `json:"mfaRequired"`
	ActiveRoleIDs          []string `json:"activeRoleIds"`
	AuthorizedRoleIDs      []string `json:"authorizedRoleIds"`
	RequiresRoleActivation bool     `json:"requiresRoleActivation"`
}

func (c *Client) Login(ctx context.Context, email, password string) error {
	var login loginResponse
	if err := c.do(ctx, http.MethodPost, "/auth/login", map[string]string{
		"emailAddress": email,
		"password":     password,
	}, &login); err != nil {
		return fmt.Errorf("sign in as %s: %w", email, err)
	}
	if login.MFARequired {
		return fmt.Errorf("sign in as %s: the account asks for a second factor", email)
	}
	c.csrf = login.CSRFToken
	if len(login.ActiveRoleIDs) > 0 && !login.RequiresRoleActivation {
		return nil
	}
	if len(login.AuthorizedRoleIDs) == 0 {
		return nil
	}
	if err := c.do(ctx, http.MethodPost, "/auth/session/roles/activate", map[string]any{
		"roleIds": login.AuthorizedRoleIDs,
	}, nil); err != nil {
		return fmt.Errorf("activate the session's roles: %w", err)
	}
	return nil
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	return c.do(ctx, http.MethodGet, path, nil, out)
}

func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	return c.send(ctx, method, c.api+path, body, out)
}

func (c *Client) post(ctx context.Context, path string, body, out any) error {
	if body == nil {
		body = map[string]any{}
	}
	return c.do(ctx, http.MethodPost, path, body, out)
}

func (c *Client) put(ctx context.Context, path string, body, out any) error {
	return c.do(ctx, http.MethodPut, path, body, out)
}

func (c *Client) send(ctx context.Context, method, target string, body, out any) error {
	path := strings.TrimPrefix(target, c.host)
	var reader io.Reader
	if body != nil {
		encoded, err := sonic.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode %s %s: %w", method, path, err)
		}
		reader = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, target, reader)
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "application/json")
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if c.csrf != "" && method != http.MethodGet {
		request.Header.Set(csrfHeader, c.csrf)
	}
	response, err := c.http.Do(request)
	if err != nil {
		return fmt.Errorf("trenova at %s: %w", c.api, err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, maxResponse))
	if err != nil {
		return fmt.Errorf("read %s %s: %w", method, path, err)
	}
	if response.StatusCode < 200 || response.StatusCode > 299 {
		return problemOf(method, path, response.StatusCode, data)
	}
	if out == nil || len(bytes.TrimSpace(data)) == 0 {
		return nil
	}
	if err = sonic.Unmarshal(data, out); err != nil {
		return fmt.Errorf("decode %s %s: %w", method, path, err)
	}
	return nil
}

func problemOf(method, path string, status int, data []byte) *Problem {
	var body struct {
		Title  string       `json:"title"`
		Detail string       `json:"detail"`
		Errors []FieldError `json:"errors"`
	}
	problem := &Problem{Method: method, Path: path, Status: status}
	if err := sonic.Unmarshal(data, &body); err != nil {
		problem.Detail = strings.TrimSpace(string(data))
		return problem
	}
	problem.Title, problem.Detail, problem.Errors = body.Title, body.Detail, body.Errors
	return problem
}

type graphQLError struct {
	Message string `json:"message"`
}

func (c *Client) GraphQL(
	ctx context.Context,
	query string,
	variables map[string]any,
	out any,
) error {
	var response struct {
		Data   jsonutils.RawJSON `json:"data"`
		Errors []graphQLError    `json:"errors"`
	}
	if err := c.send(ctx, http.MethodPost, c.host+"/graphql", map[string]any{
		"query":     query,
		"variables": variables,
	}, &response); err != nil {
		return err
	}
	if len(response.Errors) > 0 {
		messages := make([]string, 0, len(response.Errors))
		for _, e := range response.Errors {
			messages = append(messages, e.Message)
		}
		return fmt.Errorf("%w: graphql: %s", ErrTrenova, strings.Join(messages, "; "))
	}
	if out == nil || len(response.Data) == 0 {
		return nil
	}
	return sonic.Unmarshal(response.Data, out)
}

type page[T any] struct {
	Results []T `json:"results"`
}

func listPath(path string, query url.Values) string {
	if len(query) == 0 {
		return path
	}
	return path + "?" + query.Encode()
}
