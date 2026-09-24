import type {
  AgentQualityControl,
  AgentQualityOverview,
  AgentQualityRow,
  AgentSuiteRun,
  AgentWorstRatedAnswer,
  QualityPage,
  QualityPageRequest,
} from "@/lib/graphql/agent-quality";
import { stubLayout } from "@/test/layout";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

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

const fetchAgentQualityOverview = vi.fn<() => Promise<AgentQualityOverview>>();
const fetchAgentQualityAgents =
  vi.fn<(page: QualityPageRequest) => Promise<QualityPage<AgentQualityRow>>>();
const fetchAgentWorstRatedAnswers =
  vi.fn<
    (
      agentId: string | null,
      page: QualityPageRequest,
    ) => Promise<QualityPage<AgentWorstRatedAnswer>>
  >();
const fetchAgentQualityControl = vi.fn<() => Promise<AgentQualityControl>>();

vi.mock("@/lib/graphql/agent-quality", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/graphql/agent-quality")>();
  return {
    ...actual,
    fetchAgentQualityOverview: () => fetchAgentQualityOverview(),
    fetchAgentQualityAgents: (page: QualityPageRequest) => fetchAgentQualityAgents(page),
    fetchAgentWorstRatedAnswers: (agentId: string | null, page: QualityPageRequest) =>
      fetchAgentWorstRatedAnswers(agentId, page),
    fetchAgentQualityControl: () => fetchAgentQualityControl(),
    fetchAgentQuality: vi.fn(() => new Promise(() => {})),
    fetchAgentSuiteRuns: vi.fn(() => new Promise(() => {})),
  };
});

const { default: QualityTab } = await import("../quality-tab");

let restoreLayout = () => {};

beforeEach(() => {
  restoreLayout = stubLayout();
  permissions.denied.clear();
  fetchAgentQualityOverview.mockReset();
  fetchAgentQualityAgents.mockReset();
  fetchAgentWorstRatedAnswers.mockReset();
  fetchAgentQualityControl.mockReset();
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

function suiteRun(overrides: Partial<AgentSuiteRun> = {}): AgentSuiteRun {
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
  } as AgentSuiteRun;
}

const rows: AgentQualityRow[] = [
  {
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

function renderTab() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>
      <NuqsTestingAdapter hasMemory>{children}</NuqsTestingAdapter>
    </QueryClientProvider>
  );
  return render(<QualityTab />, { wrapper });
}

function primeDefaults() {
  fetchAgentQualityOverview.mockResolvedValue(overview);
  fetchAgentQualityAgents.mockResolvedValue({
    items: rows,
    endCursor: "c2",
    hasNextPage: true,
    totalCount: 30,
  });
  fetchAgentWorstRatedAnswers.mockResolvedValue({
    items: [],
    endCursor: null,
    hasNextPage: false,
    totalCount: null,
  });
  fetchAgentQualityControl.mockResolvedValue(control);
}

describe("QualityTab", () => {
  it("shows the organization's quality in five figures", async () => {
    primeDefaults();
    renderTab();

    const figures = await screen.findByRole("group", { name: "AI quality figures" });
    await within(figures).findByText("82%");
    expect(within(figures).getByText("140")).toBeTruthy();
    expect(within(figures).getByText("87%")).toBeTruthy();
    expect(within(figures).getByText("2")).toBeTruthy();
    expect(within(figures).getByText("$12.40")).toBeTruthy();
    expect(within(figures).getByText("1 agent still regressed")).toBeTruthy();
  });

  it("pages the agents on the server and draws each one's last run as a phase", async () => {
    primeDefaults();
    renderTab();

    const table = await screen.findByRole("table", { name: "Agents" });
    await within(table).findByText("Billing desk");
    expect(within(table).getByText("Completed")).toBeTruthy();
    expect(within(table).getByText("Budget stopped")).toBeTruthy();
    expect(within(table).getByText("-5 pts")).toBeTruthy();
    expect(within(table).getByText("No ratings")).toBeTruthy();

    expect(fetchAgentQualityAgents).toHaveBeenCalledWith(
      expect.objectContaining({ after: null, includeTotalCount: true }),
    );
    expect(fetchAgentQualityAgents.mock.calls[0][0].first).toBeLessThanOrEqual(100);
  });

  it("asks for the next page after the cursor the first page ended on", async () => {
    primeDefaults();
    const user = userEvent.setup();
    renderTab();

    const panel = await screen.findByRole("region", { name: "Agents" });
    await within(panel).findByText("Billing desk");
    await user.click(within(panel).getByRole("button", { name: "Go to next page" }));

    await waitFor(() =>
      expect(fetchAgentQualityAgents).toHaveBeenCalledWith(
        expect.objectContaining({ after: "c2", includeTotalCount: false }),
      ),
    );
  });

  it("mounts the golden set and the settings", async () => {
    primeDefaults();
    renderTab();

    expect(await screen.findByText("Golden set table")).toBeTruthy();
    expect(await screen.findByRole("button", { name: "Save settings" })).toBeTruthy();
  });

  // Ratings are read under their own right; without it the panel of
  // worst-rated answers is not drawn and nothing about them is asked for.
  it("leaves out the worst-rated answers without the right to read ratings", async () => {
    primeDefaults();
    permissions.denied.add("agent_feedback:read");
    renderTab();

    await screen.findByRole("table", { name: "Agents" });
    expect(screen.queryByRole("region", { name: "Worst-rated answers" })).toBeNull();
    expect(fetchAgentWorstRatedAnswers).not.toHaveBeenCalled();
  });

  it("says when the figures could not be loaded", async () => {
    primeDefaults();
    fetchAgentQualityOverview.mockRejectedValue(new Error("down"));
    renderTab();

    expect(
      await screen.findByText("How well agents are doing could not be loaded. Try again shortly."),
    ).toBeTruthy();
  });

  it("will not save the settings without the right to change AI Control", async () => {
    primeDefaults();
    permissions.denied.add("agent_control:update");
    renderTab();

    const save = await screen.findByRole("button", { name: "Save settings" });
    expect((save as HTMLButtonElement).disabled).toBe(true);
    expect(
      screen.getByText("Changing these needs the right to update AI Control and the golden set."),
    ).toBeTruthy();
  });
});
