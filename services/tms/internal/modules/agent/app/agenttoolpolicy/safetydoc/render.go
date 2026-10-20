package safetydoc

import (
	"bytes"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolpolicy"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	command = "go generate ./internal/modules/agent/app/agenttoolpolicy/safetydoc/..."
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
		title: "Money",
		meaning: "Moves or commits money. A call runs on its own only when its tool proves " +
			"it is within bounds.",
	},
}

func Render(policies []agentports.ToolPolicy) []byte {
	sorted := slices.Clone(policies)
	slices.SortStableFunc(sorted, func(a, b agentports.ToolPolicy) int {
		return strings.Compare(a.Name, b.Name)
	})

	groups := make(map[agent.EgressClass][]agentports.ToolPolicy, len(classes))
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

**Money.** A money write runs at most Ask first, so a new money tool asks a
person by default. It runs on its own only on a tool whose condition proves the
call is within bounds: the records it touches are in the state the tool
requires, and the amount it moves is within the organization's money limits for
agents in AI Control, which are a ceiling on one change and a cap on each run
and on each day in both the number of changes and the total moved. The limits
are amounts in the organization's functional currency, so a change in any other
currency waits for a person, and the caps count only changes in it. The day is
the organization's own and counts every agent, a delegate's changes count
against the run that handed it the task, and a change that failed moved nothing
and counts against nothing. A call outside any bound, or whose bounds, limits or
day cannot be read, is filed as a proposal that names the bound it exceeded. A
run that has read outside text never moves money on its own, with one
exception a tool declares: a bound that reads a record again from the
organization's own records, and takes the amount, the customer and the
references from it rather than from the call, may vouch for having read that
record. Only a bank receipt can be vouched for this way, by the tools that say
so below, and only for a receipt the run itself read and named: a receipt
handed across a delegation, one read without naming which, or reads past what
the run can keep track of hold the change, as does any other outside text the
run read.

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

**Escalation.** Raising an exception, which hands the run to a person for
review, runs whatever the agent's autonomy ceiling, the turn's taint or the run being
unattended, because it only records the run for a person inside the
organization. A run that is unsure has to be able to hand its work to someone
without first waiting for someone. A tier a person set for the tool on the
agent still holds.

**Unattended runs.** A run nobody is in is authorized as the agent, from the
fixed table of what any agent may do, and never from a role: it cannot approve,
and it cannot reach a person's own records. What it writes is attributed to the
instance's system user, so a record's created-by and updated-by, an automatic
write's executor and the audit log's user name that account rather than nobody,
and the audit log's description says which agent ran it ("Ran by Dispatch
Agent"); the account lends its name, not its permissions.

**Data access.** A read shows a field only when the reader's data access
reaches it. An unattended agent reads at its own data access setting, Internal
unless someone whose role reaches Restricted raises it; a run a person is in
reads at the lower of that setting and the person's own role. Amounts, pay,
memos and raw EDI above that tier are left out and named in withheldByAccess,
and Confidential fields never reach a model at all.

`, command, total)
}

func writeClassTable(buf *bytes.Buffer, policies []agentports.ToolPolicy) {
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

func writeGroup(buf *bytes.Buffer, class agent.EgressClass, policies []agentports.ToolPolicy) {
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
			stringutils.MarkdownTableCell(fmt.Sprintf(
				"%s (`%s`)",
				stringutils.HumanizeCamelCaseSentence(
					agentports.ReadingOrder(policy.Name),
				),
				policy.Name,
			)),
			stringutils.MarkdownTableCell(classList(policy.Egress)),
			agenttoolpolicy.Promotable(*policy).Label(),
			stringutils.MarkdownTableCell(condition(policy)),
			stringutils.MarkdownTableCell(externalRead(policy)),
			stringutils.MarkdownTableCell(policy.Rationale),
		)
	}
	buf.WriteString("\n")
}

func reaching(policies []agentports.ToolPolicy, class agent.EgressClass) int {
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

func condition(policy *agentports.ToolPolicy) string {
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
	if policy.Escalates {
		parts = append(parts, "Runs above the agent's ceiling: it only puts the run in front "+
			"of a person, so it waits for no one unless a person set its tier on the agent.")
	}
	if len(parts) == 0 {
		return none
	}

	return strings.Join(parts, " ")
}

func externalRead(policy *agentports.ToolPolicy) string {
	var read string
	switch policy.ReadsExternal {
	case agent.ExternalReadAlways:
		read = "Always, from " + sourceLabels(policy)
	case agent.ExternalReadMarked:
		read = "When the record is marked, from " + sourceLabels(policy)
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

func sourceLabels(policy *agentports.ToolPolicy) string {
	if len(policy.Sources) == 0 {
		return sourceLabel(policy.Source)
	}

	labels := make([]string, 0, len(policy.Sources)+1)
	labels = append(labels, sourceLabel(policy.Source))
	for _, source := range policy.Sources {
		labels = append(labels, sourceLabel(source))
	}
	last := len(labels) - 1

	return strings.Join(labels[:last], ", ") + " or " + labels[last]
}

func sourceLabel(source agent.TaintSource) string {
	if source == agent.TaintSourceEDI {
		return "EDI"
	}

	return stringutils.HumanizeSnakeCase(source.String())
}

func yesNo(value bool) string {
	if value {
		return "Yes"
	}

	return "No"
}
