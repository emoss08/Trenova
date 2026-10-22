package rules

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"sync"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/shopspring/decimal"
)

const (
	OpFormulaCatalog  = "formula.catalog"
	OpFormulaCheck    = "formula.check"
	OpFormulaEvaluate = "formula.evaluate"

	SeverityError   = "error"
	SeverityWarning = "warning"

	DiagnosticNullable = "nullable"

	ErrorMissingField = "missingField"
)

var errFormulaResponse = errors.New("rules: the formula module answered with an unexpected shape")

type FormulaExpect string

const (
	ExpectAmount    = FormulaExpect("amount")
	ExpectPredicate = FormulaExpect("predicate")
)

type FormulaVariable struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Nullable    bool     `json:"nullable"`
	Computed    bool     `json:"computed"`
	Enum        []string `json:"enum,omitempty"`
}

type FormulaInput struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type FormulaSchema struct {
	Variables []FormulaVariable `json:"variables,omitempty"`
	Inputs    []FormulaInput    `json:"inputs,omitempty"`
	Operators []string          `json:"operators,omitempty"`
}

type FormulaTable struct {
	Name   string `json:"name"`
	Axes   int    `json:"axes"`
	Banded bool   `json:"banded"`
}

type CheckRequest struct {
	Source string         `json:"source"`
	Expect FormulaExpect  `json:"expect,omitempty"`
	Schema *FormulaSchema `json:"schema"`
	Tables []FormulaTable `json:"tables"`
}

type FormulaChange struct {
	From   int    `json:"from"`
	To     int    `json:"to"`
	Insert string `json:"insert"`
}

type FormulaFix struct {
	Kind    string          `json:"kind"`
	Label   Message         `json:"label"`
	Changes []FormulaChange `json:"changes,omitempty"`
	Name    string          `json:"name,omitempty"`
	Type    string          `json:"type,omitempty"`
}

type FormulaDiagnostic struct {
	From     int
	To       int
	Severity string
	Code     string
	Message  Message
	Name     string
	Fixes    []FormulaFix
}

type wireDiagnostic struct {
	From     int          `json:"from"`
	To       int          `json:"to"`
	Severity string       `json:"severity"`
	Code     string       `json:"code"`
	Name     string       `json:"name,omitempty"`
	Fixes    []FormulaFix `json:"fixes"`
}

func (d *FormulaDiagnostic) UnmarshalJSON(data []byte) error {
	var wire wireDiagnostic
	if err := sonic.Unmarshal(data, &wire); err != nil {
		return err
	}
	if err := d.Message.UnmarshalJSON(data); err != nil {
		return err
	}
	d.From, d.To, d.Severity, d.Code = wire.From, wire.To, wire.Severity, wire.Code
	d.Name, d.Fixes = wire.Name, wire.Fixes
	return nil
}

//nolint:gocritic // hugeParam: a value receiver, so a value held by value marshals through it
func (d FormulaDiagnostic) MarshalJSON() ([]byte, error) {
	return d.AppendJSON(make([]byte, 0, 256), "")
}

func (d *FormulaDiagnostic) AppendJSON(dst []byte, scope string) ([]byte, error) {
	dst = append(dst, '{')
	if scope != "" {
		dst = append(dst, `"scope":`...)
		dst = appendString(dst, scope)
		dst = append(dst, ',')
	}
	dst = append(dst, `"from":`...)
	dst = strconv.AppendInt(dst, int64(d.From), 10)
	dst = append(dst, `,"to":`...)
	dst = strconv.AppendInt(dst, int64(d.To), 10)
	dst = append(dst, `,"severity":`...)
	dst = appendString(dst, d.Severity)
	dst = append(dst, `,"code":`...)
	dst = appendString(dst, d.Code)
	dst, err := appendFlatMessage(dst, d.Message)
	if err != nil {
		return nil, err
	}
	if d.Name != "" {
		dst = append(dst, `,"name":`...)
		dst = appendString(dst, d.Name)
	}
	fixes := d.Fixes
	if fixes == nil {
		fixes = []FormulaFix{}
	}
	encoded, err := sonic.Marshal(fixes)
	if err != nil {
		return nil, err
	}
	dst = append(dst, `,"fixes":`...)
	dst = append(dst, encoded...)
	return append(dst, '}'), nil
}

func (d *FormulaDiagnostic) IsError() bool {
	return d.Severity == SeverityError
}

type FormulaBinding struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Reads    int    `json:"reads"`
	From     int    `json:"from"`
	To       int    `json:"to"`
	NameFrom int    `json:"nameFrom"`
	NameTo   int    `json:"nameTo"`
}

type FormulaStatement struct {
	Index int    `json:"index"`
	Kind  string `json:"kind"`
	Name  string `json:"name,omitempty"`
	Code  string `json:"code"`
	From  int    `json:"from"`
	To    int    `json:"to"`
}

type FormulaTableRef struct {
	Name     string `json:"name"`
	Function string `json:"function"`
	From     int    `json:"from"`
	To       int    `json:"to"`
}

type FormulaCheck struct {
	Empty         bool                `json:"empty"`
	Pending       bool                `json:"pending"`
	Diagnostics   []FormulaDiagnostic `json:"diagnostics"`
	ResultType    string              `json:"resultType"`
	Reads         []string            `json:"reads"`
	SchemaReads   []string            `json:"schemaReads"`
	InputReads    []string            `json:"inputReads"`
	FunctionCalls []string            `json:"functionCalls"`
	Bindings      []FormulaBinding    `json:"bindings"`
	Statements    []FormulaStatement  `json:"statements"`
	LookupTables  []FormulaTableRef   `json:"lookupTables"`
}

func (c *FormulaCheck) Accepted() bool {
	return c.FirstError() == nil
}

func (c *FormulaCheck) FirstError() *FormulaDiagnostic {
	for index := range c.Diagnostics {
		if c.Diagnostics[index].IsError() {
			return &c.Diagnostics[index]
		}
	}
	return nil
}

func (c *FormulaCheck) Unguarded() []string {
	names := make([]string, 0, 2)
	for index := range c.Diagnostics {
		diagnostic := &c.Diagnostics[index]
		if diagnostic.Code == DiagnosticNullable && diagnostic.Name != "" &&
			!slices.Contains(names, diagnostic.Name) {
			names = append(names, diagnostic.Name)
		}
	}
	return names
}

func (c *FormulaCheck) ReadsField(name string) bool {
	return slices.Contains(c.Reads, name)
}

type FormulaError struct {
	Code    string
	Message Message
	From    *int
	To      *int
	Fields  []string
	Guards  []string
}

type wireFormulaError struct {
	Code   string   `json:"code"`
	From   *int     `json:"from"`
	To     *int     `json:"to"`
	Fields []string `json:"fields"`
	Guards []string `json:"guards"`
}

func (e *FormulaError) Error() string {
	return e.Message.String()
}

func (e *FormulaError) Localize(locale i18n.Locale) string {
	return e.Message.Localize(locale)
}

func (e *FormulaError) LocalizedMessage() (message string, args []any) {
	return e.Message.LocalizedMessage()
}

func (e *FormulaError) UnmarshalJSON(data []byte) error {
	var wire wireFormulaError
	if err := sonic.Unmarshal(data, &wire); err != nil {
		return err
	}
	if err := e.Message.UnmarshalJSON(data); err != nil {
		return err
	}
	e.Code, e.From, e.To, e.Fields, e.Guards = wire.Code, wire.From, wire.To, wire.Fields, wire.Guards
	return nil
}

//nolint:gocritic // hugeParam: a value receiver, so a value held by value marshals through it
func (e FormulaError) MarshalJSON() ([]byte, error) {
	return e.AppendJSON(make([]byte, 0, 256), "")
}

func (e *FormulaError) AppendJSON(dst []byte, scope string) ([]byte, error) {
	dst = append(dst, '{')
	if scope != "" {
		dst = append(dst, `"scope":`...)
		dst = appendString(dst, scope)
		dst = append(dst, ',')
	}
	dst = append(dst, `"code":`...)
	dst = appendString(dst, e.Code)
	dst, err := appendFlatMessage(dst, e.Message)
	if err != nil {
		return nil, err
	}
	if e.From != nil && e.To != nil {
		dst = append(dst, `,"from":`...)
		dst = strconv.AppendInt(dst, int64(*e.From), 10)
		dst = append(dst, `,"to":`...)
		dst = strconv.AppendInt(dst, int64(*e.To), 10)
	}
	dst = appendStrings(dst, "fields", e.Fields)
	dst = appendStrings(dst, "guards", e.Guards)
	return append(dst, '}'), nil
}

func appendFlatMessage(dst []byte, message Message) ([]byte, error) {
	dst = append(dst, `,"message":`...)
	dst = appendString(dst, message.Template)
	dst = append(dst, `,"args":`...)
	return appendMessageArgs(dst, message.Args)
}

func appendStrings(dst []byte, key string, values []string) []byte {
	dst = append(dst, ',', '"')
	dst = append(dst, key...)
	dst = append(dst, '"', ':', '[')
	for index, value := range values {
		if index > 0 {
			dst = append(dst, ',')
		}
		dst = appendString(dst, value)
	}
	return append(dst, ']')
}

type FormulaProgram struct {
	ID     string        `json:"id"`
	Source string        `json:"source"`
	Expect FormulaExpect `json:"expect"`
}

type FormulaBudget struct {
	Steps     uint64 `json:"steps,omitempty"`
	TextBytes uint64 `json:"textBytes,omitempty"`
	Items     uint64 `json:"items,omitempty"`
}

type EvaluateRequest struct {
	Programs        []FormulaProgram
	Operators       []string
	Env             map[string]any
	EncodedEnv      RawJSON
	Tables          RawJSON
	TraceStatements bool
	TraceLookups    bool
	Budget          FormulaBudget
}

type FormulaLookupMatch struct {
	MatchedKey string           `json:"matchedKey,omitempty"`
	BandMin    *decimal.Decimal `json:"bandMin,omitempty"`
	BandMax    *decimal.Decimal `json:"bandMax,omitempty"`
	Adjusted   bool             `json:"adjusted,omitempty"`
}

type FormulaLookup struct {
	Function string              `json:"function"`
	Table    string              `json:"table"`
	Keys     []RawJSON           `json:"keys"`
	Value    *decimal.Decimal    `json:"value,omitempty"`
	Match    *FormulaLookupMatch `json:"match,omitempty"`
	Error    *FormulaError       `json:"error,omitempty"`
}

type FormulaTraceStatement struct {
	Index int           `json:"index"`
	Kind  string        `json:"kind"`
	Name  string        `json:"name,omitempty"`
	Code  string        `json:"code"`
	From  int           `json:"from"`
	To    int           `json:"to"`
	Value RawJSON       `json:"value,omitempty"`
	Error *FormulaError `json:"error,omitempty"`
}

type FormulaResult struct {
	ID         string                  `json:"id"`
	Value      RawJSON                 `json:"value"`
	Error      *FormulaError           `json:"error"`
	Statements []FormulaTraceStatement `json:"statements"`
	Lookups    []FormulaLookup         `json:"lookups"`
}

func (r *FormulaResult) Amount() (decimal.Decimal, error) {
	if r.Error != nil {
		return decimal.Zero, r.Error
	}
	var text string
	if err := sonic.Unmarshal(r.Value, &text); err != nil {
		return decimal.Zero, fmt.Errorf("%w: %s is not an amount", errFormulaResponse, r.Value)
	}
	amount, err := decimal.NewFromString(text)
	if err != nil {
		return decimal.Zero, fmt.Errorf("%w: %w", errFormulaResponse, err)
	}
	return amount, nil
}

func (r *FormulaResult) Holds() (bool, error) {
	if r.Error != nil {
		return false, r.Error
	}
	switch string(r.Value) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	}
	amount, err := r.Amount()
	if err != nil {
		return false, err
	}
	return !amount.IsZero(), nil
}

type EvaluateResponse struct {
	Results []FormulaResult `json:"results"`
}

func (r *EvaluateResponse) Result(id string) *FormulaResult {
	for index := range r.Results {
		if r.Results[index].ID == id {
			return &r.Results[index]
		}
	}
	return nil
}

type FormulaFunction struct {
	Name        string  `json:"name"`
	Signature   string  `json:"signature"`
	Description Message `json:"description"`
	Example     Message `json:"example"`
	Category    string  `json:"category"`
	Operator    bool    `json:"operator"`
	MinArgs     int     `json:"minArgs"`
	MaxArgs     int     `json:"maxArgs"`
	Returns     string  `json:"returns"`
	NumericArgs bool    `json:"numericArgs"`
	Predicate   bool    `json:"predicate"`
}

type FormulaDisabled struct {
	Name        string  `json:"name"`
	Reason      string  `json:"reason"`
	Replacement string  `json:"replacement,omitempty"`
	Message     Message `json:"message"`
}

type FormulaCatalog struct {
	Functions []FormulaFunction `json:"functions"`
	Builtins  []FormulaFunction `json:"builtins"`
	Disabled  []FormulaDisabled `json:"disabled"`
	Reserved  []string          `json:"reserved"`
}

func (c *FormulaCatalog) IsReserved(name string) bool {
	_, found := slices.BinarySearch(c.Reserved, name)
	return found
}

func (r *Runtime) FormulaCatalog(ctx context.Context) (*FormulaCatalog, error) {
	var catalog FormulaCatalog
	if err := r.Call(ctx, OpFormulaCatalog, struct{}{}, &catalog); err != nil {
		return nil, err
	}
	slices.Sort(catalog.Reserved)
	return &catalog, nil
}

func (r *Runtime) CheckFormula(ctx context.Context, req *CheckRequest) (*FormulaCheck, error) {
	var result FormulaCheck
	if err := r.Call(ctx, OpFormulaCheck, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *Runtime) EvaluateFormulas(
	ctx context.Context,
	req *EvaluateRequest,
) (*EvaluateResponse, error) {
	input, err := encodeEvaluateRequest(req)
	if err != nil {
		return nil, fmt.Errorf("rules: %s: encoding the input: %w", OpFormulaEvaluate, err)
	}
	output, err := r.CallRaw(ctx, OpFormulaEvaluate, input)
	if err != nil {
		return nil, err
	}
	var response EvaluateResponse
	if err = sonic.Unmarshal(output, &response); err != nil {
		return nil, fmt.Errorf("rules: %s: decoding the output: %w", OpFormulaEvaluate, err)
	}
	if len(response.Results) != len(req.Programs) {
		return nil, fmt.Errorf(
			"%w: %d results for %d programs",
			errFormulaResponse,
			len(response.Results),
			len(req.Programs),
		)
	}
	return &response, nil
}

func EncodeEnv(env map[string]any) (RawJSON, error) {
	if env == nil {
		return RawJSON("{}"), nil
	}
	encoded, err := appendStringMap(make([]byte, 0, 2048), env, "")
	if err != nil {
		return nil, err
	}
	return encoded, nil
}

func encodeEvaluateRequest(req *EvaluateRequest) ([]byte, error) {
	programs, err := sonic.Marshal(req.Programs)
	if err != nil {
		return nil, err
	}
	env := req.EncodedEnv
	if env == nil {
		if env, err = EncodeEnv(req.Env); err != nil {
			return nil, err
		}
	}
	dst := make([]byte, 0, len(programs)+len(env)+len(req.Tables)+128)
	dst = append(dst, `{"programs":`...)
	dst = append(dst, programs...)
	if len(req.Operators) > 0 {
		operators, opErr := sonic.Marshal(req.Operators)
		if opErr != nil {
			return nil, opErr
		}
		dst = append(dst, `,"schema":{"operators":`...)
		dst = append(dst, operators...)
		dst = append(dst, '}')
	}
	dst = append(dst, `,"env":`...)
	dst = append(dst, env...)
	if req.Tables != nil {
		dst = append(dst, `,"tables":`...)
		dst = append(dst, req.Tables...)
	}
	dst = append(dst, `,"trace":{"statements":`...)
	dst = strconv.AppendBool(dst, req.TraceStatements)
	dst = append(dst, `,"lookups":`...)
	dst = strconv.AppendBool(dst, req.TraceLookups)
	dst = append(dst, '}')
	if req.Budget != (FormulaBudget{}) {
		budget, budgetErr := sonic.Marshal(req.Budget)
		if budgetErr != nil {
			return nil, budgetErr
		}
		dst = append(dst, `,"budget":`...)
		dst = append(dst, budget...)
	}
	return append(dst, '}'), nil
}

var defaultCatalog struct {
	mu      sync.Mutex
	catalog *FormulaCatalog
}

func Catalog(ctx context.Context) (*FormulaCatalog, error) {
	defaultCatalog.mu.Lock()
	defer defaultCatalog.mu.Unlock()
	if defaultCatalog.catalog != nil {
		return defaultCatalog.catalog, nil
	}
	catalog, err := Default().FormulaCatalog(ctx)
	if err != nil {
		return nil, err
	}
	defaultCatalog.catalog = catalog
	return catalog, nil
}

func CheckFormula(ctx context.Context, req *CheckRequest) (*FormulaCheck, error) {
	return Default().CheckFormula(ctx, req)
}

func EvaluateFormulas(ctx context.Context, req *EvaluateRequest) (*EvaluateResponse, error) {
	return Default().EvaluateFormulas(ctx, req)
}
