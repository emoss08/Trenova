package tms

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bytedance/sonic"
)

const (
	Port           = 8090
	defaultTimeout = 10 * time.Second
	maxBody        = 4 << 20
	maxDetail      = 300
)

var (
	ErrUnauthorized = errors.New("the TMS refuses the managed token")
	ErrNoToken      = errors.New("the managed token is not in the TMS's Secret yet")
)

type StatusError struct {
	Status int
	Title  string
	Detail string
}

func (e *StatusError) Error() string {
	msg := fmt.Sprintf("the TMS answered %d", e.Status)
	if e.Title != "" {
		msg += ": " + e.Title
	}
	if e.Detail != "" {
		msg += " (" + e.Detail + ")"
	}
	return msg
}

type Reading struct {
	Meter    string    `json:"meter"`
	Hour     time.Time `json:"hour"`
	Quantity int64     `json:"quantity"`
}

type Window struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

type Meter struct {
	Meter     string `json:"meter"`
	Unit      string `json:"unit"`
	Quantity  int64  `json:"quantity"`
	Allowance int64  `json:"allowance"`
	HardCap   bool   `json:"hardCap"`
	Stopped   bool   `json:"stopped"`
}

type State struct {
	Window  *Window `json:"window,omitempty"`
	Overage bool    `json:"overage"`
	Meters  []Meter `json:"meters"`
}

type Invite struct {
	TokenSHA256 string    `json:"tokenSha256"`
	ExpiresAt   time.Time `json:"expiresAt"`
}

type Administrator struct {
	SetUp             bool   `json:"setUp"`
	InviteTokenSHA256 string `json:"inviteTokenSha256,omitempty"`
}

type Tokens interface {
	Token(ctx context.Context) (string, error)
	Refresh(ctx context.Context) (string, error)
}

type Client struct {
	base   *url.URL
	doer   *http.Client
	tokens Tokens
}

type Params struct {
	URL    string
	Tokens Tokens
	HTTP   *http.Client
}

func New(p Params) (*Client, error) {
	base, err := url.Parse(strings.TrimSuffix(p.URL, "/") + "/")
	if err != nil || (base.Scheme != "http" && base.Scheme != "https") || base.Host == "" {
		return nil, fmt.Errorf("the TMS URL %q is not an http(s) URL", p.URL)
	}
	if p.Tokens == nil {
		return nil, errors.New("the TMS client needs a token source")
	}
	doer := p.HTTP
	if doer == nil {
		doer = &http.Client{Timeout: defaultTimeout}
	}
	return &Client{base: base, doer: doer, tokens: p.Tokens}, nil
}

func (c *Client) Readings(ctx context.Context, since time.Time) ([]Reading, error) {
	u := c.base.JoinPath("managed", "v1", "readings")
	u.RawQuery = url.Values{"since": {since.UTC().Format(time.RFC3339)}}.Encode()
	body, err := c.do(ctx, http.MethodGet, u, nil, http.StatusOK)
	if err != nil {
		return nil, fmt.Errorf("read the TMS's usage: %w", err)
	}
	var out struct {
		Readings []Reading `json:"readings"`
	}
	if err = sonic.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("read the TMS's usage: %w", err)
	}
	return out.Readings, nil
}

func (c *Client) PutState(ctx context.Context, state State) error {
	if state.Meters == nil {
		state.Meters = []Meter{}
	}
	body, err := sonic.Marshal(state)
	if err != nil {
		return fmt.Errorf("send the usage state to the TMS: %w", err)
	}
	u := c.base.JoinPath("managed", "v1", "state")
	if _, err = c.do(ctx, http.MethodPut, u, body, http.StatusNoContent); err != nil {
		return fmt.Errorf("send the usage state to the TMS: %w", err)
	}
	return nil
}

func (c *Client) InstallInvite(ctx context.Context, invite Invite) (Administrator, error) {
	invite.ExpiresAt = invite.ExpiresAt.UTC()
	body, err := sonic.Marshal(invite)
	if err != nil {
		return Administrator{}, fmt.Errorf("install the first administrator's invite: %w", err)
	}
	u := c.base.JoinPath("managed", "v1", "first-administrator", "invite")
	out, err := c.do(ctx, http.MethodPut, u, body, http.StatusOK)
	if err != nil {
		return Administrator{}, fmt.Errorf("install the first administrator's invite: %w", err)
	}
	var admin Administrator
	if err = sonic.Unmarshal(out, &admin); err != nil {
		return Administrator{}, fmt.Errorf("install the first administrator's invite: %w", err)
	}
	return admin, nil
}

func (c *Client) do(
	ctx context.Context,
	method string,
	u *url.URL,
	body []byte,
	want int,
) ([]byte, error) {
	token, err := c.tokens.Token(ctx)
	if err != nil {
		return nil, err
	}
	status, out, err := c.send(ctx, request{method: method, url: u, body: body, token: token})
	if err != nil {
		return nil, err
	}
	if status == http.StatusUnauthorized {
		if token, err = c.tokens.Refresh(ctx); err != nil {
			return nil, err
		}
		status, out, err = c.send(ctx, request{method: method, url: u, body: body, token: token})
		if err != nil {
			return nil, err
		}
	}
	switch status {
	case want:
		return out, nil
	case http.StatusUnauthorized:
		return nil, ErrUnauthorized
	default:
		return nil, problem(status, out)
	}
}

type request struct {
	method string
	url    *url.URL
	body   []byte
	token  string
}

func (c *Client) send(ctx context.Context, r request) (int, []byte, error) {
	var body io.Reader
	if r.body != nil {
		body = bytes.NewReader(r.body)
	}
	req, err := http.NewRequestWithContext(ctx, r.method, r.url.String(), body)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+r.token)
	req.Header.Set("Accept", "application/json")
	if r.body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := c.doer.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer res.Body.Close()
	out, err := io.ReadAll(io.LimitReader(res.Body, maxBody+1))
	if err != nil {
		return 0, nil, err
	}
	if len(out) > maxBody {
		return 0, nil, fmt.Errorf("the TMS's answer is larger than %d bytes", maxBody)
	}
	return res.StatusCode, out, nil
}

func problem(status int, body []byte) error {
	var p struct {
		Title  string `json:"title"`
		Detail string `json:"detail"`
	}
	_ = sonic.Unmarshal(body, &p)
	return &StatusError{Status: status, Title: clip(p.Title), Detail: clip(p.Detail)}
}

func clip(s string) string {
	if len(s) <= maxDetail {
		return s
	}
	return s[:maxDetail] + "…"
}
