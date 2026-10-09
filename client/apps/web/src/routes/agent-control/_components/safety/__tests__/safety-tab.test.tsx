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
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { definition, rule, safety, tool } from "./fixtures";

const api = vi.hoisted(() => ({
  fetchAgentSafetySummary: vi.fn(),
  fetchAgentSafetyHeaders: vi.fn(),
  fetchAgentToolHolders: vi.fn(),
  fetchToolHolderAnswers: vi.fn(),
  fetchAgentDefinitions: vi.fn(),
}));

vi.mock("@/lib/graphql/agent-safety", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/agent-safety")>()),
  fetchAgentSafetySummary: () => api.fetchAgentSafetySummary(),
  fetchAgentSafetyHeaders: (...args: unknown[]) => api.fetchAgentSafetyHeaders(...args),
  fetchAgentToolHolders: () => api.fetchAgentToolHolders(),
  fetchToolHolderAnswers: (...args: unknown[]) => api.fetchToolHolderAnswers(...args),
}));
vi.mock("@/lib/graphql/agent-definition", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/agent-definition")>()),
  fetchAgentDefinitions: () => api.fetchAgentDefinitions(),
}));

type StubTableProps = {
  name: string;
  graphql: DataTableGraphQLSource<Record<string, unknown>>;
  columns: ColumnDef<Record<string, unknown>>[];
  TablePanel?: ComponentType<DataTablePanelProps<Record<string, unknown>>>;
  enableReadOnlyPanel?: boolean;
  enableCreateAction?: boolean;
};

/**
 * The data table has its own tests; here it only has to show what the tab hands it: the
 * rows a test gives it through the tab's own columns, and the tab's own panel on the row a
 * test names.
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

const { default: SafetyTab } = await import("../safety-tab");

const summary: AgentSafetySummary = {
  toolCount: 132,
  runWithoutPerson: 2,
  leaveOrganization: 9,
  openWithSensitive: 4,
  resources: ["shipment", "general", "customer"],
  egressCounts: [
    { egress: "Internal", count: 30 },
    { egress: "CustomerVisible", count: 5 },
    { egress: "Money", count: 2 },
  ],
  unattendedTools: ["Assign move", "Release billing hold"],
  openSensitiveAgentIds: ["agdef_1", "agdef_2", "agdef_3", "agdef_4"],
};

const agents = [
  definition("agdef_1", "Customer desk", { toolNames: ["email_customer"] }),
  definition("agdef_2", "Dispatch desk"),
  definition("agdef_3", "Billing desk", { enabled: false }),
  definition("agdef_4", "Rates desk"),
  definition("agdef_5", "Safety desk"),
];

const emailCustomer = rule(
  {
    name: "email_customer",
    title: "Email customer",
    egress: ["ExternalRecipient"],
    leavesOrganization: true,
    promotableTier: "ActWithApproval",
    needs: { resource: "customer", operation: "update" },
    rationale: "An email reaches a customer and cannot be taken back.",
  },
  false,
);

let restoreLayout = () => {};

beforeEach(() => {
  restoreLayout = stubLayout();
  api.fetchAgentSafetySummary.mockResolvedValue(summary);
  api.fetchAgentSafetyHeaders.mockResolvedValue([]);
  api.fetchAgentToolHolders.mockResolvedValue(
    new Map([
      ["email_customer", ["agdef_1", "agdef_2"]],
      ["assign_move", ["agdef_1", "agdef_2", "agdef_3", "agdef_4", "agdef_5"]],
    ]),
  );
  api.fetchToolHolderAnswers.mockResolvedValue([]);
  api.fetchAgentDefinitions.mockResolvedValue(agents);
  table.rows = [];
  table.openRow = null;
  table.props = [];
});

afterEach(() => {
  restoreLayout();
  cleanup();
  vi.clearAllMocks();
});

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

describe("SafetyTab", () => {
  it("says what runs without a person, what leaves, and which open agents hold it", async () => {
    renderTab("rules");

    expect(await screen.findByRole("link", { name: "2 tools run" })).toBeInTheDocument();
    expect(
      screen.getByText(/can send outside the organization, and every one waits/),
    ).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "4 agents" })).toBeInTheDocument();
    expect(
      screen.getByText("on at least one agent · Assign move, Release billing hold"),
    ).toBeInTheDocument();
  });

  it("says nothing runs alone and offers no review when nothing does and nothing is open", async () => {
    api.fetchAgentSafetySummary.mockResolvedValue({
      ...summary,
      runWithoutPerson: 0,
      unattendedTools: [],
      openWithSensitive: 0,
      openSensitiveAgentIds: [],
    });
    renderTab("rules");

    expect(await screen.findByText(/No tool runs without a person\./)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Review open agents" })).not.toBeInTheDocument();
  });

  it("narrows the rules to the tools that run alone", async () => {
    const updates: URLSearchParams[] = [];
    renderTab("rules", "", (event) => updates.push(event.searchParams));

    await userEvent.click(await screen.findByRole("link", { name: "2 tools run" }));

    await waitFor(() => expect(updates.at(-1)?.get("fieldFilters")).toContain("runsWithoutPerson"));
  });

  it("compares the first three open agents when asked to review them", async () => {
    const updates: URLSearchParams[] = [];
    renderTab("rules", "", (event) => updates.push(event.searchParams));

    await userEvent.click(await screen.findByRole("button", { name: "Review open agents" }));

    await waitFor(() => expect(updates.at(-1)?.get("safety")).toBe("agents"));
    expect(updates.at(-1)?.get("safetyAgents")).toBe("agdef_1,agdef_2,agdef_3");
  });

  it("maps the tools that change things by who sees their work, and narrows the rules by one", async () => {
    const updates: URLSearchParams[] = [];
    renderTab("rules", "", (event) => updates.push(event.searchParams));

    const map = await screen.findByRole("group", { name: "Who sees the work" });
    expect(
      within(map)
        .getAllByRole("button")
        .map((button) => button.textContent),
    ).toEqual(["30Internal", "5Customer", "2Money"]);
    expect(screen.getByText("37 tools that change things")).toBeInTheDocument();

    await userEvent.click(within(map).getByRole("button", { name: /Customer/ }));
    await waitFor(() => expect(updates.at(-1)?.get("fieldFilters")).toContain("CustomerVisible"));
  });

  it("marks the audience the rules are narrowed to, and clears it", async () => {
    const filters = encodeURIComponent(
      JSON.stringify([{ field: "egress", operator: "eq", value: "Money" }]),
    );
    const updates: URLSearchParams[] = [];
    renderTab("rules", `?fieldFilters=${filters}`, (event) => updates.push(event.searchParams));

    const map = await screen.findByRole("group", { name: "Who sees the work" });
    expect(within(map).getByRole("button", { name: /Money/ })).toHaveAttribute(
      "aria-pressed",
      "true",
    );

    await userEvent.click(screen.getByRole("button", { name: "Clear" }));
    await waitFor(() => expect(updates.at(-1)?.get("fieldFilters")).toBeNull());
  });

  it("says so when the figures cannot be loaded", async () => {
    api.fetchAgentSafetySummary.mockRejectedValue(new Error("offline"));
    renderTab("rules");

    expect(
      await screen.findByText(
        "What agents can do without a person could not be loaded. Try again shortly.",
      ),
    ).toBeInTheDocument();
  });
});

describe("Tool rules", () => {
  // The rules table is a lazy chunk; load it once up front so a test waits on the
  // table it asserts about, not on the first transform of its module graph.
  beforeAll(async () => {
    await import("../tool-rules-table");
  });

  it("draws the rules from the rule connection with the agents that hold each tool", async () => {
    table.rows = [rule({ name: "assign_move", title: "Assign move" }, true), emailCustomer];
    renderTab("rules");

    const rules = await screen.findByRole("region", { name: "Tool Rule table" });
    expect(lastTable().graphql.operationName).toBe("AgentToolRuleTable");
    expect(lastTable().enableCreateAction).toBe(false);
    expect(within(rules).getByText("Outside recipient")).toBeInTheDocument();
    await waitFor(() =>
      expect(within(rules).getByLabelText("Customer desk, Dispatch desk")).toBeInTheDocument(),
    );
    expect(within(rules).getByText("+1")).toBeInTheDocument();
  });

  it("opens a rule with who holds it and what each holder makes of it", async () => {
    api.fetchToolHolderAnswers.mockResolvedValue([
      tool(
        "email_customer",
        { answer: "NEEDS_APPROVAL", tier: "ActWithApproval" },
        { answer: "PROPOSE_ONLY", tier: "Propose" },
      ),
    ]);
    table.rows = [emailCustomer];
    table.openRow = emailCustomer;
    renderTab("rules");

    expect(await screen.findByText("Held by 2 agents")).toBeInTheDocument();
    expect(screen.getByText("Leaves the organization — never past approval")).toBeInTheDocument();
    expect(
      screen.getByText("An email reaches a customer and cannot be taken back."),
    ).toBeInTheDocument();
    expect(await screen.findByText("Needs approval")).toBeInTheDocument();
    expect(screen.getByText("Proposes only")).toBeInTheDocument();
    expect(api.fetchToolHolderAnswers.mock.calls[0]?.slice(0, 2)).toEqual([
      "email_customer",
      ["agdef_1", "agdef_2"],
    ]);
  });
});

describe("By agent", () => {
  it("reads no agent until one is picked, and offers every agent to start from", async () => {
    const updates: URLSearchParams[] = [];
    renderTab("agents", "", (event) => updates.push(event.searchParams));

    expect(
      await screen.findByText("Pick an agent to see what it can do without a person"),
    ).toBeInTheDocument();
    expect(api.fetchAgentSafetyHeaders).not.toHaveBeenCalled();
    expect(table.props).toHaveLength(0);

    await userEvent.click(await screen.findByRole("button", { name: "Dispatch desk" }));
    await waitFor(() => expect(updates.at(-1)?.get("safetyAgents")).toBe("agdef_2"));
  });

  it("reads the picked agents and lists their tools in one table", async () => {
    api.fetchAgentSafetyHeaders.mockResolvedValue([
      safety("agdef_1", "Customer desk", {
        reach: {
          accessMode: "Everyone",
          roles: [],
          warnings: [{ kind: "OpenWithSensitiveTools", tools: ["email_customer"] }],
        },
      }),
    ]);
    table.rows = [tool("assign_move", { answer: "RUNS_ON_ITS_OWN" })];
    renderTab("agents", "?safetyAgents=agdef_1");

    expect(await screen.findByRole("button", { name: "Limit to roles" })).toBeInTheDocument();
    expect(api.fetchAgentSafetyHeaders.mock.calls[0]?.[0]).toEqual(["agdef_1"]);
    expect(lastTable().graphql.extraVariables).toEqual({ agentIds: ["agdef_1"] });
    await userEvent.click(
      screen.getByRole("button", { name: "1 tool that leaves the organization" }),
    );
    expect(screen.getByText(/Email customer\./)).toBeInTheDocument();
  });

  it("takes an agent out and starts the table over", async () => {
    const filters = encodeURIComponent(
      JSON.stringify([{ field: "agentId", operator: "eq", value: "agdef_1" }]),
    );
    const updates: URLSearchParams[] = [];
    renderTab("agents", `?safetyAgents=agdef_1&fieldFilters=${filters}`, (event) =>
      updates.push(event.searchParams),
    );

    await userEvent.click(
      await screen.findByRole("button", { name: "Remove Customer desk from the comparison" }),
    );

    await waitFor(() => expect(updates.at(-1)?.get("safetyAgents")).toBeNull());
    expect(updates.at(-1)?.get("fieldFilters")).toBeNull();
  });

  it("offers no more than four agents at once", async () => {
    renderTab("agents", "?safetyAgents=agdef_1,agdef_2,agdef_3,agdef_4");

    await screen.findByRole("button", { name: "Remove Rates desk from the comparison" });
    expect(screen.queryByRole("button", { name: "Compare another" })).not.toBeInTheDocument();
  });

  it("opens a tool from an agent's row with that agent marked among the holders", async () => {
    const email = tool("email_customer", { answer: "NEEDS_APPROVAL", tier: "ActWithApproval" });
    table.rows = [email];
    table.openRow = email;
    renderTab("agents", "?safetyAgents=agdef_1");

    const holders = await screen.findByText("Held by 2 agents");
    const row = (
      await within(holders.parentElement as HTMLElement).findByText("Customer desk")
    ).closest(".hl-r");
    expect(row).toHaveClass("on");
  });
});
