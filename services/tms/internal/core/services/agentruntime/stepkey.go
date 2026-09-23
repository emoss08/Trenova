package agentruntime

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
)

// StepKeyParams names one tool call by what it does.
type StepKeyParams struct {
	// OwnerID is the run or turn the call belongs to, so two runs asking for
	// the same write are two writes.
	OwnerID  pulid.ID
	ToolName string
	Args     map[string]any
	// Ordinal separates a call from an identical one earlier in the same run.
	// It is almost always zero, because the repeat guard already refuses a
	// byte-identical call that failed; it exists so a tool legitimately asked
	// for twice — two reminders to the same person — is not collapsed into one.
	Ordinal int
	// Scope separates the steps of another agent's turn, on a task the
	// owner's agent handed it, from the owner's own. Empty for the owner's
	// own steps, whose keys are what they always were.
	Scope string
}

// StepKey identifies a tool call by what it does rather than by the id the
// provider happened to give it.
//
// The provider's call id is minted per completion. A run whose activity timed
// out and was retried asks the model again, gets fresh call ids for the same
// writes, and anything keyed on them sees two distinct operations where there
// was one decision. What does not change across a retry is what the model
// asked for, so that is what names the step.
//
// The hash runs over the same canonical encoding the repeat guard uses, and
// must: sonic's fast path emits map keys in whatever order it finds them, so
// two serialisations of one set of arguments would otherwise produce two keys
// and the ledger would protect nothing.
//
// An empty key means the arguments could not be encoded. The caller must treat
// that as a step it cannot guard rather than as a key.
func StepKey(p StepKeyParams) string {
	base, ok := callKey(serviceports.ToolCall{Name: p.ToolName, Arguments: p.Args})
	if !ok {
		return ""
	}

	material := p.OwnerID.String() + "\x00" + base + "\x00" + strconv.Itoa(p.Ordinal)
	if p.Scope != "" {
		material += "\x00" + p.Scope
	}
	sum := sha256.Sum256([]byte(material))

	return hex.EncodeToString(sum[:])
}
