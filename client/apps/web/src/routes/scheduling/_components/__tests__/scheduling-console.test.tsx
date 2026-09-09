import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import SchedulingConsole from "../scheduling-console";

const mocks = vi.hoisted(() => ({
  fetchRota: vi.fn(),
  fetchShiftTemplates: vi.fn(),
  fetchShiftSwapRequests: vi.fn(),
  transitionShiftSwap: vi.fn(),
  createShiftTemplate: vi.fn(),
  updateShiftTemplate: vi.fn(),
}));

vi.mock("@/lib/graphql/scheduling", () => ({
  ...mocks,
  ROTA_KEY: "rota",
  SHIFT_TEMPLATES_KEY: "shift-templates",
  SHIFT_SWAPS_KEY: "shift-swaps",
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));

vi.mock("@/components/autocomplete-fields", () => ({
  FleetCodeAutocompleteField: () => <input aria-label="Fleet" />,
}));

vi.mock("../shift-template-dialog", () => ({ ShiftTemplateDialog: () => null }));

vi.mock("@number-flow/react", () => ({
  default: ({ value, className, ...rest }: { value: number; className?: string }) => (
    <span className={className} {...rest}>
      {value}
    </span>
  ),
}));

const DAY = 86_400;
// A Monday afternoon; the rota week began on the Sunday before it.
const NOW = Date.UTC(2026, 8, 7, 15, 30) / 1000;
const WEEK_START = Date.UTC(2026, 8, 6) / 1000;

type DayOver = Partial<{ state: string; scheduled: boolean; isConflict: boolean }>;

function day(index: number, over: DayOver = {}) {
  return {
    date: WEEK_START + index * DAY,
    state: "Scheduled",
    scheduled: true,
    startMinute: 480,
    durationMinutes: 480,
    preference: null,
    assignmentCount: 0,
    isConflict: false,
    ...over,
  };
}

const week = (pattern: (index: number) => DayOver = () => ({})) =>
  Array.from({ length: 7 }, (_, index) => day(index, pattern(index)));

function row(name: string, days = week(), over: Record<string, unknown> = {}) {
  return {
    workerId: `wrk_${name.split(" ")[0].toLowerCase()}`,
    name,
    fleetCode: "SOUTH",
    fleetColor: "#2563eb",
    shiftCode: "DAYS",
    shiftName: "Days",
    shiftColor: null,
    days,
    scheduledDays: days.filter((d) => d.scheduled).length,
    conflicts: days.filter((d) => d.isConflict).length,
    ...over,
  };
}

function rota(rows: ReturnType<typeof row>[]) {
  return {
    weekStart: WEEK_START,
    weekEnd: WEEK_START + 7 * DAY,
    weeks: 1,
    rows,
    scheduledDays: rows.reduce((sum, r) => sum + r.scheduledDays, 0),
    conflicts: rows.reduce((sum, r) => sum + r.conflicts, 0),
  };
}

function template(over: Record<string, unknown> = {}) {
  return {
    id: "sht_1",
    status: "Active",
    code: "DAYS",
    name: "Days",
    description: null,
    color: null,
    daysOfWeek: "0111110",
    startMinute: 480,
    durationMinutes: 480,
    cycleWeeks: 1,
    version: 1,
    activeAssignmentCount: 3,
    ...over,
  };
}

function swap(over: Record<string, unknown> = {}) {
  return {
    id: "swp_1",
    requestingWorkerId: "wrk_ada",
    counterpartyWorkerId: "wrk_ben",
    status: "Accepted",
    shiftDate: WEEK_START + 2 * DAY,
    counterpartyShiftDate: null,
    reason: null,
    responseNote: null,
    version: 1,
    requestingWorker: { id: "wrk_ada", firstName: "Ada", lastName: "Byrne" },
    counterpartyWorker: { id: "wrk_ben", firstName: "Ben", lastName: "Cole" },
    ...over,
  };
}

function renderConsole(fixtures: {
  rota?: ReturnType<typeof rota>;
  templates?: unknown[];
  swaps?: unknown[];
}) {
  mocks.fetchRota.mockResolvedValue(fixtures.rota ?? rota([]));
  mocks.fetchShiftTemplates.mockResolvedValue(fixtures.templates ?? []);
  mocks.fetchShiftSwapRequests.mockResolvedValue(fixtures.swaps ?? []);
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <SchedulingConsole />
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

beforeEach(() => {
  // Faking the Date class rather than spying on Date.now(): the console reads today
  // through `new Date()`, which a Date.now spy does not touch. With only the spy the
  // fixture's "today" was whatever day the suite happened to run on, so the cover and
  // conflict counts below only lined up on 2026-09-07. `toFake` is limited to Date so
  // setTimeout stays real and waitFor still works.
  vi.useFakeTimers({ toFake: ["Date"] });
  vi.setSystemTime(NOW * 1000);
  window.localStorage.clear();
});

afterEach(() => {
  cleanup();
  vi.useRealTimers();
  vi.restoreAllMocks();
  vi.clearAllMocks();
});

describe("SchedulingConsole", () => {
  // While a week is read the rota tab draws the board's own outline, in the
  // density and width the reader chose, so the loaded week lands where its
  // outline was rather than replacing two grey blocks.
  it("draws the board's own shape while the week is still being read", () => {
    mocks.fetchRota.mockReturnValue(new Promise(() => {}));
    mocks.fetchShiftTemplates.mockResolvedValue([]);
    mocks.fetchShiftSwapRequests.mockResolvedValue([]);
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <MemoryRouter>
        <QueryClientProvider client={client}>
          <SchedulingConsole />
        </QueryClientProvider>
      </MemoryRouter>,
    );

    const loading = screen.getByLabelText("Loading the board");
    expect(loading).toHaveAttribute("aria-busy", "true");
    const hidden = { hidden: true } as const;
    expect(
      within(loading).getByRole("region", { name: "Needs a look", ...hidden }),
    ).toBeInTheDocument();
    const board = within(loading).getByRole("table", { name: "Rota", ...hidden });
    const [heading] = within(board).getAllByRole("row", hidden);
    expect(within(heading!).getAllByRole("columnheader", hidden)).toHaveLength(9);
  });

  it("heads the page with the board's size, today's cover, conflicts and swaps", async () => {
    renderConsole({
      rota: rota([
        row("Ada Byrne"),
        row(
          "Ben Cole",
          week((i) => (i === 1 ? { state: "TimeOff", isConflict: true } : {})),
        ),
        row(
          "Cal Diaz",
          week((i) => (i === 1 ? { state: "Leave", scheduled: false } : {})),
        ),
      ]),
      swaps: [swap(), swap({ id: "swp_2", status: "Proposed" })],
    });

    expect(await screen.findByLabelText("On the board", { selector: "span" })).toHaveTextContent(
      "3",
    );
    expect(await screen.findByLabelText("Cover today", { selector: "span" })).toHaveTextContent(
      "1",
    );
    expect(screen.getByText("of 2")).toBeInTheDocument();
    expect(screen.getByText("1 on time off · 1 on leave · 1 in conflict")).toBeInTheDocument();
    expect(screen.getByLabelText("Conflicts", { selector: "span" })).toHaveTextContent("1");
    expect(
      await screen.findByLabelText("Swaps waiting on you", { selector: "span" }),
    ).toHaveTextContent("1");
    expect(screen.getByText("1 more still waiting on a colleague")).toBeInTheDocument();
    const bar = screen.getByRole("img", { name: /Person-days on the board/ });
    expect(bar).toHaveAccessibleName("Person-days on the board: Working 19, Off 0, Away 2");
  });

  it("says when today is outside the weeks shown", async () => {
    vi.spyOn(Date, "now").mockReturnValue((WEEK_START + 9 * DAY) * 1000);
    renderConsole({ rota: rota([row("Ada Byrne")]) });

    expect(await screen.findByText("Today is outside the weeks shown")).toBeInTheDocument();
  });

  // The board shows a conflict as a ringed cell; a manager working down a list
  // needs it as a line with a name and a date.
  it("lists conflicts and people with no shift, and jumps to the swaps to decide", async () => {
    const user = userEvent.setup();
    renderConsole({
      rota: rota([
        row(
          "Ada Byrne",
          week((i) => (i === 2 ? { state: "TimeOff", isConflict: true } : {})),
        ),
        row(
          "Ben Cole",
          week(() => ({ state: "Off", scheduled: false })),
          { shiftName: null },
        ),
        row("Cal Diaz"),
      ]),
      swaps: [swap()],
    });

    const panel = await screen.findByRole("region", { name: "Needs a look" });
    const items = await within(panel).findAllByRole("link");
    expect(items).toHaveLength(2);
    expect(within(items[0]).getByText("Ada Byrne")).toBeInTheDocument();
    expect(within(items[0]).getByText(/rostered while time off/)).toBeInTheDocument();
    expect(within(items[0]).getByText("Conflict")).toBeInTheDocument();
    expect(within(items[1]).getByText("Ben Cole")).toBeInTheDocument();
    expect(within(items[1]).getByText("No shift")).toBeInTheDocument();
    expect(within(panel).queryByText("Cal Diaz")).not.toBeInTheDocument();

    await user.click(within(panel).getByRole("button", { name: "1 swap to decide" }));
    expect(await screen.findByRole("tab", { name: /Swaps/, selected: true })).toBeInTheDocument();
    expect(
      await screen.findByRole("radiogroup", { name: "Which swaps to show" }),
    ).toBeInTheDocument();
  });

  it("says so when nothing on the board needs a look", async () => {
    renderConsole({ rota: rota([row("Ada Byrne")]) });

    const panel = await screen.findByRole("region", { name: "Needs a look" });
    expect(
      within(panel).getByText(/Everyone on the board has a shift they can work/),
    ).toBeInTheDocument();
  });

  it("draws a cover row across the board that counts who can work each day", async () => {
    renderConsole({
      rota: rota([
        row("Ada Byrne"),
        row(
          "Ben Cole",
          week((i) => (i === 4 ? { state: "TimeOff", isConflict: true } : {})),
        ),
      ]),
    });

    const cells = await screen.findAllByRole("img", { name: /rostered can work/ });
    expect(cells).toHaveLength(7);
    expect(cells[0]).toHaveAccessibleName(/: 2 of 2 rostered can work/);
    expect(cells[4]).toHaveAccessibleName(/: 1 of 2 rostered can work/);
  });

  // A search narrows the rows, not the cover: Thursday is thin or not
  // regardless of who the reader is looking for.
  it("narrows the board to a searched name and keeps the cover row whole", async () => {
    const user = userEvent.setup();
    renderConsole({ rota: rota([row("Ada Byrne"), row("Ben Cole"), row("Cal Diaz")]) });

    await screen.findByText("Cal Diaz");
    await user.type(screen.getByLabelText("Find on the board"), "cal");

    await waitFor(() => expect(screen.queryByText("Ada Byrne")).not.toBeInTheDocument());
    expect(screen.getByText("Cal Diaz")).toBeInTheDocument();
    expect(screen.getByText("1 of 3")).toBeInTheDocument();
    const cells = screen.getAllByRole("img", { name: /rostered can work/ });
    expect(cells[0]).toHaveAccessibleName(/: 3 of 3 rostered can work/);
  });

  it("sums the patterns and prices each one in hours a week", async () => {
    const user = userEvent.setup();
    renderConsole({
      rota: rota([row("Ada Byrne")]),
      templates: [
        template(),
        template({
          id: "sht_2",
          code: "WKND",
          name: "Weekend",
          daysOfWeek: "1000001",
          durationMinutes: 600,
          activeAssignmentCount: 1,
        }),
        template({
          id: "sht_3",
          code: "OLD",
          name: "Old",
          status: "Inactive",
          activeAssignmentCount: 0,
        }),
      ],
    });

    const tab = await screen.findByRole("tab", { name: /Shifts/ });
    await waitFor(() => expect(tab).toHaveTextContent("2"));
    await user.click(tab);

    expect(await screen.findByLabelText("Pattern summary")).toHaveTextContent(
      "2 active · 4 people on a pattern · 30h a week on average · 1 retired",
    );
    const weekend = screen.getByRole("article", { name: "Weekend" });
    expect(within(weekend).getByText(/20h a week/)).toBeInTheDocument();
  });

  // A forty-driver roster is two screens of comfortable rows. Compact halves
  // that, and the choice is the reader's habit, so it has to survive a reload.
  it("switches the board to compact rows and remembers the choice", async () => {
    const user = userEvent.setup();
    renderConsole({ rota: rota([row("Ada Byrne")]) });

    const board = (await screen.findByText("Ada Byrne")).closest("[data-density]") as HTMLElement;
    expect(board).toHaveAttribute("data-density", "comfortable");
    expect(board).toHaveAttribute("data-cell-mode", "detail");
    expect(within(board).getAllByText("08:00")).toHaveLength(7);
    expect(within(board).getAllByText("8h")).toHaveLength(7);
    expect(within(board).getByText("AB")).toBeInTheDocument();

    const control = screen.getByRole("radiogroup", { name: "Board density" });
    await user.click(within(control).getByRole("radio", { name: "Compact" }));

    expect(board).toHaveAttribute("data-density", "compact");
    expect(board).toHaveAttribute("data-cell-mode", "time");
    expect(within(board).getAllByText("08:00")).toHaveLength(7);
    expect(within(board).queryByText("8h")).not.toBeInTheDocument();
    expect(within(board).queryByText("AB")).not.toBeInTheDocument();
    expect(window.localStorage.getItem("scheduling.rota-density")).toBe(JSON.stringify("compact"));
  });

  it("opens in the density last chosen", async () => {
    window.localStorage.setItem("scheduling.rota-density", JSON.stringify("compact"));
    renderConsole({ rota: rota([row("Ada Byrne")]) });

    const board = (await screen.findByText("Ada Byrne")).closest("[data-density]") as HTMLElement;
    expect(board).toHaveAttribute("data-density", "compact");
    expect(
      within(screen.getByRole("radiogroup", { name: "Board density" })).getByRole("radio", {
        name: "Compact",
      }),
    ).toBeChecked();
  });

  it("falls back to comfortable when the stored density is not one it knows", async () => {
    window.localStorage.setItem("scheduling.rota-density", JSON.stringify("dense"));
    renderConsole({ rota: rota([row("Ada Byrne")]) });

    const board = (await screen.findByText("Ada Byrne")).closest("[data-density]") as HTMLElement;
    expect(board).toHaveAttribute("data-density", "comfortable");
  });

  // Twenty-eight columns cannot carry a clock time; the tint and the tooltip
  // carry the day instead.
  it("draws blocks instead of times when more than one week is on the board", async () => {
    const user = userEvent.setup();
    const twoWeeks = Array.from({ length: 14 }, (_, index) => day(index));
    mocks.fetchRota.mockImplementation(async (filter: { weeks?: number | null }) =>
      filter.weeks === 2
        ? { ...rota([row("Ada Byrne", twoWeeks)]), weeks: 2, weekEnd: WEEK_START + 14 * DAY }
        : rota([row("Ada Byrne")]),
    );
    mocks.fetchShiftTemplates.mockResolvedValue([]);
    mocks.fetchShiftSwapRequests.mockResolvedValue([]);
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    render(
      <MemoryRouter>
        <QueryClientProvider client={client}>
          <SchedulingConsole />
        </QueryClientProvider>
      </MemoryRouter>,
    );

    await screen.findByText("Ada Byrne");
    await user.click(
      within(screen.getByRole("radiogroup", { name: "Weeks on the board" })).getByRole("radio", {
        name: "2 weeks",
      }),
    );

    await waitFor(() => {
      const board = screen.getByText("Ada Byrne").closest("[data-density]") as HTMLElement;
      expect(board).toHaveAttribute("data-cell-mode", "block");
    });
    const board = screen.getByText("Ada Byrne").closest("[data-density]") as HTMLElement;
    expect(within(board).queryByText("08:00")).not.toBeInTheDocument();
    expect(board.querySelectorAll("[data-cell='block']")).toHaveLength(14);
    expect(within(board).getAllByRole("img", { name: /rostered can work/ })).toHaveLength(14);
  });
});
