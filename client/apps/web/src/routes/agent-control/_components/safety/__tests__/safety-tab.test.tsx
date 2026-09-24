import type { AgentChoice } from "@/lib/graphql/agent-definition";
import type { AgentSafetySummary } from "@/lib/graphql/agent-safety";
import { stubLayout } from "@/test/layout";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type {
  ColumnDef,
  DataTableGraphQLSource,
  DataTablePanelProps,
} from "@trenova/shared/types/data-table";
import { NuqsTestingAdapter, type OnUrlUpdateFunction } from "nuqs/adapters/testing";
import type { ComponentType, ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { policy, rule, safety, tool } from "./fixtures";

const fetchAgentSafetySummary = vi.fn<() => Promise<AgentSafetySummary>>();
const fetchAgentSafetyHeaders = vi.fn();

vi.mock("@/lib/graphql/agent-safety", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/graphql/agent-safety")>();
  return {
    ...actual,
    fetchAgentSafetySummary: () => fetchAgentSafetySummary(),
    fetchAgentSafetyHeaders: (...args: unknown[]) => fetchAgentSafetyHeaders(...args),
  };
});

type StubTableProps = {
  name: string;
  graphql: DataTableGraphQLSource<Record<string, unknown>>;
  columns: ColumnDef<Record<string, unknown>>[];
  TablePanel?: ComponentType<DataTablePanelProps<Record<string, unknown>>>;
  enableReadOnlyPanel?: boolean;
  enableCreateAction?: boolean;
};

/**
 * The data table has its own tests; here it only has to show what the tab
 * hands it. It draws the rows a test gives it through the tab's own columns,
 * and opens the tab's own panel on the row a test names.
 */
const table = vi.hoisted(() => ({
  rows: [] as Record<string, unknown>[],
  openRow: null as Record<string, unknown> | null,
  props: [] as StubTableProps[],
}));

vi.mock("@/components/data-table/data-table", () => ({
  DataTable: (props: StubTableProps) => {
    table.props.push(props);
    const { TablePanel } = props;
    return (
      <section aria-label={`${props.name} table`}>
        <table>
          <thead>
            <tr>
              {props.columns.map((column, index) => (
                <th key={index}>
                  {typeof column.header === "string" ? column.header : (column.id ?? "")}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {table.rows.map((row, rowIndex) => (
              <tr key={rowIndex}>
                {props.columns.map((column, index) => (
                  <td key={index}>
                    {typeof column.cell === "function"
                      ? (column.cell as (context: unknown) => ReactNode)({ row: { original: row } })
                      : null}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
        {TablePanel && table.openRow ? (
          <TablePanel open onOpenChange={() => {}} mode="edit" row={table.openRow} />
        ) : null}
      </section>
    );
  },
}));

// The picker has its own tests; here it only has to hand back an agent.
const pickerProps = vi.fn();
vi.mock("@/components/assistant/agent-picker", () => ({
  AgentPickerList: (props: {
    source: string;
    hiddenIds: ReadonlySet<string>;
    onSelect: (agent: AgentChoice) => void;
  }) => {
    pickerProps(props);
    return (
      <button type="button" onClick={() => props.onSelect(choice("agdef_1", "Customer desk"))}>
        Pick Customer desk
      </button>
    );
  },
}));

const { default: SafetyTab } = await import("../safety-tab");

let restoreLayout = () => {};

beforeEach(() => {
  restoreLayout = stubLayout();
  fetchAgentSafetySummary.mockReset();
  fetchAgentSafetyHeaders.mockReset();
  pickerProps.mockReset();
  table.rows = [];
  table.openRow = null;
  table.props = [];
});

afterEach(() => {
  restoreLayout();
  cleanup();
});

function choice(id: string, name: string): AgentChoice {
  return {
    id,
    name,
    description: "",
    template: null,
    icon: "",
    accent: "",
    toolNames: [],
    systemKey: "",
    starters: [],
  };
}

function renderTab(view: "rules" | "agents", searchParams = "", onUrlUpdate?: OnUrlUpdateFunction) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>
      <NuqsTestingAdapter hasMemory searchParams={searchParams} onUrlUpdate={onUrlUpdate}>
        {children}
      </NuqsTestingAdapter>
    </QueryClientProvider>
  );
  return render(<SafetyTab view={view} />, { wrapper });
}

function lastTable(): StubTableProps {
  const props = table.props.at(-1);
  if (!props) {
    throw new Error("no table was drawn");
  }
  return props;
}

const summary: AgentSafetySummary = {
  toolCount: 132,
  runWithoutPerson: 4,
  leaveOrganization: 9,
  openWithSensitive: 1,
  resources: ["shipment", "general", "customer"],
};

const emailCustomer = rule(
  {
    name: "email_customer",
    title: "Email customer",
    egress: ["ExternalRecipient"],
    leavesOrganization: true,
    promotableTier: "ActWithApproval",
    needs: { resource: "customer", operation: "update" },
    rationale: "An email reaches a customer and cannot be taken back.",
    hasCondition: true,
    conditionDescription: "Only to a contact on the customer's record.",
  },
  false,
);

describe("SafetyTab", () => {
  it("heads both views with the figures counted on the server", async () => {
    fetchAgentSafetySummary.mockResolvedValue(summary);

    renderTab("rules");

    const figures = await screen.findByRole("group", { name: "AI safety figures" });
    expect(within(figures).getByText("4")).toBeTruthy();
    expect(within(figures).getByText("9")).toBeTruthy();
    expect(within(figures).getByText("1")).toBeTruthy();
  });

  it("says so when the figures cannot be loaded", async () => {
    fetchAgentSafetySummary.mockRejectedValue(new Error("offline"));

    renderTab("rules");

    expect(
      await screen.findByText(
        "What agents can do without a person could not be loaded. Try again shortly.",
      ),
    ).toBeTruthy();
  });
});

describe("Tool rules", () => {
  // The rules are the app's data table over the rule connection, with a
  // read-only panel, so they scroll, page, filter and sort like every table.
  it("draws the rules in the data table from the rule connection", async () => {
    fetchAgentSafetySummary.mockResolvedValue(summary);
    table.rows = [
      rule({ name: "assign_move", title: "Assign move" }, true),
      emailCustomer,
      rule({
        name: "get_inbound_message",
        title: "Get inbound message",
        kind: "Query",
        effect: "Lookup",
        egress: ["None"],
        readsExternal: "Always",
        source: "InboundMessage",
      }),
    ];

    renderTab("rules");

    const rules = await screen.findByRole("region", { name: "Tool Rule table" });
    const props = lastTable();
    expect(props.graphql.operationName).toBe("AgentToolRuleTable");
    expect(props.graphql.connectionKey).toBe("agentToolRuleConnection");
    expect(props.enableReadOnlyPanel).toBe(true);
    expect(props.enableCreateAction).toBe(false);

    expect(within(rules).getAllByText("email_customer").length).toBeGreaterThan(0);
    expect(within(rules).getByText("Outside recipient")).toBeTruthy();
    expect(within(rules).getByText("Always, from inbound messages")).toBeTruthy();
    expect(within(rules).getByText("On at least one agent")).toBeTruthy();
  });

  // Every column the toolbar can filter or sort names a field the server
  // answers to; the resources a rule can need come from the figures.
  it("filters and sorts on the server's fields", async () => {
    fetchAgentSafetySummary.mockResolvedValue(summary);

    renderTab("rules");

    await waitFor(() =>
      expect(
        lastTable().columns.find((column) => column.meta?.apiField === "resource")?.meta
          ?.filterOptions,
      ).toHaveLength(3),
    );
    const fields = lastTable()
      .columns.filter((column) => column.meta?.filterable)
      .map((column) => column.meta?.apiField);
    expect(fields).toEqual([
      "title",
      "name",
      "egress",
      "maxTier",
      "resource",
      "kind",
      "readsExternal",
      "runsWithoutPerson",
    ]);
  });

  // The rationale and the condition are long; they open in the row's panel
  // rather than widening the table.
  it("opens a rule's rationale and condition in its panel", async () => {
    fetchAgentSafetySummary.mockResolvedValue(summary);
    table.rows = [emailCustomer];
    table.openRow = emailCustomer;

    renderTab("rules");

    expect(
      await screen.findByText("An email reaches a customer and cannot be taken back."),
    ).toBeTruthy();
    expect(screen.getByText("Only to a contact on the customer's record.")).toBeTruthy();
  });
});

describe("By agent", () => {
  // Nothing about any agent is read, and no table is drawn, until someone
  // picks an agent.
  it("reads no agent until one is picked", async () => {
    fetchAgentSafetySummary.mockResolvedValue(summary);

    renderTab("agents");

    const byAgent = await screen.findByRole("region", { name: "By agent" });
    expect(
      within(byAgent).getByText("Add an agent to see what it can do without a person."),
    ).toBeTruthy();
    expect(fetchAgentSafetyHeaders).not.toHaveBeenCalled();
    expect(table.props).toHaveLength(0);
  });

  // The picked agents go to the server as the query's own argument, and one
  // table holds every picked agent's tools.
  it("reads the picked agent and lists its tools in one table", async () => {
    fetchAgentSafetySummary.mockResolvedValue(summary);
    fetchAgentSafetyHeaders.mockResolvedValue([
      safety("agdef_1", "Customer desk", {
        reach: {
          accessMode: "Everyone",
          roles: [],
          warnings: [{ kind: "OpenWithSensitiveTools", tools: ["email_customer"] }],
        },
      }),
    ]);
    table.rows = [
      tool("assign_move", { answer: "RUNS_ON_ITS_OWN" }),
      tool(
        "email_customer",
        { answer: "NEEDS_APPROVAL", tier: "ActWithApproval", heldBy: ["egress_class"] },
        { answer: "NEEDS_APPROVAL", tier: "ActWithApproval", heldBy: ["egress_class", "tainted"] },
        { egress: ["ExternalRecipient"], leavesOrganization: true },
      ),
    ];
    const updates: URLSearchParams[] = [];

    renderTab("agents", "", (event) => {
      updates.push(event.searchParams);
    });

    const byAgent = await screen.findByRole("region", { name: "By agent" });
    await userEvent.click(within(byAgent).getByRole("button", { name: /add an agent/i }));
    await userEvent.click(await screen.findByRole("button", { name: "Pick Customer desk" }));

    const header = await within(byAgent).findByRole("region", { name: "Customer desk" });
    expect(fetchAgentSafetyHeaders.mock.calls[0]?.[0]).toEqual(["agdef_1"]);
    expect(pickerProps.mock.calls.at(-1)?.[0].source).toBe("grantable");
    expect(within(header).getByText("Everyone who can use the assistant")).toBeTruthy();
    expect(within(header).getByText(/leave the organization: /)).toBeTruthy();
    expect(updates.at(-1)?.get("safetyAgents")).toBe("agdef_1");

    const tools = await screen.findByRole("region", { name: "Agent Tool table" });
    expect(lastTable().graphql.operationName).toBe("AgentToolSafetyTable");
    expect(lastTable().graphql.extraVariables).toEqual({ agentIds: ["agdef_1"] });
    expect(within(tools).getByText("Read outside text")).toBeTruthy();
    expect(within(tools).getAllByText("Runs on its own")).toHaveLength(2);
    expect(
      lastTable().columns.find((column) => column.meta?.apiField === "agentId")?.meta
        ?.filterOptions,
    ).toEqual([{ value: "agdef_1", label: "Customer desk" }]);
  });

  // Taking an agent out also clears the table's filters, which could name it.
  it("takes an agent out of the comparison and starts the table over", async () => {
    fetchAgentSafetySummary.mockResolvedValue(summary);
    fetchAgentSafetyHeaders.mockResolvedValue([safety("agdef_1", "Customer desk")]);
    const updates: URLSearchParams[] = [];
    const filters = encodeURIComponent(
      JSON.stringify([{ field: "agentId", operator: "eq", value: "agdef_1" }]),
    );

    renderTab("agents", `?safetyAgents=agdef_1&fieldFilters=${filters}`, (event) => {
      updates.push(event.searchParams);
    });

    const byAgent = await screen.findByRole("region", { name: "By agent" });
    await userEvent.click(
      await within(byAgent).findByRole("button", {
        name: "Remove Customer desk from the comparison",
      }),
    );

    await waitFor(() => expect(updates.at(-1)?.get("safetyAgents")).toBeNull());
    expect(updates.at(-1)?.get("fieldFilters")).toBeNull();
    expect(
      await within(byAgent).findByText("Add an agent to see what it can do without a person."),
    ).toBeTruthy();
  });

  it("offers to take out an agent that no longer exists", async () => {
    fetchAgentSafetySummary.mockResolvedValue(summary);
    fetchAgentSafetyHeaders.mockResolvedValue([]);

    renderTab("agents", "?safetyAgents=agdef_gone");

    const byAgent = await screen.findByRole("region", { name: "By agent" });
    expect(await within(byAgent).findByText("This agent no longer exists.")).toBeTruthy();
    expect(within(byAgent).getByRole("button", { name: "Remove" })).toBeTruthy();
  });

  it("opens what the agent makes of a tool in the row's panel", async () => {
    fetchAgentSafetySummary.mockResolvedValue(summary);
    fetchAgentSafetyHeaders.mockResolvedValue([safety("agdef_1", "Customer desk")]);
    const email = tool(
      "email_customer",
      { answer: "NEEDS_APPROVAL", tier: "ActWithApproval", heldBy: ["egress_class"] },
      { answer: "NEEDS_APPROVAL", tier: "ActWithApproval", heldBy: ["tainted"] },
      { rationale: policy({}).rationale },
    );
    table.rows = [email];
    table.openRow = email;

    renderTab("agents", "?safetyAgents=agdef_1");

    expect(await screen.findByText("Held by Customer desk")).toBeTruthy();
    expect(screen.getByText("Assigning a move changes only internal records.")).toBeTruthy();
  });
});
