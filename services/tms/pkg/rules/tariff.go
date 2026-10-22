package rules

import (
	"context"
	"errors"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/shopspring/decimal"
)

const (
	SeverityError   = "error"
	SeverityWarning = "warning"
)

const (
	OpTariffCheck        = "tariff.check"
	OpTariffEvaluate     = "tariff.evaluate"
	OpTariffTest         = "tariff.test"
	OpTariffFormat       = "tariff.format"
	OpTariffCatalog      = "tariff.catalog"
	OpTariffScaffoldTest = "tariff.scaffoldTest"
)

type TariffKind string

const (
	TariffKindTariff    = TariffKind("tariff")
	TariffKindLibrary   = TariffKind("library")
	TariffKindCondition = TariffKind("condition")
)

type TariffStatus string

const (
	TariffRated   = TariffStatus("rated")
	TariffRefused = TariffStatus("refused")
	TariffError   = TariffStatus("error")
	TariffNeeds   = TariffStatus("needs")
)

type TariffLineKind string

const (
	TariffLineCharge   = TariffLineKind("charge")
	TariffLineMinimum  = TariffLineKind("minimum")
	TariffLineMaximum  = TariffLineKind("maximum")
	TariffLineRounding = TariffLineKind("rounding")
)

type TariffTestStatus string

const (
	TariffTestPassed = TariffTestStatus("passed")
	TariffTestFailed = TariffTestStatus("failed")
	TariffTestError  = TariffTestStatus("error")
)

var errTariffResponse = errors.New("rules: the tariff module answered with an unexpected shape")

type TariffRange struct {
	From int `json:"from"`
	To   int `json:"to"`
}

type TariffLibrary struct {
	Path    string `json:"path"`
	Version string `json:"version,omitempty"`
	Source  string `json:"source"`
}

type TariffAxis struct {
	Match         string `json:"match"`
	Kind          string `json:"kind"`
	Label         string `json:"label,omitempty"`
	Unit          string `json:"unit,omitempty"`
	Normalization string `json:"normalization,omitempty"`
	Overflow      string `json:"overflow,omitempty"`
}

type TariffCellPosition struct {
	Key string           `json:"key,omitempty"`
	Min *decimal.Decimal `json:"min,omitempty"`
	Max *decimal.Decimal `json:"max,omitempty"`
}

type TariffCell struct {
	At      []TariffCellPosition `json:"at"`
	Value   decimal.Decimal      `json:"value"`
	Minimum *decimal.Decimal     `json:"minimum,omitempty"`
	Deficit bool                 `json:"deficit,omitempty"`
}

type TariffTable struct {
	Code      string       `json:"code"`
	Name      string       `json:"name,omitempty"`
	Axes      []TariffAxis `json:"axes"`
	ValueType string       `json:"valueType"`
	Cells     []TariffCell `json:"cells"`
	Loaded    [][]string   `json:"loaded"`
}

type TariffChargeCode struct {
	Code        string `json:"code"`
	Description string `json:"description,omitempty"`
	Active      bool   `json:"active"`
}

type TariffSettings struct {
	DistanceUnit string `json:"distanceUnit,omitempty"`
}

type TariffContext struct {
	Libraries []TariffLibrary     `json:"libraries,omitempty"`
	Tables    []TariffTable       `json:"tables,omitempty"`
	Codes     *[]TariffChargeCode `json:"codes,omitempty"`
	Settings  TariffSettings      `json:"settings"`
}

type TariffInputValues struct {
	Rule      map[string]RawJSON `json:"rule"`
	Agreement map[string]RawJSON `json:"agreement,omitempty"`
}

type TariffCheckRequest struct {
	Source  string             `json:"source"`
	Kind    TariffKind         `json:"kind,omitempty"`
	Context *TariffContext     `json:"context"`
	Values  *TariffInputValues `json:"values,omitempty"`
}

type TariffChange struct {
	Range  TariffRange `json:"range"`
	Insert string      `json:"insert"`
}

type TariffFix struct {
	Label   Message        `json:"label"`
	Changes []TariffChange `json:"changes"`
}

type TariffRelated struct {
	Range   TariffRange `json:"range"`
	Message Message     `json:"message"`
}

type TariffDiagnostic struct {
	Range    *TariffRange    `json:"range,omitempty"`
	Severity string          `json:"severity"`
	Code     string          `json:"code"`
	Message  Message         `json:"message"`
	Fixes    []TariffFix     `json:"fixes,omitempty"`
	Related  []TariffRelated `json:"related,omitempty"`
	Module   string          `json:"module,omitempty"`
	Path     string          `json:"path,omitempty"`
}

type TariffHeader struct {
	Kind TariffKind `json:"kind"`
	Name string     `json:"name"`
}

type TariffInput struct {
	Name       string       `json:"name"`
	Doc        string       `json:"doc,omitempty"`
	Type       string       `json:"type"`
	Descriptor RawJSON      `json:"descriptor"`
	Required   bool         `json:"required"`
	Default    RawJSON      `json:"default,omitempty"`
	Range      *TariffRange `json:"range,omitempty"`
}

type TariffImport struct {
	Path    string `json:"path"`
	Version string `json:"version,omitempty"`
}

type TariffChargeRef struct {
	Label string      `json:"label"`
	Code  string      `json:"code,omitempty"`
	Range TariffRange `json:"range"`
}

type TariffCheck struct {
	OK          bool               `json:"ok"`
	Header      *TariffHeader      `json:"header"`
	Diagnostics []TariffDiagnostic `json:"diagnostics"`
	Inputs      []TariffInput      `json:"inputs"`
	Imports     []TariffImport     `json:"imports"`
	Outline     RawJSON            `json:"outline,omitempty"`
	Charges     []TariffChargeRef  `json:"charges"`
}

func (c *TariffCheck) FirstError() *TariffDiagnostic {
	for i := range c.Diagnostics {
		if c.Diagnostics[i].Severity == SeverityError {
			return &c.Diagnostics[i]
		}
	}
	return nil
}

type TariffExchange struct {
	Rate     decimal.Decimal `json:"rate"`
	Currency string          `json:"currency"`
}

type TariffProgram struct {
	ID       string             `json:"id"`
	Source   string             `json:"source"`
	Kind     TariffKind         `json:"kind,omitempty"`
	Inputs   map[string]RawJSON `json:"inputs,omitempty"`
	Record   RawJSON            `json:"record,omitempty"`
	Exchange *TariffExchange    `json:"exchange,omitempty"`
}

type TariffEvaluateRequest struct {
	Context  *TariffContext  `json:"context"`
	Record   RawJSON         `json:"record"`
	Programs []TariffProgram `json:"programs"`
	Trace    bool            `json:"trace,omitempty"`
}

type TariffLine struct {
	Kind     TariffLineKind `json:"kind"`
	Label    *string        `json:"label"`
	Code     *string        `json:"code"`
	Freight  bool           `json:"freight"`
	Amount   TariffQuantity `json:"amount"`
	Quantity RawJSON        `json:"quantity,omitempty"`
	Rate     RawJSON        `json:"rate,omitempty"`
	Range    *TariffRange   `json:"range"`
}

type TariffNote struct {
	Text  string       `json:"text"`
	Range *TariffRange `json:"range"`
}

type TariffConverted struct {
	Currency string           `json:"currency"`
	Rate     decimal.Decimal  `json:"rate"`
	Lines    []TariffQuantity `json:"lines"`
	Total    TariffQuantity   `json:"total"`
}

type TariffBill struct {
	Lines            []TariffLine     `json:"lines"`
	Notes            []TariffNote     `json:"notes"`
	FreightTotal     TariffQuantity   `json:"freightTotal"`
	AccessorialTotal TariffQuantity   `json:"accessorialTotal"`
	Total            TariffQuantity   `json:"total"`
	Converted        *TariffConverted `json:"converted,omitempty"`
}

type TariffRefusal struct {
	Text    string
	Message *Message
	Range   *TariffRange
}

type wireTariffRefusal struct {
	Message sonic.NoCopyRawMessage `json:"message"`
	Range   *TariffRange           `json:"range"`
}

func (r *TariffRefusal) UnmarshalJSON(data []byte) error {
	var wire wireTariffRefusal
	if err := sonic.Unmarshal(data, &wire); err != nil {
		return err
	}
	r.Range = wire.Range
	if len(wire.Message) > 0 && wire.Message[0] == '"' {
		return sonic.Unmarshal(wire.Message, &r.Text)
	}
	var message Message
	if err := message.UnmarshalJSON(wire.Message); err != nil {
		return err
	}
	r.Message = &message
	return nil
}

func (r *TariffRefusal) Localize(locale i18n.Locale) string {
	if r.Message != nil {
		return r.Message.Localize(locale)
	}
	return r.Text
}

type TariffEvalError struct {
	Code    string       `json:"code"`
	Message Message      `json:"message"`
	Range   *TariffRange `json:"range"`
}

type TariffNeed struct {
	Table string   `json:"table"`
	Keys  []string `json:"keys"`
}

type TariffResult struct {
	ID      string           `json:"id"`
	Status  TariffStatus     `json:"status"`
	Bill    *TariffBill      `json:"bill,omitempty"`
	Value   *bool            `json:"value,omitempty"`
	Refusal *TariffRefusal   `json:"refusal,omitempty"`
	Error   *TariffEvalError `json:"error,omitempty"`
	Needs   []TariffNeed     `json:"needs,omitempty"`
	Trace   RawJSON          `json:"trace,omitempty"`
	Lookups RawJSON          `json:"lookups,omitempty"`
}

type TariffEvaluation struct {
	Results []TariffResult `json:"results"`
}

func (e *TariffEvaluation) Result(id string) *TariffResult {
	for i := range e.Results {
		if e.Results[i].ID == id {
			return &e.Results[i]
		}
	}
	return nil
}

type TariffTestRequest struct {
	Source  string         `json:"source"`
	Context *TariffContext `json:"context"`
	Names   []string       `json:"names,omitempty"`
}

type TariffTestValue struct {
	Range   TariffRange `json:"range"`
	Display string      `json:"display"`
	Value   RawJSON     `json:"value,omitempty"`
}

type TariffTestFailure struct {
	Range   TariffRange       `json:"range"`
	Message Message           `json:"message"`
	Values  []TariffTestValue `json:"values,omitempty"`
}

type TariffTestResult struct {
	Name     string              `json:"name"`
	Range    TariffRange         `json:"range"`
	Status   TariffTestStatus    `json:"status"`
	Failures []TariffTestFailure `json:"failures,omitempty"`
	Bill     RawJSON             `json:"bill,omitempty"`
	Error    *TariffEvalError    `json:"error,omitempty"`
}

type TariffTestRun struct {
	Tests []TariffTestResult `json:"tests"`
	Error *TariffEvalError   `json:"error,omitempty"`
}

func (r *TariffTestRun) Passed() bool {
	if r.Error != nil {
		return false
	}
	for i := range r.Tests {
		if r.Tests[i].Status != TariffTestPassed {
			return false
		}
	}
	return true
}

type TariffFormatRequest struct {
	Source string `json:"source"`
}

type TariffFormatted struct {
	Formatted   *string            `json:"formatted"`
	Diagnostics []TariffDiagnostic `json:"diagnostics,omitempty"`
}

type TariffScaffoldRequest struct {
	Source  string             `json:"source"`
	Context *TariffContext     `json:"context"`
	Record  RawJSON            `json:"record"`
	Inputs  map[string]RawJSON `json:"inputs,omitempty"`
	Name    string             `json:"name"`
}

type TariffScaffold struct {
	Test   *string          `json:"test"`
	Error  *TariffEvalError `json:"error,omitempty"`
	Status TariffStatus     `json:"status,omitempty"`
	Needs  []TariffNeed     `json:"needs,omitempty"`
}

func (r *Runtime) CheckTariff(ctx context.Context, req *TariffCheckRequest) (*TariffCheck, error) {
	var result TariffCheck
	if err := r.Call(ctx, OpTariffCheck, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *Runtime) EvaluateTariffs(
	ctx context.Context,
	req *TariffEvaluateRequest,
) (*TariffEvaluation, error) {
	var result TariffEvaluation
	if err := r.Call(ctx, OpTariffEvaluate, req, &result); err != nil {
		return nil, err
	}
	if len(result.Results) != len(req.Programs) {
		return nil, fmt.Errorf(
			"%w: %d results for %d programs",
			errTariffResponse,
			len(result.Results),
			len(req.Programs),
		)
	}
	return &result, nil
}

func (r *Runtime) TestTariff(ctx context.Context, req *TariffTestRequest) (*TariffTestRun, error) {
	var result TariffTestRun
	if err := r.Call(ctx, OpTariffTest, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *Runtime) FormatTariff(ctx context.Context, source string) (*TariffFormatted, error) {
	var result TariffFormatted
	request := &TariffFormatRequest{Source: source}
	if err := r.Call(ctx, OpTariffFormat, request, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *Runtime) TariffCatalog(ctx context.Context) (RawJSON, error) {
	return r.CallRaw(ctx, OpTariffCatalog, []byte("{}"))
}

func (r *Runtime) ScaffoldTariffTest(
	ctx context.Context,
	req *TariffScaffoldRequest,
) (*TariffScaffold, error) {
	var result TariffScaffold
	if err := r.Call(ctx, OpTariffScaffoldTest, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func CheckTariff(ctx context.Context, req *TariffCheckRequest) (*TariffCheck, error) {
	return Default().CheckTariff(ctx, req)
}

func EvaluateTariffs(ctx context.Context, req *TariffEvaluateRequest) (*TariffEvaluation, error) {
	return Default().EvaluateTariffs(ctx, req)
}

func TestTariff(ctx context.Context, req *TariffTestRequest) (*TariffTestRun, error) {
	return Default().TestTariff(ctx, req)
}

func FormatTariff(ctx context.Context, source string) (*TariffFormatted, error) {
	return Default().FormatTariff(ctx, source)
}

func TariffCatalog(ctx context.Context) (RawJSON, error) {
	return Default().TariffCatalog(ctx)
}

func ScaffoldTariffTest(ctx context.Context, req *TariffScaffoldRequest) (*TariffScaffold, error) {
	return Default().ScaffoldTariffTest(ctx, req)
}
