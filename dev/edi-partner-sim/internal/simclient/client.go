package simclient

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
	"github.com/emoss08/trenova/edi-partner-sim/internal/journal"
	"github.com/emoss08/trenova/edi-partner-sim/internal/runlog"
	"github.com/emoss08/trenova/edi-partner-sim/internal/sim"
)

const (
	requestTimeout = 60 * time.Second
	maxResponse    = 64 << 20
)

var ErrSimulator = errors.New("the simulator refused the request")

type Client struct {
	base string
	http *http.Client
}

func New(base string, client *http.Client) *Client {
	if client == nil {
		client = &http.Client{Timeout: requestTimeout}
	}
	return &Client{base: strings.TrimRight(base, "/"), http: client}
}

func (c *Client) Base() string {
	return c.base
}

func (c *Client) Healthy(ctx context.Context) error {
	return c.do(ctx, http.MethodGet, "/healthz", nil, nil)
}

func (c *Client) Partner(ctx context.Context, name string) (sim.PartnerInfo, error) {
	var info sim.PartnerInfo
	err := c.do(ctx, http.MethodGet, "/control/"+url.PathEscape(name), nil, &info)
	return info, err
}

func (c *Client) Configure(
	ctx context.Context,
	name string,
	req *sim.TrenovaRequest,
) (sim.PartnerInfo, error) {
	var info sim.PartnerInfo
	err := c.do(ctx, http.MethodPost, "/control/"+url.PathEscape(name)+"/trenova", req, &info)
	return info, err
}

func (c *Client) Send(
	ctx context.Context,
	name string,
	req *sim.SendRequest,
) ([]journal.Message, error) {
	var out struct {
		Messages []journal.Message `json:"messages"`
	}
	err := c.do(ctx, http.MethodPost, "/control/"+url.PathEscape(name)+"/send", req, &out)
	return out.Messages, err
}

func (c *Client) Script(
	ctx context.Context,
	name string,
	req *sim.ScriptRequest,
) ([]sim.ScriptAnswer, error) {
	var out struct {
		Answers []sim.ScriptAnswer `json:"answers"`
	}
	err := c.do(ctx, http.MethodPost, "/control/"+url.PathEscape(name)+"/script", req, &out)
	return out.Answers, err
}

func (c *Client) OverrideAcks(
	ctx context.Context,
	name string,
	req *sim.AckOverrideRequest,
) ([]sim.AckOverride, error) {
	var out struct {
		Overrides []sim.AckOverride `json:"overrides"`
	}
	err := c.do(ctx, http.MethodPost, "/control/"+url.PathEscape(name)+"/acks", req, &out)
	return out.Overrides, err
}

func (c *Client) Messages(ctx context.Context, name, key string) ([]journal.Message, error) {
	query := url.Values{"limit": {"1000"}}
	if key != "" {
		query.Set("key", key)
	}
	var out struct {
		Messages []journal.Message `json:"messages"`
	}
	path := "/control/" + url.PathEscape(name) + "/messages?" + query.Encode()
	err := c.do(ctx, http.MethodGet, path, nil, &out)
	return out.Messages, err
}

func (c *Client) StartRun(ctx context.Context, start *runlog.Start) (runlog.Summary, error) {
	var summary runlog.Summary
	err := c.do(ctx, http.MethodPost, "/control/runs", start, &summary)
	return summary, err
}

func (c *Client) FinishRun(ctx context.Context, id string, finish *runlog.Finish) error {
	return c.do(ctx, http.MethodPost, "/control/runs/"+url.PathEscape(id), finish, nil)
}

func (c *Client) PutScenario(ctx context.Context, id string, scenario *runlog.Scenario) error {
	return c.do(ctx, http.MethodPost, "/control/runs/"+url.PathEscape(id)+"/scenarios",
		scenario, nil)
}

func (c *Client) PutStep(ctx context.Context, id string, post *runlog.StepPost) error {
	return c.do(ctx, http.MethodPost, "/control/runs/"+url.PathEscape(id)+"/steps", post, nil)
}

func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	var reader io.Reader
	if body != nil {
		encoded, err := sonic.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode %s %s: %w", method, path, err)
		}
		reader = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, c.base+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := c.http.Do(request)
	if err != nil {
		return fmt.Errorf("the simulator at %s: %w", c.base, err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, maxResponse))
	if err != nil {
		return fmt.Errorf("read %s %s: %w", method, path, err)
	}
	if response.StatusCode < 200 || response.StatusCode > 299 {
		var problem struct {
			Error string `json:"error"`
		}
		_ = sonic.Unmarshal(data, &problem)
		message := strings.TrimSpace(problem.Error)
		if message == "" {
			message = strings.TrimSpace(string(data))
		}
		return fmt.Errorf("%w: %s %s answered %d: %s",
			ErrSimulator, method, path, response.StatusCode, message)
	}
	if out == nil || len(data) == 0 {
		return nil
	}
	if err = sonic.Unmarshal(data, out); err != nil {
		return fmt.Errorf("decode %s %s: %w", method, path, err)
	}
	return nil
}
