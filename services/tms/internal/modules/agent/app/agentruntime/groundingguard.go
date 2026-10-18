package agentruntime

import (
	"regexp"
	"slices"
	"strings"
	"unicode"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/numberguard"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/shopspring/decimal"
)

const (
	changeGroundingGuard         = "agent-loop-grounding-guard"
	changeGroundingTopLevelWords = "agent-loop-grounding-top-level-words"
	changeGroundingReadAmounts   = "agent-loop-grounding-read-amounts"
	changeGroundingFiguresAsRead = "agent-loop-grounding-figures-as-read"
	changeGroundingOffers        = "agent-loop-grounding-offers-not-claims"
)

const (
	groundingCorrectionLead = "Your reply does not match what this turn filed."
	groundingRetryReason    = "The reply named something the filed change does not hold, so it " +
		"is being written again."
	groundingNote = "\n\n_Check the card before relying on this reply: parts of it may " +
		"describe more than the proposal holds._"
	groundingNoteReason = "The rewritten reply still named something the filed change does " +
		"not hold, so it ends by pointing to the card."

	// unseenCorrection ends every grounding correction. Told only what was
	// wrong, a model answered the correction ("You're right, the 99% was my own
	// arithmetic") and the person read an apology for a reply they never saw.
	unseenCorrection = " The person sees neither this message nor the reply it replaces: " +
		"write your whole answer to their question again, as if for the first time, and " +
		"never mention a correction, an earlier reply or how a figure was checked."

	amountsCorrectionLead = "Your reply states figures nothing this conversation read:"
	amountsRetryReason    = "The reply stated a figure nothing this conversation read, so it " +
		"is being written again."
	amountsNoteReason = "The rewritten reply still stated a figure nothing this conversation " +
		"read, so it ends by naming the figures to check."
	minLabelWords = 2
)

var (
	schemaPropertyPath = regexp.MustCompile(`^[A-Za-z0-9_]+(\[\])?(\.[A-Za-z0-9_]+(\[\])?)*$`)
	// offerPhrases mark a sentence that offers to do something rather than
	// saying what the change holds: "If you have the receipt, I can add the
	// vendor and transaction reference before approval."
	offerPhrases = []string{
		"i can", "i could", "i'd be happy to", "happy to", "want me to", "would you like",
		"shall i", "should i", "let me know",
	}
	negationWords = []string{
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
	nested  []string
	// offered is the fields the reply names only in an offer to add them.
	offered []string
	// read is a finding on a reply that filed nothing, held only to the
	// amounts the conversation read; analyze is whether the agent can work a
	// missing one out with analyze_data.
	read    bool
	analyze bool
}

func (f groundingFinding) empty() bool {
	return len(f.figures) == 0 && len(f.fields) == 0
}

func (f *groundingFinding) dropNestedWords() {
	f.dropFields(f.nested)
	f.nested = nil
}

func (f *groundingFinding) dropOffered() {
	f.dropFields(f.offered)
	f.offered = nil
}

func (f *groundingFinding) dropFields(words []string) {
	if len(words) == 0 {
		return
	}
	f.fields = slices.DeleteFunc(f.fields, func(field string) bool {
		return slices.Contains(words, field)
	})
	for _, field := range words {
		delete(f.tools, field)
	}
}

type schemaLabel struct {
	path       string
	words      string
	nestedWord bool
}

func (s *Service) reground(
	t *Turn,
	fx TurnEffects,
	completion *agentports.ChatCompletionResult,
	state *groundingState,
) bool {
	if completion == nil {
		return false
	}

	finding, found := s.drift(t, fx, completion.Text)
	if !found {
		return false
	}

	if !state.rewritten {
		state.rewritten = true
		reason := finding.retryReason()
		fx.Emit(kernelports.StreamEvent{
			Event: agentports.AssistantEventRetrying,
			Data: agentports.AssistantRetryingEvent{
				Attempt: 1,
				Reason:  reason,
				Kind:    agentports.RetryKindRestart,
			},
		})
		fx.Emit(regroundedEvent(agentports.RegroundRewrite, finding, reason))
		t.messages = append(t.messages,
			agentports.Message{
				Role:          agentports.RoleAssistant,
				Content:       completion.Text,
				ReplyID:       completion.ProviderMessageID,
				ReplyProvider: completion.ProviderID,
			},
			agentports.Message{Role: agentports.RoleUser, Content: finding.correction()},
		)

		return true
	}

	note := finding.note()
	fx.Emit(regroundedEvent(agentports.RegroundNote, finding, finding.noteReason()))
	completion.Text += note
	fx.Emit(deltaEvent(note))

	return false
}

// drift is what the reply says that the turn does not hold. A turn that filed
// writes is held to them; one that filed nothing is held to the amounts the
// conversation read, the totals and shares a model works out in its head
// (DB-186). Each check is asked of the workflow only when it found something.
func (s *Service) drift(t *Turn, fx TurnEffects, reply string) (groundingFinding, bool) {
	if len(t.result.Actions) == 0 {
		finding := readFinding(t, reply)

		return finding, !finding.empty() && fx.Supports(changeGroundingReadAmounts)
	}

	finding := s.groundingFindings(t, fx, reply)
	if finding.empty() || !fx.Supports(changeGroundingGuard) {
		return finding, false
	}
	if len(finding.nested) > 0 && fx.Supports(changeGroundingTopLevelWords) {
		finding.dropNestedWords()
		if finding.empty() {
			return finding, false
		}
	}
	if len(finding.offered) > 0 && fx.Supports(changeGroundingOffers) {
		finding.dropOffered()
		if finding.empty() {
			return finding, false
		}
	}

	return finding, true
}

// readFinding is the money and shares a reply that filed nothing states and
// nothing the conversation read gives. A conversation that has read no tool's
// result is answering from what the trade knows, and is not checked.
func readFinding(t *Turn, reply string) groundingFinding {
	finding := groundingFinding{read: true, analyze: t.holds(agentextension.ToolAnalyzeData)}
	if strings.TrimSpace(reply) == "" || !t.readAnything() || t.readUnseenFiles() {
		return finding
	}
	supported := t.groundingSupport()
	supported = append(supported, t.statedArithmetic()...)
	finding.figures = numberguard.CheckAmounts(reply, supported).Unsupported

	return finding
}

// maxStatedFigures bounds the figures of the person's question that are
// paired up, so a pasted table does not yield thousands of sums.
const maxStatedFigures = 12

// statedArithmetic is the sums and differences of the figures in the person's
// own question. Told a check was 3,200 against a 3,700 invoice, "$500 short"
// restates what they said; it is not a figure the agent needs analyze_data for.
func (t *Turn) statedArithmetic() []decimal.Decimal {
	if t.req == nil {
		return nil
	}
	stated := numberguard.SupportedFromText(t.req.Input)
	if len(stated) > maxStatedFigures {
		stated = stated[:maxStatedFigures]
	}
	pairs := make([]decimal.Decimal, 0, len(stated)*(len(stated)-1))
	for i := range stated {
		for j := i + 1; j < len(stated); j++ {
			pairs = append(pairs, stated[i].Add(stated[j]), stated[i].Sub(stated[j]).Abs())
		}
	}

	return pairs
}

// readAnything reports whether the conversation has read a tool's result, in
// this turn or a replayed one.
func (t *Turn) readAnything() bool {
	if len(t.evidence) > 0 {
		return true
	}

	return slices.ContainsFunc(t.messages, func(message agentports.Message) bool {
		return message.Role == agentports.RoleTool
	})
}

// readUnseenFiles reports whether the model was handed a file the guard
// cannot read: a PDF or an image no text was taken from, which the model reads
// itself. A figure from a scanned rate confirmation is not invented because
// the guard never saw the page.
func (t *Turn) readUnseenFiles() bool {
	return slices.ContainsFunc(t.messages, func(message agentports.Message) bool {
		return len(message.Files) > 0
	})
}

// filedFigures is the figures a reply on a filed change cites that nothing
// read or filed gives. Before agent-loop-grounding-figures-as-read the guard
// read a JSON array of numbers as one figure that never parsed, a date's day
// and a code's digits as figures, and a figure from a file only the model can
// read as invented; an execution that decided that way keeps deciding that
// way where the two readings disagree on whether anything is wrong.
func (t *Turn) filedFigures(fx TurnEffects, reply string) []string {
	var figures []string
	if !t.readUnseenFiles() {
		figures = numberguard.CheckNumbers(reply, t.groundingSupport()).Unsupported
	}
	joined := numberguard.CheckNumbersJoined(reply, t.groundingSupportJoined()).Unsupported
	if (len(figures) == 0) == (len(joined) == 0) || fx.Supports(changeGroundingFiguresAsRead) {
		return figures
	}

	return joined
}

func isGroundingCorrection(content string) bool {
	return strings.HasPrefix(content, groundingCorrectionLead) ||
		strings.HasPrefix(content, amountsCorrectionLead)
}

func (f groundingFinding) retryReason() string {
	if f.read {
		return amountsRetryReason
	}

	return groundingRetryReason
}

func (f groundingFinding) noteReason() string {
	if f.read {
		return amountsNoteReason
	}

	return groundingNoteReason
}

func (f groundingFinding) note() string {
	if !f.read {
		return groundingNote
	}

	return "\n\n_Check the figures in this reply before relying on them: " +
		strings.Join(f.figures, ", ") +
		" came from no record, report or analysis this conversation read._"
}

func regroundedEvent(
	action agentports.RegroundAction,
	finding groundingFinding,
	reason string,
) kernelports.StreamEvent {
	return kernelports.StreamEvent{
		Event: agentports.AssistantEventReplyRegrounded,
		Data: agentports.AssistantReplyRegroundedEvent{
			Action:  action,
			Figures: finding.figures,
			Fields:  finding.fields,
			Reason:  reason,
		},
	}
}

func (f groundingFinding) correction() string {
	if f.read {
		return f.amountsCorrection()
	}

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
	b.WriteString(unseenCorrection)

	return b.String()
}

func (f groundingFinding) amountsCorrection() string {
	var b strings.Builder
	b.WriteString(amountsCorrectionLead)
	b.WriteString(" ")
	b.WriteString(strings.Join(f.figures, ", "))
	b.WriteString(". A total, share, average or difference no tool returned is one you " +
		"worked out yourself, and a reply never states one as if a record held it.")
	if f.analyze {
		b.WriteString(" Work each one out with analyze_data over the rows it comes from, " +
			"then write the reply from its result.")
	} else {
		b.WriteString(" Write the reply again from what the tools returned. Where they do " +
			"not give a figure, leave it out and say what they do show.")
	}
	b.WriteString(unseenCorrection)

	return b.String()
}

func (s *Service) groundingFindings(t *Turn, fx TurnEffects, reply string) groundingFinding {
	finding := groundingFinding{tools: make(map[string][]string, 2)}
	if strings.TrimSpace(reply) == "" {
		return finding
	}

	finding.figures = t.filedFigures(fx, reply)

	claims := claimedPhrases(reply)
	stated := claims.withoutOffers()
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
			switch {
			case !slices.Contains(finding.fields, label.words):
				finding.fields = append(finding.fields, label.words)
				if label.nestedWord {
					finding.nested = append(finding.nested, label.words)
				}
			case !label.nestedWord:
				finding.nested = slices.DeleteFunc(finding.nested, func(words string) bool {
					return words == label.words
				})
			}
			if !slices.Contains(finding.tools[label.words], tool) {
				finding.tools[label.words] = append(finding.tools[label.words], tool)
			}
			if !stated.names(label.words) && !slices.Contains(finding.offered, label.words) {
				finding.offered = append(finding.offered, label.words)
			}
		}
	}
	finding.offered = slices.DeleteFunc(finding.offered, func(words string) bool {
		return stated.names(words)
	})
	slices.Sort(finding.fields)

	return finding
}

// groundingSupport is every figure the turn has read: what it was sent, and
// the figures of older results the replay shortened.
func (t *Turn) groundingSupport() []decimal.Decimal {
	return t.withEvidence(numberguard.SupportedFromText(t.groundingTexts()...))
}

// groundingSupportJoined is groundingSupport as the guard read it before
// agent-loop-grounding-figures-as-read.
func (t *Turn) groundingSupportJoined() []decimal.Decimal {
	return t.withEvidence(numberguard.SupportedFromTextJoined(t.groundingTexts()...))
}

func (t *Turn) withEvidence(supported []decimal.Decimal) []decimal.Decimal {
	for _, figure := range t.evidence {
		if value, err := decimal.NewFromString(figure); err == nil {
			supported = append(supported, value)
		}
	}

	return supported
}

// figuresOf is the figures in texts, as the guard reads them.
func figuresOf(texts []string) []string {
	if len(texts) == 0 {
		return nil
	}
	values := numberguard.SupportedFromText(texts...)
	figures := make([]string, 0, len(values))
	for _, value := range values {
		figures = append(figures, value.String())
	}

	return figures
}

func (t *Turn) groundingTexts() []string {
	texts := make([]string, 0, len(t.messages)+len(t.result.Actions)+1)
	texts = append(texts, t.system)
	for idx := range t.messages {
		message := &t.messages[idx]
		if message.Role == agentports.RoleAssistant || isGroundingCorrection(message.Content) {
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

func filedTools(actions []agentports.PendingAction) []string {
	tools := make([]string, 0, len(actions))
	for idx := range actions {
		if actions[idx].Withdrawn {
			continue
		}
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
		if action.ToolName != tool || action.Withdrawn {
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
		words := strings.ToLower(
			strings.ReplaceAll(stringutils.ConvertCamelToSnake(name), "_", " "),
		)
		kind, _ := node[toolschema.KeyType].(string)
		collection := kind == toolschema.TypeArray || kind == toolschema.TypeObject
		oneWord := len(strings.Fields(words)) < minLabelWords
		if !collection && oneWord {
			return
		}
		labels = append(labels, schemaLabel{
			path:       path,
			words:      words,
			nestedWord: oneWord && strings.Contains(path, "."),
		})
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

// withoutOffers is the claims that say what the change holds, leaving out
// those that offer to do something for the person.
func (c replyClaims) withoutOffers() replyClaims {
	return slices.DeleteFunc(slices.Clone(c), func(sentence string) bool {
		return slices.ContainsFunc(offerPhrases, func(phrase string) bool {
			return containsWords(sentence, phrase)
		})
	})
}

func (c replyClaims) names(words string) bool {
	singular := stringutils.SingularNoun(words)
	for _, sentence := range c {
		if containsWords(sentence, words) ||
			(singular != words && containsWords(sentence, singular)) {
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
