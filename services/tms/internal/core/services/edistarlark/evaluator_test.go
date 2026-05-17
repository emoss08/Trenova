package edistarlark

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.starlark.net/starlark"
)

func TestEvaluate_ValueContextReturnsScalar(t *testing.T) {
	t.Parallel()

	ctx, err := BuildContext(
		map[string]any{"bol": " BOL-123 "},
		nil,
		map[string]any{"transactionControlNumber": "0001"},
		nil,
	)
	require.NoError(t, err)

	result := Evaluate(t.Context(), EvalRequest{
		Script: `def value(ctx):
    return ctx["shipment"]["bol"]`,
		Context:         ctx,
		SegmentID:       "B2",
		ElementPosition: 2,
		Path:            "shipment.bol",
	})

	require.Empty(t, result.Diagnostics)
	assert.Equal(t, "BOL-123", result.Value)
	assert.NotZero(t, result.ExecutionSteps)
}

func TestEvaluate_ValueWithItemReadsRepeatContext(t *testing.T) {
	t.Parallel()

	payload := edi.LoadTenderPayload{
		BOL: "BOL-9",
	}
	ctx, err := BuildContextFromPayload(payload, nil, nil, nil)
	require.NoError(t, err)

	result := Evaluate(t.Context(), EvalRequest{
		Script: `def value(ctx, item):
    return item["locationName"] + ":" + ctx["shipment"]["bol"]`,
		Context: ctx,
		Item: edi.LoadTenderStop{
			LocationName: "Chicago Dock",
		},
		SegmentID:       "N1",
		ElementPosition: 2,
		Path:            "moves.stops.locationName",
	})

	require.Empty(t, result.Diagnostics)
	assert.Equal(t, "Chicago Dock:BOL-9", result.Value)
}

func TestEvaluate_CoalesceAndDefaultFallbacks(t *testing.T) {
	t.Parallel()

	result := Evaluate(t.Context(), EvalRequest{
		Script: `def value(ctx):
    return concat(coalesce(None, "", ctx["shipment"]["customer"]), "|", default(ctx["shipment"]["missing"], "fallback"))`,
		Context: map[string]any{
			"shipment": map[string]any{
				"customer": "Acme",
				"missing":  "",
			},
		},
	})

	require.Empty(t, result.Diagnostics)
	assert.Equal(t, "Acme|fallback", result.Value)
}

func TestEvaluate_ApprovedHelpers(t *testing.T) {
	t.Parallel()

	shipDate := time.Date(2026, 5, 16, 14, 30, 0, 0, time.UTC).Unix()
	result := Evaluate(t.Context(), EvalRequest{
		Script: `def value(ctx):
    return concat(
        trim("  abc  "), "|",
        upper("az"), "|",
        lower("BY"), "|",
        substring("abcdef", 1, 4), "|",
        left_pad(7, 3, "0"), "|",
        right_pad("A", 3, " "), "|",
        truncate("abcdef", 3), "|",
        remove_punctuation("A-B.C"), "|",
        format_date(ctx["shipment"]["shipDate"]), "|",
        format_time(ctx["shipment"]["shipDate"]), "|",
        format_date("05/16/2026"), "|",
        format_decimal("12.345", places=2), "|",
        format_int("12.6"), "|",
        normalize_phone("+1 (312) 555-0100"), "|",
        normalize_state(" il "), "|",
        normalize_postal("60601-1234"), "|",
        qualifier("prepaid", {"prepaid": "PP"}, fallback="CC"), "|",
        required("ok"), "|",
        empty_if_none(None), "|",
        exists("x"),
    )`,
		Context: map[string]any{
			"shipment": map[string]any{
				"shipDate": shipDate,
			},
		},
	})

	require.Empty(t, result.Diagnostics)
	assert.Equal(
		t,
		"abc|AZ|by|bcd|007|A  |abc|ABC|20260516|1430|20260516|12.35|13|3125550100|IL|606011234|PP|ok||Y",
		result.Value,
	)
}

func TestEvaluate_RequiredHelperReportsRuntimeError(t *testing.T) {
	t.Parallel()

	result := Evaluate(t.Context(), EvalRequest{
		Script: `def value(ctx):
    return required("", "BOL is required")`,
		Context: map[string]any{},
		Path:    "shipment.bol",
	})

	requireDiagnostic(t, result, "starlark_runtime_error")
	assert.Contains(t, result.Diagnostics[0].Message, "BOL is required")
}

func TestEvaluate_InvalidSyntaxReturnsDiagnostic(t *testing.T) {
	t.Parallel()

	result := Evaluate(t.Context(), EvalRequest{
		Script: `def value(ctx)
    return "bad"`,
		Context: map[string]any{},
		Path:    "bad.star",
	})

	requireDiagnostic(t, result, "starlark_syntax_error")
}

func TestEvaluate_RuntimeErrorReturnsDiagnostic(t *testing.T) {
	t.Parallel()

	result := Evaluate(t.Context(), EvalRequest{
		Script: `def value(ctx):
    return 1 / 0`,
		Context: map[string]any{},
	})

	requireDiagnostic(t, result, "starlark_runtime_error")
}

func TestEvaluate_InfiniteLoopStoppedByStepLimit(t *testing.T) {
	t.Parallel()

	evaluator := NewEvaluator(Options{
		MaxExecutionSteps: 1_000,
		Timeout:           time.Second,
	})
	result := evaluator.Evaluate(t.Context(), EvalRequest{
		Script: `def value(ctx):
    while True:
        pass`,
		Context: map[string]any{},
	})

	requireDiagnostic(t, result, "starlark_step_limit")
	assert.NotZero(t, result.ExecutionSteps)
}

func TestEvaluate_ContextTimeoutCancelsExpensiveScript(t *testing.T) {
	t.Parallel()

	evaluator := NewEvaluator(Options{
		MaxExecutionSteps: 1_000_000_000_000,
		Timeout:           time.Millisecond,
	})
	result := evaluator.Evaluate(t.Context(), EvalRequest{
		Script: `def value(ctx):
    total = 0
    while True:
        total = total + 1
    return total`,
		Context: map[string]any{},
	})

	requireDiagnostic(t, result, "starlark_timeout")
}

func TestEvaluate_ContextTimeoutReturnsWhileHelperBlocked(t *testing.T) {
	t.Parallel()

	helperStarted := make(chan struct{})
	releaseHelper := make(chan struct{})
	helperExited := make(chan struct{})

	var releaseOnce sync.Once
	release := func() {
		releaseOnce.Do(func() {
			close(releaseHelper)
		})
	}
	t.Cleanup(release)

	evaluator := NewEvaluator(Options{
		MaxExecutionSteps: 1_000_000,
		Timeout:           20 * time.Millisecond,
	})
	evaluator.predeclared = approvedHelpers()
	evaluator.predeclared["blocking_helper"] = starlark.NewBuiltin(
		"blocking_helper",
		func(_ *starlark.Thread, _ *starlark.Builtin, _ starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
			close(helperStarted)
			defer close(helperExited)

			<-releaseHelper
			return starlark.String("late result"), nil
		},
	)
	evaluator.predeclared.Freeze()

	resultCh := make(chan EvalResult, 1)
	go func() {
		resultCh <- evaluator.Evaluate(t.Context(), EvalRequest{
			Script: `def value(ctx):
    return blocking_helper()`,
			Context: map[string]any{},
		})
	}()

	select {
	case <-helperStarted:
	case result := <-resultCh:
		t.Fatalf("Evaluate returned before blocking helper started: %+v", result.Diagnostics)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("blocking helper did not start")
	}

	blockedAt := time.Now()
	var result EvalResult
	select {
	case result = <-resultCh:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Evaluate did not return promptly after timeout")
	}

	requireDiagnostic(t, result, "starlark_timeout")
	assert.Less(t, time.Since(blockedAt), 100*time.Millisecond)

	select {
	case <-helperExited:
		t.Fatal("blocking helper exited before test released it")
	default:
	}

	release()
	select {
	case <-helperExited:
	case <-time.After(time.Second):
		t.Fatal("blocking helper did not exit after release")
	}
}

func TestEvaluate_DisallowsImportsAndUnapprovedNames(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		script     string
		wantCode   string
		wantNeedle string
	}{
		{
			name: "load",
			script: `load("secret.star", "secret")
def value(ctx):
    return secret`,
			wantCode:   "starlark_runtime_error",
			wantNeedle: "imports are disabled",
		},
		{
			name: "open",
			script: `def value(ctx):
    return open("/etc/passwd")`,
			wantCode:   "starlark_runtime_error",
			wantNeedle: "undefined: open",
		},
		{
			name: "env",
			script: `def value(ctx):
    return env("HOME")`,
			wantCode:   "starlark_runtime_error",
			wantNeedle: "undefined: env",
		},
		{
			name: "unapproved helper",
			script: `def value(ctx):
    return sha256("x")`,
			wantCode:   "starlark_runtime_error",
			wantNeedle: "undefined: sha256",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := Evaluate(t.Context(), EvalRequest{
				Script:  tt.script,
				Context: map[string]any{},
			})

			requireDiagnostic(t, result, tt.wantCode)
			assert.Contains(t, result.Diagnostics[0].Message, tt.wantNeedle)
		})
	}
}

func TestEvaluate_FrozenContextCannotBeMutated(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		script string
		item   any
	}{
		{
			name: "ctx",
			script: `def value(ctx):
    ctx["shipment"]["bol"] = "changed"
    return ctx["shipment"]["bol"]`,
		},
		{
			name: "item",
			script: `def value(ctx, item):
    item["locationName"] = "changed"
    return item["locationName"]`,
			item: map[string]any{"locationName": "Chicago"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := Evaluate(t.Context(), EvalRequest{
				Script: tt.script,
				Context: map[string]any{
					"shipment": map[string]any{"bol": "BOL-1"},
				},
				Item: tt.item,
			})

			requireDiagnostic(t, result, "starlark_runtime_error")
			assert.True(t, strings.Contains(result.Diagnostics[0].Message, "frozen"))
		})
	}
}

func TestEvaluate_InvalidResultReturnsDiagnostic(t *testing.T) {
	t.Parallel()

	result := Evaluate(t.Context(), EvalRequest{
		Script: `def value(ctx):
    return {"not": "scalar"}`,
		Context: map[string]any{},
	})

	requireDiagnostic(t, result, "starlark_invalid_result")
}

func TestEvaluate_PanicFromHelperReturnsDiagnostic(t *testing.T) {
	t.Parallel()

	evaluator := NewEvaluator(Options{})
	evaluator.predeclared = approvedHelpers()
	evaluator.predeclared["bad_helper"] = starlark.NewBuiltin(
		"bad_helper",
		func(_ *starlark.Thread, _ *starlark.Builtin, _ starlark.Tuple, _ []starlark.Tuple) (starlark.Value, error) {
			panic("test helper panic")
		},
	)
	evaluator.predeclared.Freeze()

	result := evaluator.Evaluate(t.Context(), EvalRequest{
		Script: `def value(ctx):
    return bad_helper()`,
		Context: map[string]any{},
	})

	requireDiagnostic(t, result, "starlark_panic")
	assert.Contains(t, result.Diagnostics[0].Message, "test helper panic")
}

func requireDiagnostic(t *testing.T, result EvalResult, code string) {
	t.Helper()

	require.Len(t, result.Diagnostics, 1)
	assert.Equal(t, code, result.Diagnostics[0].Code)
	assert.Equal(t, DiagnosticSeverityError, result.Diagnostics[0].Severity)
}
