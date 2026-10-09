package agentevalgate

import (
	"context"
	"fmt"
	"maps"
	"os"
	"strings"
	"sync"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime"
	"github.com/emoss08/trenova/internal/core/services/agentruntime/agentruntimetest"
	"github.com/emoss08/trenova/shared/mdtable"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/recordids"
	"gopkg.in/yaml.v3"
)

const (
	conversationModel = "conversation-script"
	tableRowPrefix    = "PRO-"
	tableRowStatus    = "Late"
	firstTableRow     = 1001
)

var (
	conversationOrganization = pulid.ID("org_01J9Z3QK8M5T7V2X4Y6W0R1N8P")
	conversationBusinessUnit = pulid.ID("bu_01J9Z3QK8M5T7V2X4Y6W0R1N8P")
	conversationUser         = pulid.ID("usr_01J9Z3QK8M5T7V2X4Y6W0R1N8P")
)

// ConversationSuite is scripted turns that hold the runtime to its contract
// with a model: what a tool receives when the model sends a value the way a
// small model does, what a refusal tells it, and what is left of a reply
// that breaks the rules the prompt gives. The model's side is scripted, so a
// case fails on the runtime and never on a model's mood; a case that fails
// today is a behaviour the runtime does not yet hold.
type ConversationSuite struct {
	Cases []ConversationCase `yaml:"cases"`
}

type ConversationCase struct {
	Name   string             `yaml:"name"`
	Input  string             `yaml:"input"`
	Tools  []ConversationTool `yaml:"tools"`
	Script []ConversationStep `yaml:"script"`
	Expect ConversationExpect `yaml:"expect"`
}

// ConversationTool is a read the agent holds. Result is what it returns; a
// Table instead makes it return that many late shipments and keeps them
// beside the conversation as a table, the way a list tool's result is kept.
type ConversationTool struct {
	Name   string         `yaml:"name"`
	Schema map[string]any `yaml:"schema"`
	Result map[string]any `yaml:"result"`
	Table  *TableFixture  `yaml:"table"`
}

type TableFixture struct {
	Title string `yaml:"title"`
	Rows  int    `yaml:"rows"`
}

// ConversationStep is one completion. ReprintRows writes that many of the
// table fixture's rows out after Reply as a markdown table, which is what a
// small model does with a list it was told to point to.
type ConversationStep struct {
	Reply       string             `yaml:"reply"`
	ReprintRows int                `yaml:"reprintRows"`
	Calls       []ConversationCall `yaml:"calls"`
}

type ConversationCall struct {
	Name string         `yaml:"name"`
	Args map[string]any `yaml:"args"`
}

type ConversationExpect struct {
	// PointsToTable names the tool whose kept table the recorded reply must
	// point to.
	PointsToTable   string `yaml:"pointsToTable"`
	NoMarkdownTable bool   `yaml:"noMarkdownTable"`
	NoRecordIDs     bool   `yaml:"noRecordIds"`
	// NoRefusals says no call was refused.
	NoRefusals bool           `yaml:"noRefusals"`
	Received   []ReceivedCall `yaml:"received"`
	Refusal    *RefusalExpect `yaml:"refusal"`
}

// ReceivedCall is what a tool must have been handed: on its Call'th call,
// counting from one, the arguments Args, and Calls times in all.
type ReceivedCall struct {
	Tool  string         `yaml:"tool"`
	Call  int            `yaml:"call"`
	Calls int            `yaml:"calls"`
	Args  map[string]any `yaml:"args"`
}

// RefusalExpect is a refusal the model must have been told, at least once,
// for a call of Tool, and the words it must name.
type RefusalExpect struct {
	Tool     string   `yaml:"tool"`
	Mentions []string `yaml:"mentions"`
}

type ConversationOutcome struct {
	Case     *ConversationCase
	Result   *serviceports.RunResult
	Received map[string][]map[string]any
	Tables   map[string]serviceports.ShownArtifact
}

func LoadConversationSuite(path string) (*ConversationSuite, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var suite ConversationSuite
	if err = yaml.Unmarshal(raw, &suite); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	return &suite, nil
}

// recordingTool is a stub read that keeps every set of arguments it was
// handed, after the runtime read them, in the order it was called.
type recordingTool struct {
	*agentruntimetest.StubQueryTool

	mu       sync.Mutex
	received []map[string]any
}

func (t *recordingTool) Query(
	ctx context.Context,
	params *serviceports.QueryToolParams,
) (any, error) {
	t.mu.Lock()
	t.received = append(t.received, maps.Clone(params.Params))
	t.mu.Unlock()

	return t.StubQueryTool.Query(ctx, params)
}

// RunConversation drives one case through the runtime the kit builds, with
// the case's tools as the only ones registered.
func RunConversation(ctx context.Context, c *ConversationCase) (*ConversationOutcome, error) {
	tools := make([]*recordingTool, 0, len(c.Tools))
	queries := make([]serviceports.AgentQueryTool, 0, len(c.Tools))
	names := make([]string, 0, len(c.Tools))
	tables := make(map[string]*TableFixture, len(c.Tools))
	for idx := range c.Tools {
		spec := &c.Tools[idx]
		result, err := toolResult(spec)
		if err != nil {
			return nil, err
		}
		tool := &recordingTool{StubQueryTool: &agentruntimetest.StubQueryTool{
			ToolName: spec.Name,
			Desc:     "Scripted read " + spec.Name + ".",
			Schema:   spec.Schema,
			Result:   result,
		}}
		tools = append(tools, tool)
		queries = append(queries, tool)
		names = append(names, spec.Name)
		if spec.Table != nil {
			tables[spec.Name] = spec.Table
		}
	}

	script, err := conversationScript(c, tables)
	if err != nil {
		return nil, err
	}

	kit := FromTools(queries, nil, agentruntime.RuntimePolicies())
	rt := kit.NewRuntime(RuntimeParams{
		Completion:  script,
		Permissions: &agentruntimetest.StubPermissions{},
	})

	outcome := &ConversationOutcome{
		Case:     c,
		Received: make(map[string][]map[string]any, len(tools)),
		Tables:   make(map[string]serviceports.ShownArtifact, len(tables)),
	}
	var shown sync.Mutex
	definition := &agentdefinition.Definition{
		Name:            "Scripted agent",
		Instructions:    "Work the organization's records as the tools allow.",
		ToolNames:       names,
		AutonomyCeiling: agent.TierActWithApproval,
		TriggerMode:     agentdefinition.TriggerChat,
		Enabled:         true,
	}
	definition.ApplyDefaults()

	result, err := rt.Run(ctx, &serviceports.RunRequest{
		Definition: definition,
		Actor: &serviceports.RequestActor{
			PrincipalType:  serviceports.PrincipalTypeUser,
			PrincipalID:    conversationUser,
			UserID:         conversationUser,
			OrganizationID: conversationOrganization,
			BusinessUnitID: conversationBusinessUnit,
		},
		Input: c.Input,
		ToolObserver: func(
			observation serviceports.ToolObservation,
		) (*serviceports.ShownArtifact, error) {
			table, kept := tables[observation.Call.Name]
			if !kept || observation.Failed {
				return nil, nil //nolint:nilnil // a call that showed nothing
			}
			artifact := tableArtifact(table)
			shown.Lock()
			outcome.Tables[observation.Call.Name] = artifact
			shown.Unlock()

			return &artifact, nil
		},
	})
	if err != nil {
		return nil, err
	}

	outcome.Result = result
	for _, tool := range tools {
		outcome.Received[tool.ToolName] = tool.received
	}

	return outcome, nil
}

func toolResult(spec *ConversationTool) (any, error) {
	if spec.Table == nil {
		return normalized(spec.Result)
	}
	rows := make([]any, 0, spec.Table.Rows)
	for idx := range spec.Table.Rows {
		rows = append(rows, map[string]any{
			"proNumber": tableLabel(idx),
			"status":    tableRowStatus,
		})
	}

	return map[string]any{
		"items":   rows,
		"count":   float64(spec.Table.Rows),
		"columns": []any{"proNumber", "status"},
	}, nil
}

func tableLabel(idx int) string {
	return fmt.Sprintf("%s%d", tableRowPrefix, firstTableRow+idx)
}

func tableArtifact(table *TableFixture) serviceports.ShownArtifact {
	labels := make([]string, 0, table.Rows)
	for idx := range table.Rows {
		labels = append(labels, tableLabel(idx))
	}

	return serviceports.ShownArtifact{
		ID:     pulid.MustNew("aart_"),
		Kind:   "table_view",
		Title:  table.Title,
		Rows:   table.Rows,
		Labels: labels,
	}
}

// conversationScript is the case's completions. Arguments go through JSON,
// as a model's do, so a number in the YAML arrives as the float64 a provider
// hands the runtime.
func conversationScript(
	c *ConversationCase,
	tables map[string]*TableFixture,
) (*agentruntimetest.ScriptedCompletion, error) {
	turns := make([]*serviceports.ChatCompletionResult, 0, len(c.Script))
	for stepIdx := range c.Script {
		step := &c.Script[stepIdx]
		result := &serviceports.ChatCompletionResult{
			Text:            step.Reply + reprint(step.ReprintRows, tables),
			ModelIdentifier: conversationModel,
		}
		for callIdx := range step.Calls {
			call := &step.Calls[callIdx]
			args, err := normalized(call.Args)
			if err != nil {
				return nil, err
			}
			result.ToolCalls = append(result.ToolCalls, serviceports.ToolCall{
				ID:        fmt.Sprintf("call_%d_%d", stepIdx, callIdx),
				Name:      call.Name,
				Arguments: args,
			})
		}
		turns = append(turns, result)
	}

	return &agentruntimetest.ScriptedCompletion{Turns: turns}, nil
}

func reprint(rows int, tables map[string]*TableFixture) string {
	if rows == 0 || len(tables) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("\n\n| Pro | Status |\n| --- | --- |\n")
	for idx := range rows {
		fmt.Fprintf(&b, "| **%s** | %s |\n", tableLabel(idx), tableRowStatus)
	}

	return b.String()
}

func normalized(value map[string]any) (map[string]any, error) {
	if value == nil {
		return map[string]any{}, nil
	}
	encoded, err := sonic.Marshal(value)
	if err != nil {
		return nil, err
	}
	out := make(map[string]any, len(value))
	if err = sonic.Unmarshal(encoded, &out); err != nil {
		return nil, err
	}

	return out, nil
}

// Failures is every expectation the outcome missed, in words a reviewer can
// act on. None is a pass.
func (o *ConversationOutcome) Failures() []string {
	expect := o.Case.Expect
	reply := o.Result.Reply
	failures := make([]string, 0, 4)

	if tool := expect.PointsToTable; tool != "" {
		table, kept := o.Tables[tool]
		switch {
		case !kept:
			failures = append(failures, tool+" kept no table")
		case !strings.Contains(reply, agentruntime.ArtifactRef(&table)):
			failures = append(failures, "the reply does not point to "+tool+"'s table")
		}
	}
	if expect.NoMarkdownTable && len(mdtable.Find(reply)) > 0 {
		failures = append(failures, "the reply still writes out a markdown table")
	}
	if expect.NoRecordIDs && recordids.Contains(reply) {
		failures = append(failures, "the reply still names an internal record id")
	}

	refusals := o.refusals()
	if expect.NoRefusals && len(refusals) > 0 {
		failures = append(failures, fmt.Sprintf("a call was refused: %s", refusals[0].Content))
	}
	if expect.Refusal != nil {
		failures = append(failures, o.refusalFailures(expect.Refusal, refusals)...)
	}
	for idx := range expect.Received {
		failures = append(failures, o.receivedFailures(&expect.Received[idx])...)
	}

	return failures
}

func (o *ConversationOutcome) refusals() []conversation.Message {
	out := make([]conversation.Message, 0, 2)
	for idx := range o.Result.Messages {
		message := o.Result.Messages[idx]
		if message.Role == conversation.RoleTool && message.ToolFailed {
			out = append(out, message)
		}
	}

	return out
}

func (o *ConversationOutcome) refusalFailures(
	expect *RefusalExpect,
	refusals []conversation.Message,
) []string {
	for idx := range refusals {
		refusal := &refusals[idx]
		if refusal.ToolName != expect.Tool {
			continue
		}
		missing := make([]string, 0, len(expect.Mentions))
		for _, mention := range expect.Mentions {
			if !strings.Contains(refusal.Content, mention) {
				missing = append(missing, mention)
			}
		}
		if len(missing) == 0 {
			return nil
		}

		return []string{fmt.Sprintf("the refusal of %s does not name %s: %s",
			expect.Tool, strings.Join(missing, ", "), refusal.Content)}
	}

	return []string{"no call of " + expect.Tool + " was refused"}
}

func (o *ConversationOutcome) receivedFailures(expect *ReceivedCall) []string {
	received := o.Received[expect.Tool]
	failures := make([]string, 0, 2)
	if expect.Calls > 0 && len(received) != expect.Calls {
		failures = append(failures, fmt.Sprintf("%s ran %d times, want %d",
			expect.Tool, len(received), expect.Calls))
	}
	if expect.Args == nil {
		return failures
	}
	call := max(expect.Call, 1)
	if len(received) < call {
		return append(failures, fmt.Sprintf("%s never ran a call %d", expect.Tool, call))
	}
	want, err := normalized(expect.Args)
	if err != nil {
		return append(failures, err.Error())
	}
	got, err := normalized(received[call-1])
	if err != nil {
		return append(failures, err.Error())
	}
	for key, value := range want {
		if !sameJSON(got[key], value) {
			failures = append(failures, fmt.Sprintf("%s call %d received %s = %v, want %v",
				expect.Tool, call, key, got[key], value))
		}
	}

	return failures
}

func sameJSON(left, right any) bool {
	a, errA := sonic.Marshal(left)
	b, errB := sonic.Marshal(right)

	return errA == nil && errB == nil && string(a) == string(b)
}
