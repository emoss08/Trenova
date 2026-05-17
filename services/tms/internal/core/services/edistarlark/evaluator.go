package edistarlark

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/shared/jsonutils"
	"go.starlark.net/resolve"
	"go.starlark.net/starlark"
	"go.starlark.net/syntax"
)

type executionResult struct {
	result EvalResult
}

func NewEvaluator(options Options) *Evaluator {
	if options.MaxExecutionSteps == 0 {
		options.MaxExecutionSteps = DefaultMaxExecutionSteps
	}
	if options.Timeout == 0 {
		options.Timeout = DefaultTimeout
	}

	predeclared := approvedHelpers()
	predeclared.Freeze()
	return &Evaluator{
		options:     options,
		predeclared: predeclared,
	}
}

func Evaluate(ctx context.Context, req EvalRequest) EvalResult {
	return NewEvaluator(Options{}).Evaluate(ctx, req)
}

func BuildContext(
	shipment map[string]any,
	partner map[string]any,
	runtime map[string]any,
	mapping map[string]any,
) (map[string]any, error) {
	return map[string]any{
		"shipment": ensureMap(shipment),
		"partner":  ensureMap(partner),
		"runtime":  ensureMap(runtime),
		"mapping":  ensureMap(mapping),
	}, nil
}

func BuildContextFromPayload(
	payload edi.LoadTenderPayload,
	partner map[string]any,
	runtime map[string]any,
	mapping map[string]any,
) (map[string]any, error) {
	shipment, err := jsonutils.ToJSON(payload)
	if err != nil {
		return nil, fmt.Errorf("build shipment context: %w", err)
	}
	return BuildContext(shipment, partner, runtime, mapping)
}

func (e *Evaluator) Evaluate(ctx context.Context, req EvalRequest) EvalResult {
	if ctx == nil {
		ctx = context.Background()
	}

	evalCtx, cancel := context.WithTimeout(ctx, e.options.Timeout)
	defer cancel()

	thread := e.newThread()
	thread.SetMaxExecutionSteps(e.options.MaxExecutionSteps)

	done := make(chan executionResult, 1)
	go func() {
		done <- executionResult{result: e.evaluateOnThread(thread, req)}
	}()

	select {
	case result := <-done:
		return result.result
	case <-evalCtx.Done():
		thread.Cancel("execution timed out")

		select {
		case result := <-done:
			if len(result.result.Diagnostics) == 0 {
				result.result.Diagnostics = []Diagnostic{
					diagnostic(req, "starlark_timeout", "Starlark execution timed out"),
				}
			}
			return result.result
		case <-time.After(timeoutCancelGrace):
			return EvalResult{
				Diagnostics: []Diagnostic{
					diagnostic(req, "starlark_timeout", "Starlark execution timed out"),
				},
			}
		}
	}
}

func (e *Evaluator) evaluateOnThread(thread *starlark.Thread, req EvalRequest) (result EvalResult) {
	result.Diagnostics = []Diagnostic{}
	defer func() {
		result.ExecutionSteps = thread.ExecutionSteps()
		if recovered := recover(); recovered != nil {
			result.Value = ""
			result.Raw = nil
			result.Diagnostics = []Diagnostic{
				diagnostic(req, "starlark_panic", fmt.Sprintf("Starlark runtime panicked: %v", recovered)),
			}
		}
	}()

	ctxValue, err := toFrozenStarlarkValue(ensureMap(req.Context))
	if err != nil {
		return resultWithDiagnostic(req, thread, "starlark_runtime_error", err)
	}

	var itemValue starlark.Value
	if req.Item != nil {
		itemValue, err = toFrozenStarlarkValue(req.Item)
		if err != nil {
			return resultWithDiagnostic(req, thread, "starlark_runtime_error", err)
		}
	}

	globals, err := starlark.ExecFileOptions(
		&syntax.FileOptions{While: true},
		thread,
		filename(req),
		req.Script,
		e.predeclared,
	)
	if err != nil {
		return resultWithDiagnostic(req, thread, classifyError(err), err)
	}

	functionName := strings.TrimSpace(req.FunctionName)
	if functionName == "" {
		functionName = defaultFunctionName
	}

	fn, ok := globals[functionName]
	if !ok {
		return resultWithDiagnostic(
			req,
			thread,
			"starlark_runtime_error",
			fmt.Errorf("required Starlark function %q is not defined", functionName),
		)
	}
	if _, ok = fn.(starlark.Callable); !ok {
		return resultWithDiagnostic(
			req,
			thread,
			"starlark_runtime_error",
			fmt.Errorf("%q is not callable", functionName),
		)
	}

	args := starlark.Tuple{ctxValue}
	if req.Item != nil {
		args = starlark.Tuple{ctxValue, itemValue}
	}

	raw, err := starlark.Call(thread, fn, args, nil)
	if err != nil {
		return resultWithDiagnostic(req, thread, classifyError(err), err)
	}

	value, ok := scalarString(raw)
	if !ok {
		result.Raw = raw
		result.ExecutionSteps = thread.ExecutionSteps()
		result.Diagnostics = []Diagnostic{
			diagnostic(
				req,
				"starlark_invalid_result",
				fmt.Sprintf("Starlark function returned unsupported %s result", raw.Type()),
			),
		}
		return result
	}

	result.Raw = raw
	result.Value = value
	result.ExecutionSteps = thread.ExecutionSteps()
	return result
}

func (e *Evaluator) newThread() *starlark.Thread {
	return &starlark.Thread{
		Name:  "edi-starlark",
		Print: func(_ *starlark.Thread, _ string) {},
		Load: func(_ *starlark.Thread, _ string) (starlark.StringDict, error) {
			return nil, errors.New("imports are disabled")
		},
		OnMaxSteps: func(thread *starlark.Thread) {
			thread.Cancel("execution step limit exceeded")
		},
	}
}

func resultWithDiagnostic(
	req EvalRequest,
	thread *starlark.Thread,
	code string,
	err error,
) EvalResult {
	return EvalResult{
		Diagnostics: []Diagnostic{
			diagnostic(req, code, err.Error()),
		},
		ExecutionSteps: thread.ExecutionSteps(),
	}
}

func classifyError(err error) string {
	message := err.Error()
	switch {
	case strings.Contains(message, "execution step limit exceeded"),
		strings.Contains(message, "too many steps"):
		return "starlark_step_limit"
	case strings.Contains(message, "execution timed out"),
		strings.Contains(message, "Starlark computation cancelled"):
		return "starlark_timeout"
	}

	var evalErr *starlark.EvalError
	if errors.As(err, &evalErr) {
		return "starlark_runtime_error"
	}

	var syntaxErr syntax.Error
	if errors.As(err, &syntaxErr) {
		return "starlark_syntax_error"
	}

	var resolveErrs resolve.ErrorList
	if errors.As(err, &resolveErrs) {
		return "starlark_runtime_error"
	}

	return "starlark_syntax_error"
}

func diagnostic(req EvalRequest, code string, message string) Diagnostic {
	return Diagnostic{
		Severity:        DiagnosticSeverityError,
		Code:            code,
		SegmentID:       req.SegmentID,
		ElementPosition: req.ElementPosition,
		Path:            req.Path,
		Message:         message,
		SuggestedFix:    suggestedFix(code),
	}
}

func suggestedFix(code string) string {
	switch code {
	case "starlark_syntax_error":
		return "Fix the Starlark script syntax before rendering this element."
	case "starlark_runtime_error":
		return "Check field paths, helper arguments, and function arity in the Starlark script."
	case "starlark_step_limit":
		return "Reduce loop work or simplify the Starlark script."
	case "starlark_timeout":
		return "Reduce script execution time or simplify expensive loops."
	case "starlark_invalid_result":
		return "Return a string, number, boolean, or None from the Starlark function."
	case "starlark_panic":
		return "Review the approved helper implementation used by this script."
	default:
		return ""
	}
}

func filename(req EvalRequest) string {
	if strings.TrimSpace(req.Path) != "" {
		return req.Path
	}
	return defaultFilename
}

func ensureMap(value map[string]any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	return value
}
