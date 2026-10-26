package lading

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/emoss08/trenova/pkg/x12"
	"github.com/emoss08/trenova/shared/jsonutils"
)

const (
	OpCheck    = "lading.check"
	OpFormat   = "lading.format"
	OpRun      = "lading.run"
	OpTest     = "lading.test"
	OpFlowStep = "lading.flowStep"
	OpFlowKey  = "lading.flowKey"
	OpReplay   = "lading.replay"
	OpStarter  = "lading.starter"
	OpCatalog  = "lading.catalog"
)

const (
	MaxFiles        = 200
	MaxSamples      = 2_000
	MaxReplayInputs = 500
)

var ErrProjectHasErrors = errors.New("lading: the project has errors")

type Client struct {
	runtime *x12.Runtime
}

var defaultClient = sync.OnceValue(func() *Client {
	return New(x12.Default())
})

func New(runtime *x12.Runtime) *Client {
	return &Client{runtime: runtime}
}

func Default() *Client {
	return defaultClient()
}

func (c *Client) Check(ctx context.Context, project *Encoded) (*Check, error) {
	var out Check
	if err := c.call(ctx, OpCheck, projectRequest{Encoded: project}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) Format(ctx context.Context, source string) (*Formatted, error) {
	var out Formatted
	if err := c.call(ctx, OpFormat, formatRequest{Source: source}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) Run(ctx context.Context, project *Encoded, req *RunRequest) (*RunResult, error) {
	var out RunResult
	if err := c.call(ctx, OpRun, runRequest{Encoded: project, RunRequest: req}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) Test(ctx context.Context, project *Encoded) (*TestReport, error) {
	var out TestReport
	if err := c.call(ctx, OpTest, projectRequest{Encoded: project}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) FlowStep(
	ctx context.Context,
	project *Encoded,
	req *FlowRequest,
) (*FlowStep, error) {
	var out FlowStep
	if err := c.call(ctx, OpFlowStep, flowRequest{Encoded: project, FlowRequest: req}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) FlowKey(ctx context.Context, project *Encoded, req *FlowRequest) (string, error) {
	var out struct {
		Key *string `json:"key"`
	}
	if err := c.call(ctx, OpFlowKey, flowRequest{Encoded: project, FlowRequest: req}, &out); err != nil {
		return "", err
	}
	if out.Key == nil {
		return "", nil
	}
	return *out.Key, nil
}

func (c *Client) Replay(ctx context.Context, req *ReplayRequest) (*ReplayReport, error) {
	if len(req.Inputs) > MaxReplayInputs {
		return nil, fmt.Errorf("lading: a replay call takes at most %d inputs", MaxReplayInputs)
	}
	var out ReplayReport
	if err := c.call(ctx, OpReplay, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) Starter(ctx context.Context, req *StarterRequest) (*Starter, error) {
	var out Starter
	if err := c.call(ctx, OpStarter, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) Catalog(ctx context.Context) (jsonutils.RawJSON, error) {
	raw, err := c.runtime.CallRaw(ctx, OpCatalog, []byte("{}"))
	if err != nil {
		return nil, err
	}
	return jsonutils.RawJSON(raw), nil
}

func (c *Client) call(ctx context.Context, op string, input, output any) error {
	return c.runtime.Call(ctx, op, input, output)
}

type projectRequest struct {
	*Encoded
}

type formatRequest struct {
	Source string `json:"source"`
}

type runRequest struct {
	*Encoded
	*RunRequest
}

type flowRequest struct {
	*Encoded
	*FlowRequest
}
