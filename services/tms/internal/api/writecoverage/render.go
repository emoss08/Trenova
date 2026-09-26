package writecoverage

import (
	"bytes"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/shared/stringutils"
)

const none = "—"

type tally struct {
	total   int
	covered int
	pending int
	exempt  map[Category]int
}

func newTally() *tally {
	return &tally{exempt: make(map[Category]int, len(categories))}
}

func (t *tally) add(decision Decision) {
	t.total++
	switch decision.State() {
	case StateCovered:
		t.covered++
	case StateExempt:
		t.exempt[decision.Exempt]++
	case StatePending:
		t.pending++
	}
}

func (t *tally) exempted() int {
	count := 0
	for _, n := range t.exempt {
		count += n
	}

	return count
}

func Render(report *Report) []byte {
	writes := slices.Clone(report.Writes)
	sortWrites(writes)

	overall := newTally()
	domains := make([]string, 0)
	byDomain := make(map[string]*tally)
	used := make(map[string]struct{})
	mutations, routes, twins := 0, 0, 0

	for idx := range writes {
		write := &writes[idx]
		decision := report.Mapping.Writes[write.Key]
		overall.add(decision)
		if _, ok := byDomain[write.Domain]; !ok {
			byDomain[write.Domain] = newTally()
			domains = append(domains, write.Domain)
		}
		byDomain[write.Domain].add(decision)
		for _, name := range decision.Tools {
			used[name] = struct{}{}
		}
		if write.Kind == KindMutation {
			mutations++
		} else {
			routes++
		}
		twins += len(write.Twins)
	}

	var buf bytes.Buffer
	writePreface(&buf)
	writeCategories(&buf, overall)
	writeTotals(&buf, overall, counts{mutations: mutations, routes: routes, twins: twins})
	writePending(&buf, writes, report.Mapping)
	writeDomains(&buf, domains, byDomain)
	writeUnusedTools(&buf, report, used)
	writeEveryWrite(&buf, writes, report.Mapping)

	return buf.Bytes()
}

type counts struct {
	mutations int
	routes    int
	twins     int
}

func writePreface(buf *bytes.Buffer) {
	fmt.Fprintf(buf, `# Agent write coverage

<!-- Generated from the GraphQL schema, the REST route table, the registered agent
     tools and %s
     by running, in services/tms:
     %s
     Do not edit by hand. -->

Every write a person can make in Trenova should either have an agent tool that
performs it or a reasoned exemption. This page is that ledger: each GraphQL
mutation and each POST, PUT, PATCH or DELETE route, the tool that covers it or
the reason none should, and the writes still waiting for a tool.

## The rule

A new mutation or a new write route ships with a decision in
`+"`%s`"+`.
Each write is one entry under `+"`writes:`"+`, keyed as it appears below
(`+"`mutation createShipment`"+`, `+"`POST /api/v1/shipments/:shipmentID/cancel/`"+`),
and takes exactly one of:

- `+"`tools: [cancel_shipment]`"+`: the agent tools that perform it. A tool
  named here must be registered.
- `+"`exempt: <category>`"+` with `+"`reason: <why no agent should>`"+`: one of
  the categories below. The reason is required.
- `+"`pending: <what the tool would do>`"+`: no tool yet. Pending writes are the
  backlog; they are counted, never failed.

The generator refuses, and `+"`TestCoverageIsCurrent`"+` and the `+"`Agent write coverage`"+`
step of `+"`Codegen Checks`"+` fail, when a write has no entry, an entry names a
write the app no longer exposes, an entry names a tool that does not exist, an
exemption has no reason or an unknown category, or this page differs from what
the generator writes. Each message names the key and the file to edit. After
editing the file, run `+"`task generate-write-coverage`"+` (or the command above) and
commit this page; `+"`task generate-write-coverage-check`"+` runs the CI check.

## How writes are found

- **GraphQL.** Every field of the `+"`Mutation`"+` type in
  `+"`internal/api/graphql/schema/*.graphqls`"+`, parsed with gqlparser. The
  domain is the schema file.
- **REST.** The gin route table itself: `+"`api.RouteTable`"+` registers every
  handler with zero-valued dependencies and reads back what gin holds, so a
  route cannot be missed by a parser. Routes one handler function serves (a
  trailing-slash alias, a PUT and a PATCH on one method) are one write, keyed
  by its shortest route. The domain is the handler package.
- **Twins.** A route is merged into a mutation, and needs no entry of its own,
  when the two reach exactly the same set of service methods (reads such as
  `+"`Get`"+` and `+"`List`"+` are ignored), found by walking the handler and the
  resolver with go/ast through their helper methods. When several mutations
  match, the one named for the handler method wins (`+"`patch`"+` to
  `+"`patchTractor`"+`); when that still leaves several, both are listed.
- **Blind spots.** A handler registered as a closure (`+"`h.review(h.service.Approve)`"+`)
  or one that passes a service method as a value is not merged with its twin;
  a route whose registration depends on configuration would be missed if the
  zero-valued configuration turns it off; a write reached only by a background
  job, an EDI message or an inbound webhook is not a write a person makes and is
  not listed.

`, MappingDisplayPath, GenerateCommand, MappingDisplayPath)
}

func writeCategories(buf *bytes.Buffer, overall *tally) {
	buf.WriteString("## Exemption categories\n\n")
	buf.WriteString("| Category | Means | Writes |\n")
	buf.WriteString("| --- | --- | --- |\n")
	for _, info := range categories {
		fmt.Fprintf(
			buf,
			"| `%s` | %s | %d |\n",
			info.Category,
			stringutils.MarkdownTableCell(info.Meaning),
			overall.exempt[info.Category],
		)
	}
	buf.WriteString("\n")
}

func writeTotals(buf *bytes.Buffer, overall *tally, c counts) {
	buf.WriteString("## Totals\n\n")
	fmt.Fprintf(buf,
		"%d writes: %d GraphQL mutations and %d REST writes, after merging %d REST "+
			"routes into the mutation they duplicate.\n\n",
		overall.total, c.mutations, c.routes, c.twins)
	buf.WriteString("| Decision | Writes |\n")
	buf.WriteString("| --- | --- |\n")
	fmt.Fprintf(buf, "| Covered by a tool | %d |\n", overall.covered)
	fmt.Fprintf(buf, "| Exempt | %d |\n", overall.exempted())
	for _, info := range categories {
		if n := overall.exempt[info.Category]; n > 0 {
			fmt.Fprintf(buf, "| — %s | %d |\n", info.Title, n)
		}
	}
	fmt.Fprintf(buf, "| **Pending** | **%d** |\n", overall.pending)
	fmt.Fprintf(buf, "| Total | %d |\n\n", overall.total)

	needTool := overall.covered + overall.pending
	if needTool > 0 {
		fmt.Fprintf(buf,
			"Of the %d writes an agent should be able to make, %d have a tool (%d%%).\n\n",
			needTool, overall.covered, overall.covered*100/needTool)
	}
}

func writePending(buf *bytes.Buffer, writes []Write, mapping Mapping) {
	buf.WriteString("## Pending\n\n")
	buf.WriteString("The writes no tool performs yet, and what the tool would do.\n\n")
	buf.WriteString("| Domain | Write | What the tool would do |\n")
	buf.WriteString("| --- | --- | --- |\n")
	for idx := range writes {
		decision := mapping.Writes[writes[idx].Key]
		if decision.State() != StatePending {
			continue
		}
		fmt.Fprintf(
			buf,
			"| %s | %s | %s |\n",
			writes[idx].Domain,
			code(writes[idx].Key),
			stringutils.MarkdownTableCell(orNone(decision.Pending)),
		)
	}
	buf.WriteString("\n")
}

func writeDomains(buf *bytes.Buffer, domains []string, byDomain map[string]*tally) {
	buf.WriteString("## By domain\n\n")
	buf.WriteString("| Domain | Writes | Covered | Exempt | Pending |\n")
	buf.WriteString("| --- | --- | --- | --- | --- |\n")
	for _, domain := range domains {
		t := byDomain[domain]
		fmt.Fprintf(buf, "| %s | %d | %d | %d | %d |\n",
			domain, t.total, t.covered, t.exempted(), t.pending)
	}
	buf.WriteString("\n")
}

func writeUnusedTools(buf *bytes.Buffer, report *Report, used map[string]struct{}) {
	unused := make([]string, 0)
	for _, name := range report.ToolOrder {
		if report.Tools[name].Kind != agent.ToolKindAction {
			continue
		}
		if _, ok := used[name]; !ok {
			unused = append(unused, name)
		}
	}
	slices.Sort(unused)

	buf.WriteString("## Action tools no write maps to\n\n")
	buf.WriteString("Tools that change something no person-facing write does, such as " +
		"sending a message or raising an exception for review.\n\n")
	if len(unused) == 0 {
		buf.WriteString("None.\n\n")
		return
	}
	buf.WriteString("| Tool | Resource | Operation |\n")
	buf.WriteString("| --- | --- | --- |\n")
	for _, name := range unused {
		policy := report.Tools[name]
		fmt.Fprintf(buf, "| `%s` | %s | %s |\n", name, policy.Resource, policy.Operation)
	}
	buf.WriteString("\n")
}

func writeEveryWrite(buf *bytes.Buffer, writes []Write, mapping Mapping) {
	buf.WriteString("## Every write\n")
	domain := ""
	for idx := range writes {
		write := &writes[idx]
		if write.Domain != domain {
			domain = write.Domain
			fmt.Fprintf(buf, "\n### %s\n\n", domain)
			buf.WriteString("| Write | Decision |\n")
			buf.WriteString("| --- | --- |\n")
		}
		fmt.Fprintf(buf, "| %s | %s |\n", writeCell(write), decisionCell(mapping.Writes[write.Key]))
	}
}

func writeCell(write *Write) string {
	parts := []string{code(write.Key)}
	if write.Kind == KindRoute {
		parts = append(parts, write.Handler)
	}
	for _, route := range write.Routes {
		if route.String() != write.Key {
			parts = append(parts, "also "+code(route.String()))
		}
	}
	for _, twin := range write.Twins {
		for _, route := range twin.Routes {
			parts = append(parts, "twin "+code(route.String()))
		}
	}

	return strings.Join(parts, "<br>")
}

func decisionCell(decision Decision) string {
	switch decision.State() {
	case StateCovered:
		names := make([]string, 0, len(decision.Tools))
		for _, name := range decision.Tools {
			names = append(names, code(name))
		}
		return "Tool: " + strings.Join(names, ", ")
	case StateExempt:
		return stringutils.MarkdownTableCell(
			fmt.Sprintf("Exempt, %s: %s", decision.Exempt, decision.Reason),
		)
	case StatePending:
		return stringutils.MarkdownTableCell("Pending: " + orNone(decision.Pending))
	}

	return none
}

func code(text string) string {
	return "`" + text + "`"
}

func orNone(text string) string {
	if strings.TrimSpace(text) == "" {
		return none
	}

	return text
}
