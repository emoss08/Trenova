package replycheck

import (
	"regexp"
	"slices"
	"strings"
	"unicode"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/domain/codemode"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/inflect"
	"github.com/emoss08/trenova/shared/numberguard"
	"github.com/emoss08/trenova/shared/textcase"
	"github.com/shopspring/decimal"
)

const (
	groundingCorrectionLead = "Your reply does not match what this turn filed."
	groundingRetryReason    = "The reply named something the filed change does not hold, so it " +
		"is being written again."
	GroundingNote = "\n\n_Check the card before relying on this reply: parts of it may " +
		"describe more than the proposal holds._"
	groundingNoteReason = "The rewritten reply still named something the filed change does " +
		"not hold, so it ends by pointing to the card."

	// UnseenCorrection ends every grounding correction. Told only what was
	// wrong, a model answered the correction ("You're right, the 99% was my own
	// arithmetic") and the person read an apology for a reply they never saw.
	UnseenCorrection = " The person sees neither this message nor the reply it replaces: " +
		"write your whole answer to their question again, as if for the first time, and " +
		"never mention a correction, an earlier reply or how a figure was checked."

	AmountsCorrectionLead = "Your reply states figures nothing this conversation read:"
	amountsRetryReason    = "The reply stated a figure nothing this conversation read, so it " +
		"is being written again."
	amountsNoteReason = "The rewritten reply still stated a figure nothing this conversation " +
		"read, so it ends by naming the figures to check."
	minLabelWords = 2

	minReadValueRunes = 3
	maxReadValueWords = 8
	readValueMarks    = "*_`\"'()“”‘’"
	readValueClauses  = ",;:"
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
	changeSpeak = []string{
		"proposal", "proposals", "propose", "proposed", "proposing", "file", "filed",
		"filing", "would", "will", "i'll", "it'll", "we'll", "going to", "approve",
		"approved", "approval", "approving", "change", "changes", "changed", "update",
		"updates", "updated", "set", "sets",
	}
	leadingArticles = []string{"the", "a", "an"}
	negationWords   = []string{
		"no", "not", "none", "never", "without", "nor", "omit", "omitted", "omitting",
		"excluded", "exclude", "excluding", "skip", "skipped", "missing", "left",
		"cannot", "unable",
	}
)

type GroundingState struct {
	Rewritten bool
}

type GroundingFinding struct {
	figures []string
	fields  []string
	tools   map[string][]string
	nested  []string
	// offered is the fields the reply names only in an offer to add them.
	offered []string
	// read is a finding on a reply that filed nothing, held only to the
	// amounts the conversation read; analyze is whether the agent can work a
	// missing one out with analysis_run.
	read    bool
	analyze bool
}

func (f GroundingFinding) empty() bool {
	return len(f.figures) == 0 && len(f.fields) == 0
}

func (f *GroundingFinding) dropNestedWords() {
	f.dropFields(f.nested)
	f.nested = nil
}

func (f *GroundingFinding) dropOffered() {
	f.dropFields(f.offered)
	f.offered = nil
}

func (f *GroundingFinding) dropFields(words []string) {
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

// Drift is what the reply says that the turn does not hold. A turn that filed
// writes is held to them; one that filed nothing is held to the amounts the
// conversation read, the totals and shares a model works out in its head
// (DB-186).
func Drift(t Turn, specFor SpecFor, reply string) (GroundingFinding, bool) {
	if len(t.Result().Actions) == 0 {
		finding := readFinding(t, reply)

		return finding, !finding.empty()
	}

	finding := groundingFindings(t, specFor, reply)
	finding.dropNestedWords()
	finding.dropOffered()

	return finding, !finding.empty()
}

// readFinding is the money and shares a reply that filed nothing states and
// nothing the conversation read gives. A conversation that has read no tool's
// result is answering from what the trade knows, and is not checked.
func readFinding(t Turn, reply string) GroundingFinding {
	finding := GroundingFinding{read: true, analyze: t.Holds(codemode.ToolRunScript)}
	if strings.TrimSpace(reply) == "" || !readAnything(t) || readUnseenFiles(t) {
		return finding
	}
	separated := groundingSupportAs(t, numberguard.ReadSeparated)
	supported := append(slices.Clone(separated), statedArithmetic(t)...)
	candidates := numberguard.CheckAmountsAs(numberguard.ReadSeparated, reply, supported).
		Unsupported
	if len(candidates) > 0 {
		finding.figures = narrowed(candidates, numberguard.CheckAmounts(reply,
			withDerived(t, reply, slices.Concat(GroundingSupport(t), separated))).Unsupported)
	}

	return finding
}

// withWorking is what the turn read, and what the reply's own figures and the
// person's work out to in one step (numberguard.Worked): "112 gallons at $3.89
// come to $435.68" shows its working, and the guard checks the result. It is
// what a figure the reply says a filed change holds is held to.
func withWorking(t Turn, reply string, read []decimal.Decimal) []decimal.Decimal {
	return append(withStated(t, read), numberguard.Worked(reply, read)...)
}

// withDerived is what the turn read and the person's figures work out to, and
// what the reply derives from them in the steps it shows (numberguard.Derived):
// "$1,240 for the miles plus $150 for the tarp, $1,390 in all" (DB-244), "the
// $3,200 pays $1,562.50 and $1,637.50 of $2,531.25, leaving $893.75", and a
// list of six loads followed by their total. Each operand is a figure the
// reply states, so a share whose sides it leaves out still goes to a script
// (DB-186), and the work is bounded by the reply, not by what the conversation
// read: at most sixteen operands over two steps, and a total per list, table
// column and sentence.
func withDerived(t Turn, reply string, read []decimal.Decimal) []decimal.Decimal {
	supported := withStated(t, read)

	return append(supported, numberguard.Derived(reply, supported)...)
}

// withStated is what the turn read, and what the person's own figures work out
// to in one step.
func withStated(t Turn, read []decimal.Decimal) []decimal.Decimal {
	return append(slices.Clone(read),
		numberguard.Combinations(statedFigures(t, numberguard.ReadCodesApart))...)
}

// maxStatedFigures bounds the figures of the person's question that are
// paired up, so a pasted table does not yield thousands of sums.
const maxStatedFigures = 12

// statedArithmetic is the sums and differences of the figures in the person's
// own question. Told a check was 3,200 against a 3,700 invoice, "$500 short"
// restates what they said; it is not a figure the agent needs a script for.
func statedArithmetic(t Turn) []decimal.Decimal {
	stated := statedFigures(t, numberguard.ReadSeparated)
	pairs := make([]decimal.Decimal, 0, len(stated)*(len(stated)-1))
	for i := range stated {
		for j := i + 1; j < len(stated); j++ {
			pairs = append(pairs, stated[i].Add(stated[j]), stated[i].Sub(stated[j]).Abs())
		}
	}

	return pairs
}

// statedFigures is the figures of the person's question, the first
// maxStatedFigures of them.
func statedFigures(t Turn, how numberguard.Reading) []decimal.Decimal {
	if t.Request() == nil {
		return nil
	}
	stated := numberguard.SupportedFromTextAs(how, t.Request().Input)
	if len(stated) > maxStatedFigures {
		stated = stated[:maxStatedFigures]
	}

	return stated
}

// readAnything reports whether the conversation has read a tool's result, in
// this turn or a replayed one.
func readAnything(t Turn) bool {
	if len(t.Evidence()) > 0 {
		return true
	}

	return slices.ContainsFunc(t.Messages(), func(message llmports.Message) bool {
		return message.Role == llmports.RoleTool
	})
}

// readUnseenFiles reports whether the model was handed a file the guard
// cannot read: a PDF or an image no text was taken from, which the model reads
// itself. A figure from a scanned rate confirmation is not invented because
// the guard never saw the page.
func readUnseenFiles(t Turn) bool {
	return slices.ContainsFunc(t.Messages(), func(message llmports.Message) bool {
		return len(message.Files) > 0
	})
}

// filedFigures is the figures a reply on a filed change cites that nothing
// read or filed gives. A figure from a file only the model can read is not
// held against the reply. A figure the reply states in a sentence about the
// change (changeSpeak: "the proposal bills $1,390") is held to what was read
// and filed and one step of the reply's working; one it states about what the
// conversation read may be derived as a reply that filed nothing may.
func filedFigures(t Turn, reply string) []string {
	if readUnseenFiles(t) {
		return nil
	}
	separated := groundingSupportAs(t, numberguard.ReadSeparated)
	figures := numberguard.CheckNumbersAs(numberguard.ReadSeparated, reply, separated).Unsupported
	if len(figures) == 0 {
		return figures
	}

	read := slices.Concat(GroundingSupport(t), separated)
	figures = narrowed(figures, numberguard.CheckNumbers(reply, withWorking(t, reply, read)).
		Unsupported)
	if len(figures) == 0 {
		return figures
	}
	derived := numberguard.CheckNumbers(reply, withDerived(t, reply, read)).Unsupported
	claimed := numberguard.CheckNumbers(changeSentences(reply), nil).Unsupported

	return slices.DeleteFunc(figures, func(figure string) bool {
		return !slices.Contains(derived, figure) && !slices.Contains(claimed, figure)
	})
}

// changeSentences is the reply's sentences that speak of the change, one to a
// line.
func changeSentences(reply string) string {
	sentences := numberguard.Sentences(reply)
	claims := make([]string, 0, len(sentences))
	for _, sentence := range sentences {
		if speaksOfChange(strings.ToLower(sentence)) {
			claims = append(claims, sentence)
		}
	}

	return strings.Join(claims, "\n")
}

func narrowed(candidates, unsupported []string) []string {
	return slices.DeleteFunc(slices.Clone(unsupported), func(figure string) bool {
		return !slices.Contains(candidates, figure)
	})
}

func isGroundingCorrection(content string) bool {
	return strings.HasPrefix(content, groundingCorrectionLead) ||
		strings.HasPrefix(content, AmountsCorrectionLead)
}

func (f GroundingFinding) RetryReason() string {
	if f.read {
		return amountsRetryReason
	}

	return groundingRetryReason
}

func (f GroundingFinding) NoteReason() string {
	if f.read {
		return amountsNoteReason
	}

	return groundingNoteReason
}

func (f GroundingFinding) Note() string {
	if !f.read {
		return GroundingNote
	}

	return "\n\n_Check the figures in this reply before relying on them: " +
		strings.Join(f.figures, ", ") +
		" came from no record, report or analysis this conversation read._"
}

func RegroundedEvent(
	action assistantports.RegroundAction,
	finding GroundingFinding,
	reason string,
) kernelports.StreamEvent {
	return kernelports.StreamEvent{
		Event: assistantports.AssistantEventReplyRegrounded,
		Data: assistantports.AssistantReplyRegroundedEvent{
			Action:  action,
			Figures: finding.figures,
			Fields:  finding.fields,
			Reason:  reason,
		},
	}
}

func (f GroundingFinding) Correction() string {
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
	b.WriteString(UnseenCorrection)

	return b.String()
}

func (f GroundingFinding) amountsCorrection() string {
	var b strings.Builder
	b.WriteString(AmountsCorrectionLead)
	b.WriteString(" ")
	b.WriteString(strings.Join(f.figures, ", "))
	b.WriteString(". A total, share, average or difference no tool returned is one you " +
		"worked out yourself, and a reply never states one as if a record held it.")
	if f.analyze {
		b.WriteString(" Work each one out with analysis_run over the rows it comes from, " +
			"then write the reply from its result.")
	} else {
		b.WriteString(" Write the reply again from what the tools returned. Where they do " +
			"not give a figure, leave it out and say what they do show.")
	}
	b.WriteString(UnseenCorrection)

	return b.String()
}

func groundingFindings(t Turn, specFor SpecFor, reply string) GroundingFinding {
	finding := GroundingFinding{tools: make(map[string][]string, 2)}
	if strings.TrimSpace(reply) == "" {
		return finding
	}

	finding.figures = filedFigures(t, reply)

	claims := claimedPhrases(reply)
	stated := claims.withoutOffers()
	reads := readsOf(t)
	for _, tool := range filedTools(t.Result().Actions) {
		spec, ok := specFor(tool)
		if !ok {
			continue
		}
		filed := filedFor(t, tool)
		for _, label := range schemaLabels(spec.Parameters) {
			if !claims.names(label.words) || filed.holds(label) ||
				stated.readFrom(label.words, reads) {
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

// GroundingSupport is every figure the turn has read: what it was sent, and
// the figures of older results the replay shortened.
func GroundingSupport(t Turn) []decimal.Decimal {
	return groundingSupportAs(t, numberguard.ReadCodesApart)
}

// groundingSupportAs is groundingSupport in the given reading.
func groundingSupportAs(t Turn, how numberguard.Reading) []decimal.Decimal {
	return withEvidence(t, numberguard.SupportedFromTextAs(how, GroundingTexts(t)...))
}

func withEvidence(t Turn, supported []decimal.Decimal) []decimal.Decimal {
	for _, figure := range t.Evidence() {
		if value, err := decimal.NewFromString(figure); err == nil {
			supported = append(supported, value)
		}
	}

	return supported
}

// FiguresOf is the figures in texts, as the guard reads them.
func FiguresOf(texts []string) []string {
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

func GroundingTexts(t Turn) []string {
	texts := make([]string, 0, len(t.Messages())+len(t.Result().Actions)+1)
	texts = append(texts, t.System())
	for idx := range t.Messages() {
		message := &t.Messages()[idx]
		if message.Role == llmports.RoleAssistant || isGroundingCorrection(message.Content) {
			continue
		}
		texts = append(texts, message.Content)
	}
	for idx := range t.Result().Actions {
		if encoded, err := sonic.MarshalString(t.Result().Actions[idx].Arguments); err == nil {
			texts = append(texts, encoded)
		}
	}

	return texts
}

func filedTools(actions []agentruntimeports.PendingAction) []string {
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

type readResults []string

func readsOf(t Turn) readResults {
	filed := make(map[string]bool, len(t.Result().Actions))
	for idx := range t.Result().Actions {
		filed[t.Result().Actions[idx].ToolCallID] = true
	}
	reads := make(readResults, 0, 4)
	for idx := range t.Messages() {
		message := &t.Messages()[idx]
		if message.Role != llmports.RoleTool || message.IsError || filed[message.ToolCallID] {
			continue
		}
		reads = append(reads, strings.ToLower(message.Content))
	}

	return reads
}

func (r readResults) hold(after string) bool {
	tokens := strings.Fields(after)
	for len(tokens) > 0 &&
		slices.Contains(leadingArticles, strings.Trim(tokens[0], readValueMarks)) {
		tokens = tokens[1:]
	}
	value := make([]string, 0, maxReadValueWords)
	for _, token := range tokens {
		bare := strings.TrimRight(token, readValueMarks)
		word := strings.Trim(strings.TrimRight(bare, readValueClauses), readValueMarks)
		if word != "" {
			value = append(value, word)
		}
		ends := bare != "" && strings.IndexByte(readValueClauses, bare[len(bare)-1]) >= 0
		if ends || len(value) == maxReadValueWords {
			break
		}
	}
	for n := len(value); n > 0; n-- {
		phrase := strings.Join(value[:n], " ")
		if valueRunes(phrase) < minReadValueRunes {
			return false
		}
		if slices.ContainsFunc(r, func(read string) bool { return containsWords(read, phrase) }) {
			return true
		}
	}

	return false
}

func valueRunes(text string) int {
	count := 0
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			count++
		}
	}

	return count
}

type filedCalls struct {
	arguments []map[string]any
	results   []string
}

func filedFor(t Turn, tool string) filedCalls {
	var filed filedCalls
	callIDs := make([]string, 0, 2)
	for idx := range t.Result().Actions {
		action := &t.Result().Actions[idx]
		if action.ToolName != tool || action.Withdrawn {
			continue
		}
		filed.arguments = append(filed.arguments, action.Arguments)
		callIDs = append(callIDs, action.ToolCallID)
	}
	for idx := range t.Result().Messages {
		message := &t.Result().Messages[idx]
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
			strings.ReplaceAll(textcase.ConvertCamelToSnake(name), "_", " "),
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

func (c replyClaims) readFrom(words string, reads readResults) bool {
	if len(reads) == 0 {
		return false
	}
	singular := inflect.SingularNoun(words)
	named := false
	for _, sentence := range c {
		at, size := indexWords(sentence, words), len(words)
		if at < 0 && singular != words {
			at, size = indexWords(sentence, singular), len(singular)
		}
		if at < 0 {
			continue
		}
		named = true
		if speaksOfChange(sentence) || !reads.hold(sentence[at+size:]) {
			return false
		}
	}

	return named
}

func speaksOfChange(sentence string) bool {
	return slices.ContainsFunc(changeSpeak, func(words string) bool {
		return containsWords(sentence, words)
	})
}

func (c replyClaims) names(words string) bool {
	singular := inflect.SingularNoun(words)
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
	return indexWords(text, words) >= 0
}

func indexWords(text, words string) int {
	for offset := 0; offset < len(text); {
		idx := strings.Index(text[offset:], words)
		if idx < 0 {
			return -1
		}
		start := offset + idx
		if boundary(text, start-1) && boundary(text, start+len(words)) {
			return start
		}
		offset = start + 1
	}

	return -1
}

func boundary(text string, idx int) bool {
	if idx < 0 || idx >= len(text) {
		return true
	}
	r := rune(text[idx])

	return !unicode.IsLetter(r) && !unicode.IsDigit(r)
}
