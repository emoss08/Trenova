package safetydoc

import (
	"bytes"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolpolicy"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	command = "go generate ./internal/core/services/agenttoolpolicy/safetydoc/..."
	none    = "—"
)

type classInfo struct {
	title   string
	meaning string
}

var classes = map[agent.EgressClass]classInfo{
	agent.EgressNone: {
		title:   "Reads only",
		meaning: "Looks something up. Nothing changes and nothing is sent.",
	},
	agent.EgressPersonal: {
		title:   "The caller's own records",
		meaning: "Changes only the records of the person using the agent.",
	},
	agent.EgressInternal: {
		title:   "Inside the organization",
		meaning: "Changes records only people inside the organization see.",
	},
	agent.EgressCustomerVisible: {
		title:   "Seen by a customer",
		meaning: "Changes something a customer can see.",
	},
	agent.EgressDriverVisible: {
		title:   "Seen by a driver",
		meaning: "Changes something a driver can see.",
	},
	agent.EgressExternalRecipient: {
		title:   "Sent outside the organization",
		meaning: "Sends to someone outside the organization.",
	},
	agent.EgressMoney: {
		title:   "Money",
		meaning: "Moves or commits money.",
	},
}

func Render(policies []serviceports.ToolPolicy) []byte {
	sorted := slices.Clone(policies)
	slices.SortStableFunc(sorted, func(a, b serviceports.ToolPolicy) int {
		return strings.Compare(a.Name, b.Name)
	})

	groups := make(map[agent.EgressClass][]serviceports.ToolPolicy, len(classes))
	for idx := range sorted {
		class := sorted[idx].HighestEgress()
		groups[class] = append(groups[class], sorted[idx])
	}

	var buf bytes.Buffer
	writePreface(&buf, len(sorted))
	writeClassTable(&buf, sorted)
	for _, class := range agent.EgressClasses() {
		writeGroup(&buf, class, groups[class])
	}

	return buf.Bytes()
}

func writePreface(buf *bytes.Buffer, total int) {
	fmt.Fprintf(buf, `# AI tool safety

<!-- Generated from the tool policies in code by running, in services/tms:
     %s
     Do not edit by hand. -->

What every tool an agent can call may do without a person, read from the
policies the tools declare in code. The runtime decides each call from the same
policies, so this page cannot drift from what runs: CI regenerates it and fails
when it differs. Each tool is listed once, under the furthest class its work
can reach.

Tools listed: %d.

## The model

**Classes.** Every tool says where its work can be seen or felt: nowhere (it
only reads), the caller's own records, inside the organization, a customer, a
driver, someone outside the organization, or money. A tool whose calls differ
classifies each call, and the class of that call decides.

**Tiers.** An agent runs a tool at one of three tiers: *Propose* (a person
decides and nothing runs until then), *Ask first* (it runs once a person
approves) or *Automatic* (it runs without asking). The tier a call gets is the
lowest of the tier the agent sets for the tool, the agent's ceiling, the tool's
max tier, its class's ceiling and any condition the call's record must meet.
Work a customer or a driver can see, or that is sent outside the organization,
never runs past Ask first. Earned autonomy moves a tool up one tier after a
streak of clean approvals, and never past those limits.

**Taint.** Some tools return text written outside the organization: an inbound
message, a document, an EDI transaction, a bank receipt, a weather alert, a
comment a driver or a trading partner left on a shipment. Once a run has read
such text, every call that would leave the organization or move money waits
for approval, whatever its tier, and so does any call a tool's own taint hold
names, so an instruction hidden in that text cannot act on its own. Every call
taint held names it among what held it, whatever else held it too.

**Personal exemption.** A call that changes only the caller's own records runs
without a decision while that person is in the conversation, unless a person
set the tool's tier on the agent. An unattended run never has it.

`, command, total)
}

func writeClassTable(buf *bytes.Buffer, policies []serviceports.ToolPolicy) {
	buf.WriteString("## Classes\n\n")
	buf.WriteString("| Class | Means | Runs at most | Held once tainted | Tools that reach it |\n")
	buf.WriteString("| --- | --- | --- | --- | --- |\n")
	for _, class := range agent.EgressClasses() {
		info := classes[class]
		fmt.Fprintf(buf, "| %s | %s | %s | %s | %d |\n",
			info.title,
			info.meaning,
			class.Ceiling().Label(),
			yesNo(agenttoolpolicy.HeldWhenTainted(class)),
			reaching(policies, class),
		)
	}
	buf.WriteString("\n")
}

func writeGroup(buf *bytes.Buffer, class agent.EgressClass, policies []serviceports.ToolPolicy) {
	if len(policies) == 0 {
		return
	}

	info := classes[class]
	fmt.Fprintf(buf, "## %s\n\n%s\n\n", info.title, info.meaning)
	buf.WriteString("| Tool | Classes | Max tier | Condition | Reads outside text | Rationale |\n")
	buf.WriteString("| --- | --- | --- | --- | --- | --- |\n")
	for idx := range policies {
		policy := &policies[idx]
		fmt.Fprintf(buf, "| %s | %s | %s | %s | %s | %s |\n",
			cell(fmt.Sprintf("%s (`%s`)",
				stringutils.HumanizeCamelCaseSentence(policy.Name), policy.Name)),
			cell(classList(policy.Egress)),
			agenttoolpolicy.Promotable(*policy).Label(),
			cell(condition(policy)),
			cell(externalRead(policy)),
			cell(policy.Rationale),
		)
	}
	buf.WriteString("\n")
}

func reaching(policies []serviceports.ToolPolicy, class agent.EgressClass) int {
	count := 0
	for idx := range policies {
		if policies[idx].HasEgress(class) {
			count++
		}
	}

	return count
}

func classList(egress []agent.EgressClass) string {
	names := make([]string, 0, len(egress))
	for _, class := range egress {
		names = append(names, strings.ToLower(classes[class].title))
	}
	if len(names) == 0 {
		return none
	}

	return stringutils.CapitalizeFirst(strings.Join(names, ", "))
}

func condition(policy *serviceports.ToolPolicy) string {
	parts := make([]string, 0, 4)
	if policy.Classify != nil {
		parts = append(parts, "Each call is classified by what it reaches.")
	}
	if policy.Condition != nil {
		parts = append(parts, policy.Condition.Description)
	}
	if policy.TaintHold != nil {
		parts = append(parts, policy.TaintHold.Description)
	}
	if policy.PersonalRunsUnasked {
		parts = append(parts, "A call on the caller's own records runs unasked while they "+
			"are present.")
	}
	if len(parts) == 0 {
		return none
	}

	return strings.Join(parts, " ")
}

func externalRead(policy *serviceports.ToolPolicy) string {
	var read string
	switch policy.ReadsExternal {
	case agent.ExternalReadAlways:
		read = "Always, from " + sourceLabel(policy.Source)
	case agent.ExternalReadMarked:
		read = "When the record is marked, from " + sourceLabel(policy.Source)
	default:
		read = ""
	}
	if policy.CarriesTaint {
		if read != "" {
			read += ". "
		}
		read += "Carries outside text into later runs"
	}
	if read == "" {
		return none
	}

	return read
}

func sourceLabel(source agent.TaintSource) string {
	if source == agent.TaintSourceEDI {
		return "EDI"
	}

	return stringutils.HumanizeSnakeCase(source.String())
}

func cell(text string) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(text, "|", `\|`)), " ")
}

func yesNo(value bool) string {
	if value {
		return "Yes"
	}

	return "No"
}
