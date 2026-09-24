import type {
  AgentQualityControl,
  AgentQualityOverview,
  AgentQualityRow,
  AgentSuiteRun,
  AgentSuiteRunRow,
  AgentWorstRatedRow,
} from "@/lib/graphql/agent-quality";
import { stubLayout } from "@/test/layout";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type {
  ColumnDef,
  DataTableGraphQLSource,
  DataTablePanelProps,
  RowAction,
} from "@trenova/shared/types/data-table";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { NuqsTestingAdapter, type OnUrlUpdateFunction } from "nuqs/adapters/testing";
import type { ComponentType, ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { QualityView } from "../../../ai-control-tabs";

const permissions = vi.hoisted(() => ({ denied: new Set<string>() }));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: (resource: string, operation: string) => ({
    allowed: !permissions.denied.has(`${resource}:${operation}`),
    isLoading: false,
  }),
}));

// The golden set is the existing cases table, tested on its own; here it
// only has to be mounted.
vi.mock("../cases", () => ({
  EvalCasesTable: () => <p>Golden set table</p>,
}));

type StubTableProps = {
  name: string;
  graphql: DataTableGraphQLSource<Record<string, unknown>>;
  columns: ColumnDef<Record<string, unknown>>[];
  contextMenuActions?: RowAction<Record<string, unknown>>[];
  TablePanel?: ComponentType<DataTablePanelProps<Record<string, unknown>>>;
  enableReadOnlyPanel?: boolean;
};

/**
 * The data table has its own tests; here it only has to show what the tab
 * hands it: the rows a test gives it through the tab's own columns, and the
 * tab's own panel on the row a test names.
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

const fetchAgentQualityOverview = vi.fn<() => Promise<AgentQualityOverview>>();
const fetchAgentQualityControl = vi.fn<() => Promise<AgentQualityControl>>();
const fetchAgentQuality = vi.fn();
const fetchAgentSuiteRun = vi.fn();

vi.mock("@/lib/graphql/agent-quality", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/graphql/agent-quality")>();
  return {
    ...actual,
    fetchAgentQualityOverview: () => fetchAgentQualityOverview(),
    fetchAgentQualityControl: () => fetchAgentQualityControl(),
    fetchAgentQuality: (...args: unknown[]) => fetchAgentQuality(...args),
    fetchAgentSuiteRun: (...args: unknown[]) => fetchAgentSuiteRun(...args),
  };
});

const { default: QualityTab } = await import("../quality-tab");

let restoreLayout = () => {};

beforeEach(() => {
  restoreLayout = stubLayout();
  permissions.denied.clear();
  fetchAgentQualityOverview.mockReset();
  fetchAgentQualityControl.mockReset();
  fetchAgentQuality.mockReset();
  fetchAgentSuiteRun.mockReset();
  table.rows = [];
  table.openRow = null;
  table.props = [];
});

afterEach(() => {
  restoreLayout();
  cleanup();
});

const overview: AgentQualityOverview = {
  windowDays: 30,
  since: 1_790_000_000,
  ratingsVisible: true,
  satisfaction: 0.82,
  ratings: 140,
  qualityScore: 0.87,
  agentsScored: 3,
  suiteRuns: 12,
  regressions: 2,
  openRegressions: 1,
  evalSpendMonthUsd: "12.40",
  evalUnpricedCalls: 0,
  monthlyBudgetUsd: "50.00",
  monthStartedAt: 1_788_000_000,
  sweepEnabled: true,
  nextSweepHourLocal: 2,
  nextSweepTimezone: "America/Chicago",
  agentsWithCases: 3,
  judgeEnabled: false,
  regressionThreshold: 0.1,
};

function suiteRun(overrides: Partial<AgentSuiteRun> = {}): AgentSuiteRunRow {
  return {
    id: "asr_1",
    agentDefinitionId: "agdef_1",
    agentName: "Billing desk",
    trigger: "Scheduled",
    fingerprintHash: "f".repeat(64),
    fingerprintChanges: [],
    changeSummary: "Nothing about the agent changed; its cases or the records they read did.",
    suiteRevision: "r".repeat(64),
    status: "Completed",
    casesTotal: 12,
    casesPassed: 11,
    casesFailed: 1,
    casesSkipped: 0,
    hardFailures: 0,
    deterministicScore: 0.9,
    judgeScore: null,
    qualityScore: 0.9,
    baselineScore: 0.88,
    baselineRunId: "asr_0",
    regression: false,
    costUsd: "0.420000",
    startedAt: 1_790_000_000,
    finishedAt: 1_790_000_600,
    comments: "",
    requestedByUserId: null,
    ...overrides,
  } as AgentSuiteRunRow;
}

const rows: AgentQualityRow[] = [
  {
    id: "agdef_1",
    agentDefinitionId: "agdef_1",
    name: "Billing desk",
    enabled: true,
    ratingsVisible: true,
    satisfaction: 0.8,
    satisfactionDelta: -0.05,
    ratings: 40,
    qualityScore: 0.9,
    openRegression: false,
    qualityPoints: [
      { suiteRunId: "asr_0", at: 1, qualityScore: 0.88, status: "Completed", regression: false },
      { suiteRunId: "asr_1", at: 2, qualityScore: 0.9, status: "Completed", regression: false },
    ],
    lastSuiteRun: suiteRun(),
  } as AgentQualityRow,
  {
    id: "agdef_2",
    agentDefinitionId: "agdef_2",
    name: "Dispatch desk",
    enabled: true,
    ratingsVisible: true,
    satisfaction: null,
    satisfactionDelta: null,
    ratings: 0,
    qualityScore: null,
    openRegression: false,
    qualityPoints: [],
    lastSuiteRun: suiteRun({ id: "asr_9", status: "BudgetStopped", qualityScore: null }),
  } as AgentQualityRow,
];

const control: AgentQualityControl = {
  id: null,
  enabled: true,
  runHourLocal: 2,
  timezone: "",
  maxCasesPerAgent: 50,
  nightlyBudgetUsd: "5.00",
  monthlyBudgetUsd: "50.00",
  judgeEnabled: false,
  judgeSampleRate: 0.2,
  regressionThreshold: 0.1,
  minCases: 10,
  forceRerunDays: 7,
  version: 0,
  updatedAt: 0,
} as AgentQualityControl;

function renderTab(view: QualityView, searchParams = "", onUrlUpdate?: OnUrlUpdateFunction) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>
      <NuqsTestingAdapter hasMemory searchParams={searchParams} onUrlUpdate={onUrlUpdate}>
        {children}
      </NuqsTestingAdapter>
    </QueryClientProvider>
  );
  return render(<QualityTab view={view} />, { wrapper });
}

function lastTable(): StubTableProps {
  const props = table.props.at(-1);
  if (!props) {
    throw new Error("no table was drawn");
  }
  return props;
}

function primeDefaults() {
  fetchAgentQualityOverview.mockResolvedValue(overview);
  fetchAgentQualityControl.mockResolvedValue(control);
  fetchAgentQuality.mockReturnValue(new Promise(() => {}));
}

describe("QualityTab", () => {
  it("shows the organization's quality in five figures", async () => {
    primeDefaults();
    renderTab("agents");

    const figures = await screen.findByRole("group", { name: "AI quality figures" });
    await within(figures).findByText("82%");
    expect(within(figures).getByText("140")).toBeTruthy();
    expect(within(figures).getByText("87%")).toBeTruthy();
    expect(within(figures).getByText("2")).toBeTruthy();
    expect(within(figures).getByText("$12.40")).toBeTruthy();
    expect(within(figures).getByText("1 agent still regressed")).toBeTruthy();
  });

  it("says when the figures could not be loaded", async () => {
    primeDefaults();
    fetchAgentQualityOverview.mockRejectedValue(new Error("down"));
    renderTab("agents");

    expect(
      await screen.findByText("How well agents are doing could not be loaded. Try again shortly."),
    ).toBeTruthy();
  });

  // Each view is one data table, so only the view asked for is drawn.
  it("draws the agents in the data table and each one's last run as a phase", async () => {
    primeDefaults();
    table.rows = rows;
    renderTab("agents");

    const agents = await screen.findByRole("region", { name: "Agent Score table" });
    expect(lastTable().graphql.operationName).toBe("AgentQualityAgentTable");
    expect(lastTable().graphql.extraVariables).toEqual({ window: 30 });
    expect(within(agents).getByText("Billing desk")).toBeTruthy();
    expect(within(agents).getAllByText("Completed").length).toBeGreaterThan(0);
    expect(within(agents).getByText("Budget stopped")).toBeTruthy();
    expect(within(agents).getByText("-5 pts")).toBeTruthy();
    expect(within(agents).getByText("No ratings")).toBeTruthy();
    expect(table.props.every((props) => props.name === "Agent Score")).toBe(true);
  });

  // Satisfaction is sorted on the server only for someone who may read it.
  it("sorts by satisfaction only with the right to read ratings", async () => {
    primeDefaults();
    permissions.denied.add(`${Resource.AgentFeedback}:${Operation.Read}`);
    renderTab("agents");

    await screen.findByRole("region", { name: "Agent Score table" });
    const satisfaction = lastTable().columns.find(
      (column) => column.meta?.apiField === "satisfaction",
    );
    expect(satisfaction?.meta?.sortable).toBe(false);
    const actions = lastTable().contextMenuActions ?? [];
    const ratings = actions.find((action) => action.id === "ratings");
    expect(ratings?.hidden?.({ original: rows[0] } as never)).toBe(true);
  });

  // The agent's panel leads to its runs: the runs view, narrowed to the
  // agent, with the table's own state cleared.
  it("opens an agent's suite runs from its panel", async () => {
    primeDefaults();
    table.rows = rows;
    table.openRow = rows[0] as unknown as Record<string, unknown>;
    const updates: URLSearchParams[] = [];
    renderTab("agents", "?panelType=edit&panelEntityId=agdef_1", (event) => {
      updates.push(event.searchParams);
    });

    await userEvent.click(await screen.findByRole("button", { name: "Its suite runs" }));

    await waitFor(() => expect(updates.at(-1)?.get("quality")).toBe("runs"));
    expect(updates.at(-1)?.get("tab")).toBe("quality");
    expect(updates.at(-1)?.get("agent")).toBe("agdef_1");
    expect(updates.at(-1)?.get("panelType")).toBeNull();
  });

  it("narrows the suite runs to the agent a link names, and widens them again", async () => {
    primeDefaults();
    fetchAgentQuality.mockResolvedValue({ agentName: "Billing desk" });
    table.rows = [suiteRun()];
    const updates: URLSearchParams[] = [];
    renderTab("runs", "?tab=quality&quality=runs&agent=agdef_1", (event) => {
      updates.push(event.searchParams);
    });

    await screen.findByRole("region", { name: "Suite Run table" });
    expect(lastTable().graphql.operationName).toBe("AgentSuiteRunTable");
    expect(lastTable().graphql.extraVariables).toEqual({ agentDefinitionId: "agdef_1" });
    expect(await screen.findByText("Showing Billing desk only")).toBeTruthy();

    await userEvent.click(screen.getByRole("button", { name: "Show every agent" }));
    await waitFor(() => expect(updates.at(-1)?.get("agent")).toBeNull());
  });

  // A notification links to a run's cases; they replace the runs, with the
  // way back above them.
  it("shows a run's cases in place of the runs", async () => {
    primeDefaults();
    fetchAgentSuiteRun.mockResolvedValue(suiteRun());
    const updates: URLSearchParams[] = [];
    renderTab("runs", "?tab=quality&agent=agdef_1&suiteRun=asr_1", (event) => {
      updates.push(event.searchParams);
    });

    await screen.findByRole("region", { name: "Suite Run Case table" });
    expect(lastTable().graphql.operationName).toBe("AgentSuiteRunCaseTable");
    expect(lastTable().graphql.extraVariables).toEqual({ suiteRunId: "asr_1" });
    expect(fetchAgentSuiteRun).toHaveBeenCalledWith("asr_1", expect.anything());

    await userEvent.click(screen.getByRole("button", { name: "Back to suite runs" }));
    await waitFor(() => expect(updates.at(-1)?.get("suiteRun")).toBeNull());
    expect(updates.at(-1)?.get("agent")).toBe("agdef_1");
  });

  it("opens what a person saw when they rated an answer down", async () => {
    primeDefaults();
    const answer = {
      id: "AssistantMessage:msg_1:",
      targetType: "AssistantMessage",
      targetId: "msg_1",
      targetPart: "",
      positive: 0,
      negative: 3,
      lastRatedAt: 1_790_000_000,
      threadId: "thr_1",
      canOpenThread: false,
      agentDefinitionId: "agdef_1",
      agentName: "Billing desk",
      sample: {
        id: "fb_1",
        reasons: [],
        comment: "Wrong invoice total.",
        createdAt: 1_790_000_000,
        turnSnapshot: {
          question: "What is the total on invoice 12?",
          answer: "It is $12.",
          tools: [],
          omittedTools: 0,
          redacted: false,
        },
      },
    } as unknown as AgentWorstRatedRow;
    table.rows = [answer as unknown as Record<string, unknown>];
    table.openRow = answer as unknown as Record<string, unknown>;
    renderTab("ratings");

    await screen.findByRole("region", { name: "Worst-Rated Answer table" });
    expect(lastTable().graphql.extraVariables).toEqual({ window: 30 });
    expect(await screen.findByText("Wrong invoice total.")).toBeTruthy();
    expect(screen.getAllByText("What is the total on invoice 12?").length).toBeGreaterThan(0);
  });

  it("mounts the golden set and the settings as views of their own", async () => {
    primeDefaults();
    renderTab("golden");
    expect(await screen.findByText("Golden set table")).toBeTruthy();
    cleanup();

    renderTab("settings");
    expect(await screen.findByRole("button", { name: "Save settings" })).toBeTruthy();
    expect(screen.queryByRole("group", { name: "AI quality figures" })).toBeNull();
  });

  it("will not save the settings without the right to change AI Control", async () => {
    primeDefaults();
    permissions.denied.add(`${Resource.AgentControl}:${Operation.Update}`);
    renderTab("settings");

    const save = await screen.findByRole("button", { name: "Save settings" });
    expect((save as HTMLButtonElement).disabled).toBe(true);
    expect(
      screen.getByText("Changing these needs the right to update AI Control and the golden set."),
    ).toBeTruthy();
  });
});
