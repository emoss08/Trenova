package edistarlark

import (
	"context"
	"errors"
	"fmt"
	"sort"
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

const (
	diagnosticCodeInvalidResult = "starlark_invalid_result"
	diagnosticCodePanic         = "starlark_panic"
	diagnosticCodeRuntimeError  = "starlark_runtime_error"
	diagnosticCodeStepLimit     = "starlark_step_limit"
	diagnosticCodeSyntaxError   = "starlark_syntax_error"
	diagnosticCodeTimeout       = "starlark_timeout"
)

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

func ValidateScriptFunction(req EvalRequest) []Diagnostic {
	evaluator := NewEvaluator(Options{})
	thread := evaluator.newThread()
	thread.SetMaxExecutionSteps(evaluator.options.MaxExecutionSteps)

	globals, diagnostics := evaluator.evalLibraries(thread, req)
	if len(diagnostics) > 0 {
		return diagnostics
	}

	if strings.TrimSpace(req.Script) != "" {
		var err error
		globals, err = evaluator.evalInlineScript(thread, req, globals)
		if err != nil {
			return []Diagnostic{diagnostic(req, classifyError(err), err.Error())}
		}
	}

	functionName := strings.TrimSpace(req.FunctionName)
	if functionName == "" {
		if strings.TrimSpace(req.Script) == "" {
			return []Diagnostic{
				diagnostic(
					req,
					DiagnosticCodeFunctionNotFound,
					"Starlark function name is required when no inline script is provided",
				),
			}
		}
		functionName = defaultFunctionName
	}
	fn, ok := globals[functionName]
	if !ok {
		return []Diagnostic{
			diagnostic(
				req,
				DiagnosticCodeFunctionNotFound,
				fmt.Sprintf("required Starlark function %q is not defined", functionName),
			),
		}
	}
	if _, ok = fn.(starlark.Callable); !ok {
		return []Diagnostic{
			diagnostic(
				req,
				DiagnosticCodeFunctionNotCallable,
				fmt.Sprintf("%q is not callable", functionName),
			),
		}
	}
	return nil
}

func DiscoverFunctionNames(script string) ([]string, error) {
	file, err := (&syntax.FileOptions{While: true}).Parse(defaultFilename, script, 0)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0)
	for _, stmt := range file.Stmts {
		def, ok := stmt.(*syntax.DefStmt)
		if !ok || def.Name == nil {
			continue
		}
		names = append(names, def.Name.Name)
	}
	return names, nil
}

func ValidateLibraries(libraries []ScriptLibrary) []Diagnostic {
	evaluator := NewEvaluator(Options{})
	thread := evaluator.newThread()
	thread.SetMaxExecutionSteps(evaluator.options.MaxExecutionSteps)
	req := EvalRequest{Libraries: libraries}
	_, diagnostics := evaluator.evalLibraries(thread, req)
	return diagnostics
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
					diagnostic(req, diagnosticCodeTimeout, "Starlark execution timed out"),
				}
			}
			return result.result
		case <-time.After(timeoutCancelGrace):
			return EvalResult{
				Diagnostics: []Diagnostic{
					diagnostic(req, diagnosticCodeTimeout, "Starlark execution timed out"),
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
				diagnostic(
					req,
					diagnosticCodePanic,
					fmt.Sprintf("Starlark runtime panicked: %v", recovered),
				),
			}
		}
	}()

	ctxValue, err := toFrozenStarlarkValue(ensureMap(req.Context))
	if err != nil {
		return resultWithDiagnostic(req, thread, diagnosticCodeRuntimeError, err)
	}

	var itemValue starlark.Value
	if req.Item != nil {
		itemValue, err = toFrozenStarlarkValue(req.Item)
		if err != nil {
			return resultWithDiagnostic(req, thread, diagnosticCodeRuntimeError, err)
		}
	}

	globals, diagnostics := e.evalLibraries(thread, req)
	if len(diagnostics) > 0 {
		return EvalResult{
			Diagnostics:    diagnostics,
			ExecutionSteps: thread.ExecutionSteps(),
		}
	}

	if strings.TrimSpace(req.Script) != "" {
		globals, err = e.evalInlineScript(thread, req, globals)
		if err != nil {
			return resultWithDiagnostic(req, thread, classifyError(err), err)
		}
	}

	functionName := strings.TrimSpace(req.FunctionName)
	if functionName == "" {
		if strings.TrimSpace(req.Script) == "" {
			return resultWithDiagnostic(
				req,
				thread,
				DiagnosticCodeFunctionNotFound,
				errors.New("starlark function name is required when no inline script is provided"),
			)
		}
		functionName = defaultFunctionName
	}

	fn, ok := globals[functionName]
	if !ok {
		return resultWithDiagnostic(
			req,
			thread,
			DiagnosticCodeFunctionNotFound,
			fmt.Errorf("required Starlark function %q is not defined", functionName),
		)
	}
	if _, ok = fn.(starlark.Callable); !ok {
		return resultWithDiagnostic(
			req,
			thread,
			DiagnosticCodeFunctionNotCallable,
			fmt.Errorf("%q is not callable", functionName),
		)
	}

	args := starlark.Tuple{ctxValue}
	if req.Item != nil && callableAcceptsItem(fn) {
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
				diagnosticCodeInvalidResult,
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

func (e *Evaluator) evalLibraries(
	thread *starlark.Thread,
	req EvalRequest,
) (starlark.StringDict, []Diagnostic) {
	globals := make(starlark.StringDict, len(e.predeclared)+len(req.Libraries))
	for name, value := range e.predeclared {
		globals[name] = value
	}

	if len(req.Libraries) == 0 {
		return globals, nil
	}

	if diagnostics := e.libraryFunctionDiagnostics(req); len(diagnostics) > 0 {
		return nil, diagnostics
	}

	for _, library := range req.Libraries {
		if strings.TrimSpace(library.Script) == "" {
			continue
		}
		libraryGlobals, err := starlark.ExecFileOptions(
			&syntax.FileOptions{While: true},
			thread,
			libraryFilename(library),
			library.Script,
			globals,
		)
		if err != nil {
			libraryReq := req
			libraryReq.Path = "scriptLibrary:" + strings.TrimSpace(library.Name)
			return nil, []Diagnostic{
				diagnostic(libraryReq, classifyLibraryError(err), err.Error()),
			}
		}
		for name, value := range libraryGlobals {
			globals[name] = value
		}
	}
	return globals, nil
}

func (e *Evaluator) evalInlineScript(
	thread *starlark.Thread,
	req EvalRequest,
	globals starlark.StringDict,
) (starlark.StringDict, error) {
	inlineGlobals, err := starlark.ExecFileOptions(
		&syntax.FileOptions{While: true},
		thread,
		filename(req),
		req.Script,
		globals,
	)
	if err != nil {
		return nil, err
	}
	for name, value := range inlineGlobals {
		globals[name] = value
	}
	return globals, nil
}

func (e *Evaluator) libraryFunctionDiagnostics(req EvalRequest) []Diagnostic {
	type functionSource struct {
		Library string
		Path    string
	}

	seen := make(map[string]functionSource, len(req.Libraries))
	diagnostics := make([]Diagnostic, 0)
	for _, library := range req.Libraries {
		names, err := DiscoverFunctionNames(library.Script)
		libraryReq := req
		libraryReq.Path = "scriptLibrary:" + strings.TrimSpace(library.Name)
		if err != nil {
			diagnostics = append(
				diagnostics,
				diagnostic(libraryReq, DiagnosticCodeLibrarySyntaxError, err.Error()),
			)
			continue
		}
		for _, functionName := range names {
			if _, reserved := e.predeclared[functionName]; reserved {
				message := fmt.Sprintf(
					"function %q uses a reserved helper name",
					functionName,
				)
				diagnostics = append(
					diagnostics,
					diagnostic(libraryReq, DiagnosticCodeLibraryReservedFunction, message),
				)
				continue
			}

			source, ok := seen[functionName]
			if !ok {
				seen[functionName] = functionSource{
					Library: library.Name,
					Path:    libraryReq.Path,
				}
				continue
			}
			message := fmt.Sprintf(
				"function %q is defined by both %q and %q",
				functionName,
				source.Library,
				library.Name,
			)
			diagnostics = append(
				diagnostics,
				diagnostic(libraryReq, DiagnosticCodeLibraryDuplicateFunction, message),
			)
		}
	}
	sort.SliceStable(diagnostics, func(i, j int) bool {
		if diagnostics[i].Path == diagnostics[j].Path {
			return diagnostics[i].Message < diagnostics[j].Message
		}
		return diagnostics[i].Path < diagnostics[j].Path
	})
	return diagnostics
}

func classifyLibraryError(err error) string {
	if classifyError(err) == diagnosticCodeSyntaxError {
		return DiagnosticCodeLibrarySyntaxError
	}
	return classifyError(err)
}

func libraryFilename(library ScriptLibrary) string {
	name := strings.TrimSpace(library.Name)
	if name == "" {
		return "edi_script_library.star"
	}
	replacer := strings.NewReplacer("/", "_", "\\", "_", " ", "_", ":", "_")
	return "edi_script_library_" + replacer.Replace(name) + ".star"
}

func callableAcceptsItem(fn starlark.Value) bool {
	starlarkFn, ok := fn.(*starlark.Function)
	if !ok {
		return true
	}
	return starlarkFn.NumParams() >= 2 || starlarkFn.HasVarargs()
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
		return diagnosticCodeStepLimit
	case strings.Contains(message, "execution timed out"),
		strings.Contains(message, "Starlark computation cancelled"):
		return diagnosticCodeTimeout
	}

	var evalErr *starlark.EvalError
	if errors.As(err, &evalErr) {
		return diagnosticCodeRuntimeError
	}

	var syntaxErr syntax.Error
	if errors.As(err, &syntaxErr) {
		return diagnosticCodeSyntaxError
	}

	var resolveErrs resolve.ErrorList
	if errors.As(err, &resolveErrs) {
		return diagnosticCodeRuntimeError
	}

	return diagnosticCodeSyntaxError
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
	case diagnosticCodeSyntaxError:
		return "Fix the Starlark script syntax before rendering this element."
	case diagnosticCodeRuntimeError:
		return "Check the Starlark script, function name, helper arguments, and available context fields."
	case DiagnosticCodeLibrarySyntaxError:
		return "Fix the Starlark script library syntax before rendering this template."
	case DiagnosticCodeLibraryDuplicateFunction:
		return "Rename one library function so each function name is defined once per template version."
	case DiagnosticCodeLibraryReservedFunction:
		return "Rename the library function because this name is reserved by Trenova helper functions."
	case DiagnosticCodeFunctionNotFound:
		return "Define the referenced Starlark function in the inline script or template script libraries."
	case DiagnosticCodeFunctionNotCallable:
		return "Reference a callable Starlark function."
	case diagnosticCodeStepLimit:
		return "Reduce loop work or simplify the Starlark script."
	case diagnosticCodeTimeout:
		return "Reduce script execution time or simplify expensive loops."
	case diagnosticCodeInvalidResult:
		return "Return a string, number, boolean, or None from the Starlark function."
	case diagnosticCodePanic:
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
