import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useController, type Control, type FieldValues, type Path } from "react-hook-form";
import { MemoryRouter } from "react-router";
import {
  Operation,
  Resource,
  type OperationType,
  type ResourceType,
} from "@trenova/shared/types/permission";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import OrgStructureConsole from "../org-structure-console";

const mocks = vi.hoisted(() => ({
  fetchHeadcount: vi.fn(),
  fetchJobPositions: vi.fn(),
  fetchApprovalDelegations: vi.fn(),
  fetchPositionHolders: vi.fn(),
  assignWorkerPosition: vi.fn(),
  assignUserPosition: vi.fn(),
  revokeApprovalDelegation: vi.fn(),
  createJobPosition: vi.fn(),
  updateJobPosition: vi.fn(),
  delegateApproval: vi.fn(),
}));

vi.mock("@/lib/graphql/org-structure", () => ({
  ...mocks,
  HEADCOUNT_KEY: "headcount",
  JOB_POSITIONS_KEY: "job-positions",
  APPROVAL_DELEGATIONS_KEY: "approval-delegations",
  POSITION_HOLDERS_KEY: "position-holders",
  MY_TEAM_KEY: "my-team",
}));

// The pickers are async autocompletes; the sheet only needs a field the form
// can read.
vi.mock("@/components/autocomplete-fields", () => {
  const Field = ({
    control,
    name,
    label,
  }: {
    control: Control<FieldValues>;
    name: Path<FieldValues>;
    label?: string;
  }) => {
    const { field } = useController({ control, name });
    return (
      <input
        aria-label={label}
        value={(field.value as string) ?? ""}
        onChange={(event) => field.onChange(event.target.value)}
      />
    );
  };
  return { WorkerAutocompleteField: Field, UserAutocompleteField: Field };
});

// Everything is allowed unless a test takes a permission away; the chart
// must offer nothing it would refuse.
const permissions = vi.hoisted(() => ({ denied: new Set<string>() }));
vi.mock("@/hooks/use-permission", () => ({
  usePermission: (resource: string, operation: string) => ({
    allowed: !permissions.denied.has(`${resource}:${operation}`),
    isLoading: false,
  }),
}));
// Resource and Operation are const objects, not types; the type of a member is
// ResourceType / OperationType.
function deny(resource: ResourceType, operation: OperationType) {
  permissions.denied.add(`${resource}:${operation}`);
}

vi.mock("@trenova/shared/stores/auth-store", () => {
  const state = { user: { id: "usr_me", name: "Me" } };
  const useAuthStore = (selector: (s: typeof state) => unknown) => selector(state);
  useAuthStore.getState = () => state;
  return { useAuthStore };
});

vi.mock("../position-dialog", () => ({
  PositionDialog: ({
    open,
    position,
    defaultReportsToPositionId,
  }: {
    open: boolean;
    position: { id: string } | null;
    defaultReportsToPositionId?: string | null;
  }) =>
    open ? (
      <div
        data-testid="position-dialog"
        data-position={position?.id ?? ""}
        data-reports-to={defaultReportsToPositionId ?? ""}
      />
    ) : null,
}));
vi.mock("../delegation-dialog", () => ({ DelegationDialog: () => null }));

vi.mock("@number-flow/react", () => ({
  default: ({ value, className, ...rest }: { value: number; className?: string }) => (
    <span className={className} {...rest}>
      {value}
    </span>
  ),
}));

const DAY = 86_400;
const NOW = Date.UTC(2026, 8, 7, 15, 30) / 1000;

function position(over: Record<string, unknown> = {}) {
  return {
    id: "pos_otr",
    status: "Active",
    code: "OTR",
    title: "Over-the-Road Driver",
    description: null,
    department: "Operations",
    flsaExempt: false,
    isDrivingPosition: true,
    reportsToPositionId: null,
    version: 1,
    reportsTo: null,
    ...over,
  };
}

function row(key: string, label: string, workers: number, over: Record<string, unknown> = {}) {
  return {
    key,
    label,
    code: label,
    color: "",
    workers,
    drivers: workers,
    terminated: 0,
    staff: 0,
    ...over,
  };
}

function headcount(over: Record<string, unknown> = {}) {
  return {
    activeTotal: 43,
    driverTotal: 40,
    terminated: 2,
    staffTotal: 0,
    byFleet: [row("fc_s", "SOUTH", 30, { color: "#2563eb" }), row("fc_n", "NORTH", 13)],
    byPosition: [
      row("pos_otr", "Over-the-Road Driver", 30),
      row("pos_loc", "Local Driver", 10),
      row("pos_ops", "Operations Manager", 2, { drivers: 0 }),
      row("pos_ceo", "Chief Executive", 1, { drivers: 0 }),
    ],
    byDepartment: [row("Operations", "Operations", 42), row("Executive", "Executive", 1)],
    ...over,
  };
}

const ORG = [
  position({
    id: "pos_ceo",
    code: "CEO",
    title: "Chief Executive",
    department: "Executive",
    isDrivingPosition: false,
    flsaExempt: true,
  }),
  position({
    id: "pos_ops",
    code: "OPS",
    title: "Operations Manager",
    isDrivingPosition: false,
    reportsToPositionId: "pos_ceo",
  }),
  position({ id: "pos_otr", reportsToPositionId: "pos_ops" }),
  position({ id: "pos_loc", code: "LOC", title: "Local Driver", reportsToPositionId: "pos_ops" }),
  position({
    id: "pos_saf",
    code: "SAF",
    title: "Safety Director",
    department: "Safety",
    isDrivingPosition: false,
    reportsToPositionId: "pos_ceo",
  }),
];

function delegation(over: Record<string, unknown> = {}) {
  return {
    id: "dlg_1",
    delegatorId: "usr_me",
    delegateId: "usr_ben",
    scope: "All",
    startsAt: NOW - 5 * DAY,
    endsAt: NOW + 3 * DAY,
    reason: "Holiday",
    revokedAt: null,
    version: 1,
    delegator: { id: "usr_me", name: "Me" },
    delegate: { id: "usr_ben", name: "Ben Cole" },
    ...over,
  };
}

function renderConsole(
  fixtures: {
    headcount?: ReturnType<typeof headcount>;
    positions?: unknown[];
    given?: unknown[];
    received?: unknown[];
    holders?: Record<string, unknown[]>;
  } = {},
) {
  mocks.fetchHeadcount.mockResolvedValue(fixtures.headcount ?? headcount());
  mocks.fetchJobPositions.mockResolvedValue(fixtures.positions ?? ORG);
  mocks.fetchApprovalDelegations.mockImplementation(
    async (args: { delegatorId?: string; delegateId?: string }) =>
      args.delegatorId ? (fixtures.given ?? []) : (fixtures.received ?? []),
  );
  mocks.revokeApprovalDelegation.mockResolvedValue({ id: "dlg_1", revokedAt: NOW, version: 2 });
  mocks.fetchPositionHolders.mockImplementation(
    async (positionId: string) => fixtures.holders?.[positionId] ?? [],
  );
  mocks.assignWorkerPosition.mockResolvedValue(true);
  mocks.assignUserPosition.mockResolvedValue(true);
  mocks.updateJobPosition.mockResolvedValue({
    id: "pos_saf",
    title: "Safety Director",
    version: 2,
  });
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <OrgStructureConsole />
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

beforeEach(() => {
  // The whole Date has to move, not just Date.now: the delegation panel dates
  // itself with getTodayDate, which builds a `new Date()`, and a spy on
  // Date.now leaves that reading the wall clock — which quietly turned every
  // fixture below into a fixture about the day the file was written. Only Date
  // is faked, so the timers userEvent and waitFor run on stay real.
  vi.useFakeTimers({ toFake: ["Date"], shouldAdvanceTime: true });
  vi.setSystemTime(NOW * 1000);
});

afterEach(() => {
  cleanup();
  permissions.denied.clear();
  vi.useRealTimers();
  vi.restoreAllMocks();
  vi.clearAllMocks();
});

describe("OrgStructureConsole", () => {
  // The loading state is the loaded page drawn in grey: the KPI strip, the
  // indented chart, the cover panel and the aside, at the sizes they take
  // once the headcount lands, so the page does not reflow when it does.
  it("draws the page's own shape while the headcount is still being read", () => {
    mocks.fetchHeadcount.mockReturnValue(new Promise(() => {}));
    mocks.fetchJobPositions.mockReturnValue(new Promise(() => {}));
    mocks.fetchApprovalDelegations.mockReturnValue(new Promise(() => {}));
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <MemoryRouter>
        <QueryClientProvider client={client}>
          <OrgStructureConsole />
        </QueryClientProvider>
      </MemoryRouter>,
    );

    const loading = screen.getByLabelText("Loading the organisation");
    expect(loading).toHaveAttribute("aria-busy", "true");
    const hidden = { hidden: true } as const;
    const chart = within(loading).getByRole("region", { name: "Org chart", ...hidden });
    expect(within(chart).getAllByRole("listitem", hidden).length).toBeGreaterThan(1);
    for (const name of [
      "Approval cover",
      "By terminal",
      "By department",
      "Titles nobody holds yet",
    ]) {
      expect(within(loading).getByRole("region", { name, ...hidden })).toBeInTheDocument();
    }
  });

  it("heads the page with people, positions, terminals and cover", async () => {
    renderConsole({
      given: [delegation()],
      received: [
        delegation({
          id: "dlg_2",
          delegatorId: "usr_ada",
          delegateId: "usr_me",
          endsAt: null,
          delegator: { id: "usr_ada", name: "Ada Byrne" },
          delegate: { id: "usr_me", name: "Me" },
        }),
      ],
    });

    expect(await screen.findByLabelText("People", { selector: "span" })).toHaveTextContent("43");
    expect(screen.getByRole("img", { name: /Who they are/ })).toHaveAccessibleName(
      "Who they are: Drivers 40, Other workers 3, Staff 0",
    );
    expect(screen.getByLabelText("Positions", { selector: "span" })).toHaveTextContent("5");
    expect(screen.getByText("1 title nobody holds")).toBeInTheDocument();
    expect(screen.getByLabelText("Terminals", { selector: "span" })).toHaveTextContent("2");
    expect(screen.getByText("Largest: SOUTH, 30 people")).toBeInTheDocument();
    expect(await screen.findByLabelText("Cover in force", { selector: "span" })).toHaveTextContent(
      "2",
    );
    expect(screen.getByText("1 ending this week · 1 until called back")).toBeInTheDocument();
  });

  // A list rather than boxes and lines: each position under the one it
  // reports to, with its own people and the people beneath it.
  it("draws the chart as an indented tree with rolled-up headcounts", async () => {
    renderConsole();

    const user = userEvent.setup();
    const list = await screen.findByRole("list", { name: "Positions" });
    await user.click(screen.getByRole("button", { name: "Expand all" }));
    const items = within(list).getAllByRole("listitem");
    expect(items.map((item) => item.getAttribute("aria-label"))).toEqual([
      "Chief Executive",
      "Operations Manager",
      "Over-the-Road Driver",
      "Local Driver",
      "Safety Director",
    ]);
    expect(items.map((item) => item.getAttribute("data-depth"))).toEqual(["0", "1", "2", "2", "1"]);
    expect(within(items[0]).getByLabelText("Chief Executive headcount")).toHaveTextContent(
      "1 · 42 below",
    );
    expect(within(items[1]).getByLabelText("Operations Manager headcount")).toHaveTextContent(
      "2 · 40 below",
    );
    expect(within(items[2]).getByLabelText("Over-the-Road Driver headcount")).toHaveTextContent(
      "30",
    );
    expect(within(items[0]).getByText("Exempt")).toBeInTheDocument();
    expect(within(items[2]).getByText("Driving")).toBeInTheDocument();
  });

  it("finds a position and keeps the chain above it", async () => {
    const user = userEvent.setup();
    renderConsole();

    await screen.findByRole("list", { name: "Positions" });
    await user.type(screen.getByLabelText("Find a position"), "local");

    await waitFor(() => {
      const items = within(screen.getByRole("list", { name: "Positions" })).getAllByRole(
        "listitem",
      );
      expect(items.map((item) => item.getAttribute("aria-label"))).toEqual([
        "Chief Executive",
        "Operations Manager",
        "Local Driver",
      ]);
    });
  });

  it("names open positions nobody holds and counts people with no position", async () => {
    renderConsole({
      headcount: headcount({
        byPosition: [row("pos_otr", "Over-the-Road Driver", 30), row("", "No position", 3)],
      }),
    });

    const panel = await screen.findByRole("region", { name: "Titles nobody holds yet" });
    expect(within(panel).getByText("Chief Executive")).toBeInTheDocument();
    expect(within(panel).getByText("Local Driver")).toBeInTheDocument();
    expect(within(panel).queryByText("Over-the-Road Driver")).not.toBeInTheDocument();
    expect(screen.getByText("3 people with no position")).toBeInTheDocument();
  });

  it("shows terminals and departments as shares of the roster", async () => {
    renderConsole();

    const terminals = await screen.findByRole("region", { name: "By terminal" });
    expect(within(terminals).getByRole("img", { name: "SOUTH: 30 of 43" })).toBeInTheDocument();
    const departments = screen.getByRole("region", { name: "By department" });
    expect(
      within(departments).getByRole("img", { name: "Operations: 42 of 43" }),
    ).toBeInTheDocument();
  });

  // In force comes first in either list, because that is what somebody opens
  // the panel for; the other side is one click away with its own count.
  it("shows both sides of cover with counts, in force first, and calls one back", async () => {
    const user = userEvent.setup();
    renderConsole({
      given: [
        delegation({
          id: "dlg_old",
          endsAt: NOW - 10 * DAY,
          startsAt: NOW - 20 * DAY,
          delegate: { id: "usr_cal", name: "Cal Diaz" },
        }),
        delegation(),
      ],
      received: [
        delegation({
          id: "dlg_r",
          delegatorId: "usr_ada",
          delegateId: "usr_me",
          delegator: { id: "usr_ada", name: "Ada Byrne" },
          delegate: { id: "usr_me", name: "Me" },
        }),
      ],
    });

    const control = await screen.findByRole("radiogroup", { name: "Which cover to show" });
    await waitFor(() =>
      expect(within(control).getByRole("radio", { name: /Handed out/ })).toHaveTextContent(
        "1 in force",
      ),
    );
    expect(within(control).getByRole("radio", { name: /Covering for/ })).toHaveTextContent(
      "1 in force",
    );

    const given = await screen.findByRole("list", { name: "Handed out" });
    const rows = within(given).getAllByRole("listitem");
    expect(within(rows[0]).getByText("Ben Cole")).toBeInTheDocument();
    expect(within(rows[0]).getByText("In force")).toBeInTheDocument();
    expect(within(rows[1]).getByText("Cal Diaz")).toBeInTheDocument();
    expect(within(rows[1]).getByText("Finished")).toBeInTheDocument();
    expect(within(rows[1]).queryByRole("button", { name: /Call back/ })).not.toBeInTheDocument();

    await user.click(
      within(rows[0]).getByRole("button", { name: "Call back the delegation to Ben Cole" }),
    );
    expect(mocks.revokeApprovalDelegation).toHaveBeenCalledWith("dlg_1");

    await user.click(within(control).getByRole("radio", { name: /Covering for/ }));
    const received = await screen.findByRole("list", { name: "Covering for" });
    expect(within(received).getByText("Ada Byrne")).toBeInTheDocument();
  });

  it("says how to start when there is nothing to count", async () => {
    renderConsole({
      headcount: headcount({
        activeTotal: 0,
        driverTotal: 0,
        terminated: 0,
        byFleet: [],
        byPosition: [],
        byDepartment: [],
      }),
      positions: [],
    });

    expect(await screen.findByText("Nothing to count yet")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Add a position" })).toBeInTheDocument();
  });

  // A roster with people but no titles is where a fresh tenant sits; the
  // first position has to be creatable from here, not only from an empty
  // roster or an existing chart.
  it("offers the first position when people are on the roster but no titles are", async () => {
    const user = userEvent.setup();
    renderConsole({
      headcount: headcount({ byPosition: [row("", "No position", 43)] }),
      positions: [],
    });

    expect(await screen.findByText("No positions yet")).toBeInTheDocument();
    expect(screen.queryByRole("list", { name: "Positions" })).not.toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Add a position" }));

    const dialog = screen.getByTestId("position-dialog");
    expect(dialog).toHaveAttribute("data-position", "");
    expect(dialog).toHaveAttribute("data-reports-to", "");
  });

  it("offers no way to add a position without permission to create one", async () => {
    deny(Resource.JobPosition, Operation.Create);
    const user = userEvent.setup();

    renderConsole({
      headcount: headcount({ byPosition: [row("", "No position", 43)] }),
      positions: [],
    });
    expect(await screen.findByText("No positions yet")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Add a position/ })).not.toBeInTheDocument();
    cleanup();

    renderConsole({
      headcount: headcount({
        activeTotal: 0,
        driverTotal: 0,
        terminated: 0,
        byFleet: [],
        byPosition: [],
        byDepartment: [],
      }),
      positions: [],
    });
    expect(await screen.findByText("Nothing to count yet")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /Add a position/ })).not.toBeInTheDocument();
    cleanup();

    renderConsole();
    await screen.findByRole("list", { name: "Positions" });
    await user.click(screen.getByRole("button", { name: "Expand all" }));
    expect(screen.queryByRole("button", { name: /Add a position/ })).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: "More for Safety Director" })).toBeInTheDocument();
  });

  // A deep chart fits on a screen only if branches fold; the top level and
  // its direct reports are open, the rest is a click away.
  it("opens one level deep and folds branches on demand", async () => {
    const user = userEvent.setup();
    renderConsole();

    const list = await screen.findByRole("list", { name: "Positions" });
    const labels = () =>
      within(list)
        .getAllByRole("listitem")
        .map((item) => item.getAttribute("aria-label"));
    expect(labels()).toEqual(["Chief Executive", "Operations Manager", "Safety Director"]);

    await user.click(screen.getByRole("button", { name: "Expand Operations Manager" }));
    expect(labels()).toEqual([
      "Chief Executive",
      "Operations Manager",
      "Over-the-Road Driver",
      "Local Driver",
      "Safety Director",
    ]);

    await user.click(screen.getByRole("button", { name: "Collapse all" }));
    expect(labels()).toEqual(["Chief Executive"]);
  });

  it("opens folded branches on the way to a search match", async () => {
    const user = userEvent.setup();
    renderConsole();

    await screen.findByRole("list", { name: "Positions" });
    await user.type(screen.getByLabelText("Find a position"), "local");

    await waitFor(() =>
      expect(
        within(screen.getByRole("list", { name: "Positions" }))
          .getAllByRole("listitem")
          .map((item) => item.getAttribute("aria-label")),
      ).toEqual(["Chief Executive", "Operations Manager", "Local Driver"]),
    );
  });

  it("adds a position straight under the one it will report to", async () => {
    const user = userEvent.setup();
    renderConsole();

    await screen.findByRole("list", { name: "Positions" });
    await user.click(
      screen.getByRole("button", { name: "Add a position under Operations Manager" }),
    );

    const dialog = screen.getByTestId("position-dialog");
    expect(dialog).toHaveAttribute("data-reports-to", "pos_ops");
    expect(dialog).toHaveAttribute("data-position", "");
  });

  // The move is the same update the edit form makes, with everything else
  // carried as it was; only the reports-to changes.
  it("moves a position under another from its menu, carrying the rest unchanged", async () => {
    const user = userEvent.setup();
    renderConsole();

    await screen.findByRole("list", { name: "Positions" });
    await user.click(screen.getByRole("button", { name: "More for Safety Director" }));
    const menu = await screen.findByRole("menu");
    expect(
      within(menu).queryByRole("menuitem", { name: "Safety Director" }),
    ).not.toBeInTheDocument();
    await user.click(within(menu).getByRole("menuitem", { name: "Operations Manager" }));

    await waitFor(() => expect(mocks.updateJobPosition).toHaveBeenCalledTimes(1));
    expect(mocks.updateJobPosition).toHaveBeenCalledWith({
      id: "pos_saf",
      version: 1,
      code: "SAF",
      title: "Safety Director",
      description: undefined,
      department: "Safety",
      flsaExempt: false,
      isDrivingPosition: false,
      reportsToPositionId: "pos_ops",
      status: "Active",
    });
  });

  it("does not offer a position's own branch as somewhere to move it", async () => {
    const user = userEvent.setup();
    renderConsole();

    await screen.findByRole("list", { name: "Positions" });
    await user.click(screen.getByRole("button", { name: "More for Operations Manager" }));
    const menu = await screen.findByRole("menu");
    const names = within(menu)
      .getAllByRole("menuitem")
      .map((item) => item.textContent?.trim());
    expect(names).toContain("Nothing (top level)");
    expect(names).toContain("Safety Director");
    expect(names).not.toContain("Over-the-Road Driver");
    expect(names).not.toContain("Local Driver");
    expect(names).not.toContain("Chief Executive");
  });

  // A dispatch desk is held by somebody who logs in. The chart counts them
  // beside the drivers, or every front-office title would read as empty.
  it("counts staff on a title beside the workers", async () => {
    renderConsole({
      headcount: headcount({
        staffTotal: 2,
        byPosition: [
          row("pos_otr", "Over-the-Road Driver", 30),
          row("pos_ops", "Operations Manager", 0, { drivers: 0, staff: 2 }),
        ],
      }),
    });

    expect(await screen.findByLabelText("People", { selector: "span" })).toHaveTextContent("45");
    const list = await screen.findByRole("list", { name: "Positions" });
    expect(
      within(list).getByRole("button", { name: "Operations Manager headcount" }),
    ).toHaveTextContent("2 · 30 below");
    const vacant = screen.getByRole("region", { name: "Titles nobody holds yet" });
    expect(within(vacant).queryByText("Operations Manager")).not.toBeInTheDocument();
  });

  // A front-office title is filled from the user picker and a driving title
  // from the worker picker; the sheet is where the chart is managed from.
  it("shows who holds a title and puts a user on a front-office one", async () => {
    const user = userEvent.setup();
    renderConsole({
      holders: {
        pos_ops: [
          { kind: "User", id: "usr_ada", name: "Ada Byrne", status: "Active", detail: "ada@x.io" },
          { kind: "User", id: "usr_old", name: "Gus Hale", status: "Inactive", detail: "gus@x.io" },
        ],
      },
    });

    const list = await screen.findByRole("list", { name: "Positions" });
    await user.click(within(list).getByRole("button", { name: "Operations Manager headcount" }));

    const people = await screen.findByRole("region", { name: "People in the position" });
    expect(await within(people).findByText("Ada Byrne")).toBeInTheDocument();
    expect(within(people).getByText("Gus Hale")).toBeInTheDocument();
    expect(within(people).getByText("Inactive")).toBeInTheDocument();
    expect(within(people).getByText("1 active · 2 on record")).toBeInTheDocument();

    await user.type(screen.getByLabelText("Put somebody on it"), "usr_ben");
    await user.click(screen.getByRole("button", { name: "Add" }));
    await waitFor(() =>
      expect(mocks.assignUserPosition).toHaveBeenCalledWith("usr_ben", "pos_ops"),
    );
    expect(mocks.assignWorkerPosition).not.toHaveBeenCalled();

    await user.click(
      within(people).getByRole("button", { name: "Take Ada Byrne off the position" }),
    );
    await waitFor(() => expect(mocks.assignUserPosition).toHaveBeenCalledWith("usr_ada", null));
  });

  it("puts a worker on a driving title from the worker picker", async () => {
    const user = userEvent.setup();
    renderConsole();

    const list = await screen.findByRole("list", { name: "Positions" });
    await user.click(screen.getByRole("button", { name: "Expand all" }));
    await user.click(within(list).getByRole("button", { name: "Local Driver headcount" }));

    await screen.findByRole("region", { name: "People in the position" });
    await user.type(screen.getByLabelText("Put a worker on it"), "wrk_cal");
    await user.click(screen.getByRole("button", { name: "Add" }));
    await waitFor(() =>
      expect(mocks.assignWorkerPosition).toHaveBeenCalledWith("wrk_cal", "pos_loc"),
    );
    expect(mocks.assignUserPosition).not.toHaveBeenCalled();
  });
});
