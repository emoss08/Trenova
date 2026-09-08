import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import MyTeamConsole from "../my-team-console";

const { fetchMyTeam, fetchApprovalDelegations } = vi.hoisted(() => ({
  fetchMyTeam: vi.fn(),
  fetchApprovalDelegations: vi.fn(),
}));

vi.mock("@/lib/graphql/org-structure", () => ({
  fetchMyTeam,
  fetchApprovalDelegations,
  MY_TEAM_KEY: "my-team",
  APPROVAL_DELEGATIONS_KEY: "approval-delegations",
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));

vi.mock("@trenova/shared/stores/auth-store", () => {
  const state = { user: { id: "usr_me", name: "Me" } };
  const useAuthStore = (selector: (s: typeof state) => unknown) => selector(state);
  useAuthStore.getState = () => state;
  return { useAuthStore };
});

// NumberFlow is a custom element; the DOM under test only needs the digits.
vi.mock("@number-flow/react", () => ({
  default: ({ value, className, ...rest }: { value: number; className?: string }) => (
    <span className={className} {...rest}>
      {value}
    </span>
  ),
}));

const DAY = 86_400;
const NOW = Date.UTC(2026, 8, 7, 15, 30) / 1000;
const ME = "usr_me";
const BOSS = "usr_boss";

function member(over: Record<string, unknown> = {}) {
  return {
    workerId: "wrk_1",
    name: "Ada Byrne",
    status: "Active",
    fleetCodeId: "fc_1",
    fleetCode: "SOUTH",
    fleetColor: "#2563eb",
    positionId: "jpos_1",
    positionTitle: "Over-the-Road Driver",
    direct: true,
    managerId: ME,
    complianceStatus: "Compliant",
    trainingHealth: "Current",
    safetyRating: "Excellent",
    hireDate: Date.UTC(2023, 10, 14) / 1000,
    terminationDate: null,
    ...over,
  };
}

function delegation(over: Record<string, unknown> = {}) {
  return {
    id: "dlg_1",
    delegatorId: BOSS,
    delegateId: ME,
    scope: "All",
    startsAt: NOW - 5 * DAY,
    endsAt: NOW + 5 * DAY,
    reason: null,
    revokedAt: null,
    version: 1,
    delegator: { id: BOSS, name: "Bea Boss" },
    delegate: { id: ME, name: "Me" },
    ...over,
  };
}

function renderConsole(rows: unknown[], delegations: unknown[] = []) {
  fetchMyTeam.mockResolvedValue(rows);
  fetchApprovalDelegations.mockResolvedValue(delegations);
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <MyTeamConsole />
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

function region(name: string | RegExp) {
  return screen.getByRole("region", { name });
}

beforeEach(() => {
  vi.spyOn(Date, "now").mockReturnValue(NOW * 1000);
});

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
  vi.clearAllMocks();
});

describe("MyTeamConsole", () => {
  // The loading state is the loaded page drawn in grey: the KPI strip, the
  // attention panel, the grouped roster and the aside, at the sizes they take
  // once the team lands, so the page does not reflow when it does.
  it("draws the page's own shape while the team is still being read", () => {
    fetchMyTeam.mockReturnValue(new Promise(() => {}));
    fetchApprovalDelegations.mockReturnValue(new Promise(() => {}));
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <MemoryRouter>
        <QueryClientProvider client={client}>
          <MyTeamConsole />
        </QueryClientProvider>
      </MemoryRouter>,
    );

    const loading = screen.getByLabelText("Loading your team");
    expect(loading).toHaveAttribute("aria-busy", "true");
    const hidden = { hidden: true } as const;
    const direct = within(loading).getByRole("region", { name: "Direct reports", ...hidden });
    expect(within(direct).getAllByRole("listitem", hidden).length).toBeGreaterThan(0);
    for (const name of ["Needs your attention", "By terminal", "Coming up", "Approval cover"]) {
      expect(within(loading).getByRole("region", { name, ...hidden })).toBeInTheDocument();
    }
  });

  // A terminal manager covering forty drivers is not forty direct reports, and
  // a page that conflated the two would misstate the span of control.
  it("separates direct reports from people reached through a terminal", async () => {
    renderConsole([
      member(),
      member({ workerId: "wrk_2", name: "Ben Cole", direct: false, managerId: null }),
      member({ workerId: "wrk_3", name: "Cal Diaz", direct: false, managerId: null }),
    ]);

    const direct = await screen.findByRole("region", { name: "Direct reports" });
    expect(within(direct).getByText("Ada Byrne")).toBeInTheDocument();
    expect(within(direct).queryByText("Ben Cole")).not.toBeInTheDocument();

    const terminal = region("Through a terminal you run");
    expect(within(terminal).getByText("Ben Cole")).toBeInTheDocument();
    expect(within(terminal).getByText("Cal Diaz")).toBeInTheDocument();
  });

  it("counts the paths separately in the summary", async () => {
    renderConsole([
      member(),
      member({ workerId: "wrk_2", name: "Ben Cole", direct: false, managerId: null }),
      member({ workerId: "wrk_3", name: "Cal Diaz", direct: false, managerId: null }),
    ]);

    const bar = await screen.findByRole("img", { name: /How they reach you/ });
    expect(bar).toHaveAccessibleName("How they reach you: Direct 1, Terminal 2");
    expect(screen.getByLabelText("On your team")).toHaveTextContent("3");
  });

  // The server flags a delegator's own reports as direct. The page has to say
  // whose they really are, or a stand-in would read a borrowed team as theirs.
  it("files a delegator's reports under the manager being covered", async () => {
    renderConsole(
      [member(), member({ workerId: "wrk_2", name: "Ben Cole", managerId: BOSS })],
      [delegation()],
    );

    const covering = await screen.findByRole("region", { name: "Covering for Bea Boss" });
    expect(within(covering).getByText("Ben Cole")).toBeInTheDocument();
    expect(within(covering).getByText(/Everything · until/)).toBeInTheDocument();
    expect(within(region("Direct reports")).queryByText("Ben Cole")).not.toBeInTheDocument();

    const cover = region("Approval cover");
    expect(within(cover).getByText("Bea Boss")).toBeInTheDocument();
    expect(within(cover).getByText("Everything")).toBeInTheDocument();

    const bar = screen.getByRole("img", { name: /How they reach you/ });
    expect(bar).toHaveAccessibleName(/Covering 1$/);
  });

  it("does not credit cover from a delegation that has ended", async () => {
    renderConsole([member({ managerId: BOSS })], [delegation({ endsAt: NOW - DAY })]);

    await screen.findByRole("region", { name: "Direct reports" });
    expect(screen.queryByRole("region", { name: /Covering for/ })).not.toBeInTheDocument();
    expect(
      within(region("Approval cover")).getByText(/Nobody has handed you their approvals/),
    ).toBeInTheDocument();
  });

  // The point of the page is knowing who needs chasing, so the count has to
  // cover every way somebody can need it — not just one.
  it("counts anybody non-compliant, blocked on training or at risk", async () => {
    renderConsole([
      member(),
      member({ workerId: "wrk_2", name: "Ben Cole", complianceStatus: "NonCompliant" }),
      member({ workerId: "wrk_3", name: "Cal Diaz", trainingHealth: "Overdue" }),
      member({ workerId: "wrk_4", name: "Dee Ellis", safetyRating: "AtRisk" }),
      member({ workerId: "wrk_5", name: "Eve Fox", trainingHealth: "Missing" }),
    ]);

    expect(await screen.findByLabelText("Needing attention")).toHaveTextContent("4");
    expect(screen.getByText("1 compliance · 2 training · 1 safety")).toBeInTheDocument();
  });

  it("lists the flagged people with every reason, worst first", async () => {
    renderConsole([
      member(),
      member({ workerId: "wrk_2", name: "Ben Cole", complianceStatus: "NonCompliant" }),
      member({
        workerId: "wrk_3",
        name: "Cal Diaz",
        trainingHealth: "Overdue",
        safetyRating: "AtRisk",
      }),
      member({ workerId: "wrk_4", name: "Dee Ellis", safetyRating: "Watch" }),
    ]);

    const panel = await screen.findByRole("region", { name: "Needs your attention" });
    const items = within(panel).getAllByRole("link");
    expect(items.map((item) => within(item).getByText(/Cole|Diaz/).textContent)).toEqual([
      "Cal Diaz",
      "Ben Cole",
    ]);
    expect(within(items[0]).getByText("Training overdue")).toBeInTheDocument();
    expect(within(items[0]).getByText("Safety at risk")).toBeInTheDocument();
    expect(within(panel).queryByText("Ada Byrne")).not.toBeInTheDocument();
    expect(within(panel).getByText("1 to keep an eye on")).toBeInTheDocument();
  });

  it("says so when nobody needs attention, and names who is on watch", async () => {
    renderConsole([
      member(),
      member({ workerId: "wrk_2", name: "Ben Cole", safetyRating: "Watch" }),
    ]);

    const panel = await screen.findByRole("region", { name: "Needs your attention" });
    expect(within(panel).getByText(/Everyone is in good standing/)).toBeInTheDocument();
    expect(within(panel).getByText("Ben Cole")).toBeInTheDocument();
  });

  it("narrows the roster to a typed name without touching the attention list", async () => {
    const user = userEvent.setup();
    renderConsole([
      member(),
      member({ workerId: "wrk_2", name: "Ben Cole", complianceStatus: "NonCompliant" }),
      member({ workerId: "wrk_3", name: "Cal Diaz", direct: false, managerId: null }),
    ]);

    await screen.findByRole("region", { name: "Direct reports" });
    await user.type(screen.getByLabelText("Find someone"), "cal");

    const roster = region("Roster");
    expect(within(roster).getByText("Cal Diaz")).toBeInTheDocument();
    expect(within(roster).queryByText("Ada Byrne")).not.toBeInTheDocument();
    expect(within(roster).queryByText("Ben Cole")).not.toBeInTheDocument();
    expect(within(region("Needs your attention")).getByText("Ben Cole")).toBeInTheDocument();
  });

  it("filters the roster to people needing attention", async () => {
    const user = userEvent.setup();
    renderConsole([
      member(),
      member({ workerId: "wrk_2", name: "Ben Cole", complianceStatus: "NonCompliant" }),
    ]);

    await screen.findByRole("region", { name: "Direct reports" });
    await user.click(screen.getByRole("button", { name: "Needs attention" }));

    const roster = region("Roster");
    expect(within(roster).getByText("Ben Cole")).toBeInTheDocument();
    expect(within(roster).queryByText("Ada Byrne")).not.toBeInTheDocument();
  });

  it("refetches with leavers included when the switch is turned on", async () => {
    const user = userEvent.setup();
    renderConsole([member()]);

    await screen.findByRole("region", { name: "Direct reports" });
    expect(fetchMyTeam).toHaveBeenLastCalledWith(false, expect.anything());

    fetchMyTeam.mockResolvedValue([
      member(),
      member({
        workerId: "wrk_2",
        name: "Gus Hale",
        status: "Terminated",
        terminationDate: NOW - 10 * DAY,
      }),
    ]);
    await user.click(screen.getByRole("switch", { name: "Include people who have left" }));

    expect(fetchMyTeam).toHaveBeenLastCalledWith(true, expect.anything());
    const row = (await screen.findByText("Gus Hale")).closest("a") as HTMLElement;
    expect(within(row).getByText("Left")).toBeInTheDocument();
  });

  it("shows the team by terminal with counts", async () => {
    renderConsole([
      member(),
      member({ workerId: "wrk_2", name: "Ben Cole", fleetCodeId: "fc_2", fleetCode: "NORTH" }),
      member({ workerId: "wrk_3", name: "Cal Diaz", fleetCodeId: "fc_2", fleetCode: "NORTH" }),
    ]);

    const panel = await screen.findByRole("region", { name: "By terminal" });
    const bars = within(panel).getAllByRole("img");
    expect(bars.map((bar) => bar.getAttribute("aria-label"))).toEqual([
      "NORTH: 2 of 3",
      "SOUTH: 1 of 3",
    ]);
  });

  it("lists anniversaries and new starters inside the window", async () => {
    renderConsole([
      member({ name: "Ada Byrne", hireDate: Date.UTC(2024, 8, 12) / 1000 }),
      member({ workerId: "wrk_2", name: "New Joiner", hireDate: NOW - 3 * DAY }),
      member({ workerId: "wrk_3", name: "Far Off", hireDate: Date.UTC(2020, 11, 25) / 1000 }),
    ]);

    const panel = await screen.findByRole("region", { name: "Coming up" });
    expect(within(panel).getByText("Ada Byrne")).toBeInTheDocument();
    expect(within(panel).getByText(/2 years on/)).toBeInTheDocument();
    expect(within(panel).getByText("In 5d")).toBeInTheDocument();
    expect(within(panel).getByText("New Joiner")).toBeInTheDocument();
    expect(within(panel).getByText("3d ago")).toBeInTheDocument();
    expect(within(panel).queryByText("Far Off")).not.toBeInTheDocument();
  });

  it("says how somebody joins the team when nobody has", async () => {
    renderConsole([]);

    expect(
      await screen.findByRole("heading", { level: 3, name: "Nobody reports to you yet" }),
    ).toBeInTheDocument();
    expect(
      screen.getByText(/A worker joins your team when their record names you/),
    ).toBeInTheDocument();
    expect(screen.getByText(/A worker joins your team/).closest("div")).not.toHaveAttribute(
      "aria-hidden",
    );
  });

  it("draws the roster it is waiting for, out of reach of assistive technology", async () => {
    renderConsole([]);

    await screen.findByRole("heading", { level: 3, name: "Nobody reports to you yet" });
    const sketch = document.querySelector("[aria-hidden]");
    expect(sketch).not.toBeNull();
    expect(sketch!.querySelectorAll(".bg-muted").length).toBeGreaterThan(3);
    expect(sketch!.textContent).toBe("");
  });

  it("keeps the approval cover panel under the empty roster", async () => {
    renderConsole([]);

    await screen.findByRole("heading", { level: 3, name: "Nobody reports to you yet" });
    expect(
      within(region("Approval cover")).getByText(/Nobody has handed you their approvals/),
    ).toBeInTheDocument();
  });
});
