package agentruntime

import (
	"regexp"
	"slices"
	"strings"
	"unicode"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/numberguard"
	"github.com/emoss08/trenova/shared/stringutils"
)

const changeGroundingGuard = "agent-loop-grounding-guard"

const (
	groundingCorrectionLead = "Your reply does not match what this turn filed."
	groundingRetryReason    = "The reply named something the filed change does not hold, so it " +
		"is being written again."
	groundingNote = "\n\n_Check the card before relying on this reply: parts of it may " +
		"describe more than the proposal holds._"
	groundingNoteReason = "The rewritten reply still named something the filed change does " +
		"not hold, so it ends by pointing to the card."
	minLabelWords = 2
)

var (
	schemaPropertyPath = regexp.MustCompile(`^[A-Za-z0-9_]+(\[\])?(\.[A-Za-z0-9_]+(\[\])?)*$`)
	negationWords      = []string{
		"no", "not", "none", "never", "without", "nor", "omit", "omitted", "omitting",
		"excluded", "exclude", "excluding", "skip", "skipped", "missing", "left",
		"cannot", "unable",
	}
)

type groundingState struct {
	rewritten bool
}

type groundingFinding struct {
	figures []string
	fields  []string
	tools   map[string][]string
}

func (f groundingFinding) empty() bool {
	return len(f.figures) == 0 && len(f.fields) == 0
}

type schemaLabel struct {
	path  string
	words string
}

func (s *Service) reground(
	t *Turn,
	fx TurnEffects,
	completion *serviceports.ChatCompletionResult,
	state *groundingState,
) bool {
	if completion == nil || len(t.result.Actions) == 0 {
		return false
	}

	finding := s.groundingFindings(t, completion.Text)
	if finding.empty() || !fx.Supports(changeGroundingGuard) {
		return false
	}

	if !state.rewritten {
		state.rewritten = true
		fx.Emit(serviceports.StreamEvent{
			Event: serviceports.AssistantEventRetrying,
			Data: serviceports.AssistantRetryingEvent{
				Attempt: 1,
				Reason:  groundingRetryReason,
				Kind:    serviceports.RetryKindRestart,
			},
		})
		fx.Emit(regroundedEvent(serviceports.RegroundRewrite, finding, groundingRetryReason))
		t.messages = append(t.messages,
			serviceports.Message{Role: serviceports.RoleAssistant, Content: completion.Text},
			serviceports.Message{Role: serviceports.RoleUser, Content: finding.correction()},
		)

		return true
	}

	fx.Emit(regroundedEvent(serviceports.RegroundNote, finding, groundingNoteReason))
	completion.Text += groundingNote
	fx.Emit(deltaEvent(groundingNote))

	return false
}

func regroundedEvent(
	action serviceports.RegroundAction,
	finding groundingFinding,
	reason string,
) serviceports.StreamEvent {
	return serviceports.StreamEvent{
		Event: serviceports.AssistantEventReplyRegrounded,
		Data: serviceports.AssistantReplyRegroundedEvent{
			Action:  action,
			Figures: finding.figures,
			Fields:  finding.fields,
			Reason:  reason,
		},
	}
}

func (f groundingFinding) correction() string {
	var b strings.Builder
	b.WriteString(groundingCorrectionLead)
	if len(f.figures) > 0 {
		b.WriteString(" It cites figures nothing this turn returned or filed: ")
		b.WriteString(strings.Join(f.figures, ", "))
		b.WriteString(".")
	}
	for _, field := range f.fields {
		b.WriteString(" It says ")
		b.WriteString(field)
		b.WriteString(" are part of the change, but the ")
		b.WriteString(strings.Join(f.tools[field], " and "))
		b.WriteString(" call that was filed carries none.")
	}
	b.WriteString(" Write the reply again from the tool results and the filed proposal " +
		"only. Where they do not say, say you do not know and point the person to the card.")

	return b.String()
}

func (s *Service) groundingFindings(t *Turn, reply string) groundingFinding {
	finding := groundingFinding{tools: make(map[string][]string, 2)}
	if strings.TrimSpace(reply) == "" {
		return finding
	}

	check := numberguard.CheckNumbers(reply, numberguard.SupportedFromText(t.groundingTexts()...))
	finding.figures = check.Unsupported

	claims := claimedPhrases(reply)
	for _, tool := range filedTools(t.result.Actions) {
		spec, ok := s.specFor(tool)
		if !ok {
			continue
		}
		filed := filedFor(t, tool)
		for _, label := range schemaLabels(spec.Parameters) {
			if !claims.names(label.words) || filed.holds(label) {
				continue
			}
			if !slices.Contains(finding.fields, label.words) {
				finding.fields = append(finding.fields, label.words)
			}
			if !slices.Contains(finding.tools[label.words], tool) {
				finding.tools[label.words] = append(finding.tools[label.words], tool)
			}
		}
	}
	slices.Sort(finding.fields)

	return finding
}

func (t *Turn) groundingTexts() []string {
	texts := make([]string, 0, len(t.messages)+len(t.result.Actions)+1)
	texts = append(texts, t.system)
	for idx := range t.messages {
		message := &t.messages[idx]
		if message.Role == serviceports.RoleAssistant ||
			strings.HasPrefix(message.Content, groundingCorrectionLead) {
			continue
		}
		texts = append(texts, message.Content)
	}
	for idx := range t.result.Actions {
		if encoded, err := sonic.MarshalString(t.result.Actions[idx].Arguments); err == nil {
			texts = append(texts, encoded)
		}
	}

	return texts
}

func filedTools(actions []serviceports.PendingAction) []string {
	tools := make([]string, 0, len(actions))
	for idx := range actions {
		if !slices.Contains(tools, actions[idx].ToolName) {
			tools = append(tools, actions[idx].ToolName)
		}
	}

	return tools
}

type filedCalls struct {
	arguments []map[string]any
	results   []string
}

func filedFor(t *Turn, tool string) filedCalls {
	var filed filedCalls
	callIDs := make([]string, 0, 2)
	for idx := range t.result.Actions {
		action := &t.result.Actions[idx]
		if action.ToolName != tool {
			continue
		}
		filed.arguments = append(filed.arguments, action.Arguments)
		callIDs = append(callIDs, action.ToolCallID)
	}
	for idx := range t.result.Messages {
		message := &t.result.Messages[idx]
		if message.Role == conversation.RoleTool && slices.Contains(callIDs, message.ToolCallID) {
			filed.results = append(filed.results, strings.ToLower(message.Content))
		}
	}

	return filed
}

func (f filedCalls) holds(label schemaLabel) bool {
	for _, arguments := range f.arguments {
		if valueAtPath(arguments, strings.Split(label.path, ".")) {
			return true
		}
	}
	for _, result := range f.results {
		if containsWords(result, label.words) {
			return true
		}
	}

	return false
}

func valueAtPath(value any, segments []string) bool {
	if len(segments) == 0 {
		return present(value)
	}

	fields, ok := value.(map[string]any)
	if !ok {
		return false
	}
	name, each := strings.CutSuffix(segments[0], "[]")
	next, found := fields[name]
	if !found {
		return false
	}
	if !each {
		return valueAtPath(next, segments[1:])
	}

	items, _ := next.([]any)
	for _, item := range items {
		if valueAtPath(item, segments[1:]) {
			return true
		}
	}

	return false
}

func present(value any) bool {
	switch typed := value.(type) {
	case nil:
		return false
	case string:
		return strings.TrimSpace(typed) != ""
	case []any:
		return len(typed) > 0
	case map[string]any:
		return len(typed) > 0
	default:
		return true
	}
}

func schemaLabels(schema map[string]any) []schemaLabel {
	labels := make([]schemaLabel, 0, 8)
	toolschema.Walk(schema, func(path string, node map[string]any) {
		if path == "" || strings.HasSuffix(path, "[]") || !schemaPropertyPath.MatchString(path) {
			return
		}
		name := path[strings.LastIndex(path, ".")+1:]
		if strings.HasSuffix(name, "Id") || strings.HasSuffix(name, "Ids") {
			return
		}
		words := strings.ToLower(strings.ReplaceAll(stringutils.ConvertCamelToSnake(name), "_", " "))
		kind, _ := node["type"].(string)
		collection := kind == "array" || kind == "object"
		if !collection && len(strings.Fields(words)) < minLabelWords {
			return
		}
		labels = append(labels, schemaLabel{path: path, words: words})
	})

	return labels
}

type replyClaims []string

func claimedPhrases(reply string) replyClaims {
	sentences := strings.FieldsFunc(strings.ToLower(reply), func(r rune) bool {
		return r == '.' || r == '!' || r == '?' || r == ';' || r == '\n'
	})
	claims := make(replyClaims, 0, len(sentences))
	for _, sentence := range sentences {
		if !negated(sentence) {
			claims = append(claims, sentence)
		}
	}

	return claims
}

func (c replyClaims) names(words string) bool {
	singular := singularOf(words)
	for _, sentence := range c {
		if containsWords(sentence, words) || (singular != words && containsWords(sentence, singular)) {
			return true
		}
	}

	return false
}

func negated(sentence string) bool {
	for _, word := range strings.FieldsFunc(sentence, func(r rune) bool {
		return !unicode.IsLetter(r) && r != '\''
	}) {
		if strings.HasSuffix(word, "n't") || slices.Contains(negationWords, word) {
			return true
		}
	}

	return false
}

func singularOf(words string) string {
	switch {
	case strings.HasSuffix(words, "ies"):
		return strings.TrimSuffix(words, "ies") + "y"
	case strings.HasSuffix(words, "ses"), strings.HasSuffix(words, "xes"):
		return strings.TrimSuffix(words, "es")
	case strings.HasSuffix(words, "s") && !strings.HasSuffix(words, "ss"):
		return strings.TrimSuffix(words, "s")
	default:
		return words
	}
}

func containsWords(text, words string) bool {
	for offset := 0; offset < len(text); {
		idx := strings.Index(text[offset:], words)
		if idx < 0 {
			return false
		}
		start := offset + idx
		end := start + len(words)
		if boundary(text, start-1) && boundary(text, end) {
			return true
		}
		offset = start + 1
	}

	return false
}

func boundary(text string, idx int) bool {
	if idx < 0 || idx >= len(text) {
		return true
	}
	r := rune(text[idx])

	return !unicode.IsLetter(r) && !unicode.IsDigit(r)
}
