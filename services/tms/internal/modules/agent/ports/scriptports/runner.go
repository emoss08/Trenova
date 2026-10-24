package scriptports

import (
	"context"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/codemode"
	"github.com/emoss08/trenova/pkg/pagination"
)

type Status string

const (
	Succeeded = Status("Succeeded")
	Errored   = Status("Errored")
	TimedOut  = Status("TimedOut")
	Stopped   = Status("Stopped")
	Failed    = Status("Failed")
)

type Request struct {
	TenantInfo pagination.TenantInfo
	Title      string
	Purpose    string
	Code       string
	Inputs     map[string]any
	Timezone   string
	Broker     codemode.Broker
	StepKey    string
}

type Outcome struct {
	Status          Status
	Output          string
	OutputTruncated bool
	Result          sonic.NoCopyRawMessage
	Error           *codemode.ScriptError
	Limit           codemode.Limit
	Failure         string
	Limits          codemode.Limits
	DurationMs      int64
}

type Runner interface {
	Run(ctx context.Context, req *Request) (*Outcome, error)
}
