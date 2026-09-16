// Package agentguard confines what an organization's agents and chat sessions may
// be used for.
//
// The confinement that actually holds is not in this package: an agent can only
// change the world through the tool registry, so a capability that has no tool —
// running code, opening a shell, calling an arbitrary URL — does not exist no
// matter what the model is asked or told. This package handles the softer half,
// keeping the assistant on topic and refusing to act as a general-purpose model,
// which is a product and cost boundary rather than a security boundary.
//
// It is layered on purpose. The deterministic pass is cheap and precise but
// deliberately narrow; the classifier pass is broader but costs a model call; the
// system prompt is advisory and assumed bypassable. Treating any single layer as
// sufficient would be a mistake.
package agentguard

// Stage records which layer decided a request's fate, so a refusal can be
// explained and a misfiring layer can be found.
type Stage string

const (
	StageDeterministic = Stage("Deterministic")
	StageClassifier    = Stage("Classifier")
	StageAllowed       = Stage("Allowed")
	StageOutput        = Stage("Output")
	StageUnavailable   = Stage("Unavailable")
)

// Reason names why a request was refused.
type Reason string

const (
	// ReasonCodeGeneration covers asking the assistant to write, debug, or explain
	// software. Trenova's assistant is not a coding assistant.
	ReasonCodeGeneration = Reason("CodeGeneration")
	// ReasonOffDomain covers requests unrelated to transportation or to operating
	// this system.
	ReasonOffDomain = Reason("OffDomain")
	// ReasonPromptManipulation covers attempts to override the assistant's
	// instructions or extract them.
	ReasonPromptManipulation = Reason("PromptManipulation")
	// ReasonOversized covers input too large to classify safely.
	ReasonOversized = Reason("Oversized")
	// ReasonClassifierUnavailable covers a configured classifier that could not be
	// reached, which fails closed.
	ReasonClassifierUnavailable = Reason("ClassifierUnavailable")
)

// Category is what a request was understood to be about. The in-scope categories
// define the assistant's remit.
type Category string

const (
	CategoryTransportationOperations = Category("TransportationOperations")
	CategorySystemAutomation         = Category("SystemAutomation")
	CategorySystemUsage              = Category("SystemUsage")
	CategoryTransportationKnowledge  = Category("TransportationKnowledge")
	CategoryCodeGeneration           = Category("CodeGeneration")
	CategoryGeneralKnowledge         = Category("GeneralKnowledge")
	CategoryPromptManipulation       = Category("PromptManipulation")
	CategoryOther                    = Category("Other")
)

// InScope reports whether a category is one the assistant serves.
func (c Category) InScope() bool {
	switch c {
	case CategoryTransportationOperations,
		CategorySystemAutomation,
		CategorySystemUsage,
		CategoryTransportationKnowledge:
		return true
	default:
		return false
	}
}

// Decision is the outcome of evaluating one request.
type Decision struct {
	Allowed  bool     `json:"allowed"`
	Stage    Stage    `json:"stage"`
	Reason   Reason   `json:"reason,omitempty"`
	Category Category `json:"category,omitempty"`
	// Message is shown to the person who asked. It explains the boundary without
	// implying the assistant is broken.
	Message string `json:"message,omitempty"`
	// MatchedRule names the deterministic rule that fired, for debugging a false
	// positive without re-running the request.
	MatchedRule string `json:"matchedRule,omitempty"`
}

func allowed(stage Stage, category Category) Decision {
	return Decision{Allowed: true, Stage: stage, Category: category}
}

func refused(stage Stage, reason Reason, category Category, rule string) Decision {
	return Decision{
		Allowed:     false,
		Stage:       stage,
		Reason:      reason,
		Category:    category,
		Message:     refusalMessage(reason),
		MatchedRule: rule,
	}
}

// refusalMessage states the boundary plainly and points somewhere useful, rather
// than leaving the person guessing what the assistant is for.
func refusalMessage(reason Reason) string {
	switch reason {
	case ReasonCodeGeneration:
		return "This assistant handles transportation operations and automation inside Trenova, " +
			"not software development. Ask about shipments, dispatch, billing, or how to get " +
			"something done in the system."
	case ReasonOffDomain:
		return "This assistant only covers transportation work and operating Trenova. " +
			"Ask about shipments, workers, equipment, customers, billing, or how to " +
			"automate something here."
	case ReasonPromptManipulation:
		return "This assistant's instructions are fixed and cannot be changed from a message. " +
			"Ask about transportation work or how to get something done in Trenova."
	case ReasonOversized:
		return "That message is too long to process. Send a shorter request, or attach the " +
			"details as a document."
	case ReasonClassifierUnavailable:
		return "The assistant cannot verify this request right now, so it has not been sent. " +
			"Try again shortly, or contact an administrator if this persists."
	default:
		return "This assistant cannot help with that request."
	}
}
