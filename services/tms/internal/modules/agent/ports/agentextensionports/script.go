package agentextensionports

import (
	"context"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/codemode"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/shopspring/decimal"
)

type ScriptStatus string

const (
	ScriptSucceeded = ScriptStatus("Succeeded")
	ScriptErrored   = ScriptStatus("Errored")
	ScriptTimedOut  = ScriptStatus("TimedOut")
	ScriptStopped   = ScriptStatus("Stopped")
	ScriptFailed    = ScriptStatus("Failed")
)

type ScriptRequest struct {
	TenantInfo pagination.TenantInfo
	Title      string
	Purpose    string
	Code       string
	Inputs     map[string]any
	Broker     codemode.Broker
	StepKey    string
}

type ScriptOutcome struct {
	Status          ScriptStatus
	Output          string
	OutputTruncated bool
	Result          sonic.NoCopyRawMessage
	Error           *codemode.ScriptError
	Limit           codemode.Limit
	Failure         string
	Limits          codemode.Limits
	DurationMs      int64
	RoundTripsMs    []int64
	CostUSD         decimal.Decimal
}

type ScriptRunner interface {
	RunScript(ctx context.Context, req *ScriptRequest) (*ScriptOutcome, error)
}
