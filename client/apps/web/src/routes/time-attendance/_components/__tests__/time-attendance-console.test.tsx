import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useController, type Control, type FieldValues, type Path } from "react-hook-form";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import TimeAttendanceConsole from "../time-attendance-console";

const mocks = vi.hoisted(() => ({
  fetchTimesheets: vi.fn(),
  fetchTimesheet: vi.fn(),
  fetchOpenTimeEntries: vi.fn(),
  fetchOpenTimeEntry: vi.fn(),
  fetchTimeClockEntries: vi.fn(),
  fetchPayrollExports: vi.fn(),
  fetchPayrollExportRows: vi.fn(),
  clockIn: vi.fn(),
  clockOut: vi.fn(),
  recordTimeEntry: vi.fn(),
  deleteTimeEntry: vi.fn(),
  transitionTimesheet: vi.fn(),
  generatePayrollExport: vi.fn(),
  voidPayrollExport: vi.fn(),
}));

vi.mock("@/lib/graphql/timesheet", () => ({
  ...mocks,
  TIMESHEETS_KEY: "timesheets",
  TIMESHEET_KEY: "timesheet",
  TIME_ENTRIES_KEY: "time-clock-entries",
  OPEN_ENTRY_KEY: "open-time-clock-entry",
  OPEN_ENTRIES_KEY: "open-time-clock-entries",
  PAYROLL_EXPORTS_KEY: "payroll-exports",
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));

vi.mock("@/hooks/use-api-mutation", () => ({
  useApiMutation: ({
    mutationFn,
    onSuccess,
  }: {
    mutationFn: (values: unknown) => Promise<unknown>;
    onSuccess?: (data: unknown) => void;
  }) => ({
    mutateAsync: async (values: unknown) => {
      const data = await mutationFn(values);
      onSuccess?.(data);
      return data;
    },
    isPending: false,
  }),
}));

// The real picker is an async autocomplete; the clock only needs a field the
// form can read and the board can write.
vi.mock("@/components/autocomplete-fields", () => ({
  WorkerAutocompleteField: <T extends FieldValues>({
    control,
    name,
    label,
    placeholder,
  }: {
    control: Control<T>;
    name: Path<T>;
    label?: string;
    placeholder?: string;
  }) => {
    const { field } = useController({ control, name });
    return (
      <input
        aria-label={label ?? placeholder}
        value={(field.value as string) ?? ""}
        onChange={(event) => field.onChange(event.target.value)}
      />
    );
  },
}));

vi.mock("../record-entry-dialog", () => ({ RecordEntryDialog: () => null }));
vi.mock("../remove-entry-dialog", () => ({ RemoveEntryDialog: () => null }));
vi.mock("../void-export-dialog", () => ({ VoidExportDialog: () => null }));

vi.mock("@number-flow/react", () => ({
  default: ({ value, className, ...rest }: { value: number; className?: string }) => (
    <span className={className} {...rest}>
      {value}
    </span>
  ),
}));

const HOUR = 3600;
const DAY = 86_400;
// A Monday afternoon, so "this week" started the day before.
const NOW = Date.UTC(2026, 8, 7, 15, 30) / 1000;
const WEEK_START = Date.UTC(2026, 8, 6) / 1000;

const ADA = { id: "wrk_ada", firstName: "Ada", lastName: "Byrne", type: "Employee" };
const BEN = { id: "wrk_ben", firstName: "Ben", lastName: "Cole", type: "Employee" };

function sheet(over: Record<string, unknown> = {}) {
  return {
    id: "tsh_1",
    workerId: ADA.id,
    status: "Submitted",
    periodStart: WEEK_START - 7 * DAY,
    periodEnd: WEEK_START,
    regularMinutes: 2400,
    overtimeMinutes: 0,
    paidLeaveMinutes: 0,
    totalMinutes: 2400,
    entryCount: 5,
    overtimeThresholdMinutes: 2400,
    submittedAt: WEEK_START,
    approvedAt: null,
    decisionNote: null,
    payrollExportId: null,
    version: 1,
    worker: ADA,
    ...over,
  };
}

function openEntry(over: Record<string, unknown> = {}) {
  return {
    id: "tce_open_1",
    workerId: ADA.id,
    source: "Clock",
    clockedInAt: NOW - 6 * HOUR,
    breakMinutes: 0,
    note: null,
    worker: {
      ...ADA,
      profilePicUrl: "",
      fleetCode: { id: "fc_1", code: "SOUTH", color: "#2563eb" },
    },
    ...over,
  };
}

type Fixtures = {
  awaiting?: unknown[];
  week?: unknown[];
  unpaid?: unknown[];
  active?: unknown[];
  paid?: unknown[];
  open?: unknown[];
  openFor?: Record<string, unknown>;
  entriesFor?: Record<string, unknown[]>;
  detail?: Record<string, unknown>;
};

function renderConsole(fixtures: Fixtures = {}) {
  mocks.fetchTimesheets.mockImplementation(async (filter: Record<string, unknown>) => {
    const statuses = (filter.statuses as string[] | undefined) ?? [];
    if (filter.unexportedOnly) return fixtures.unpaid ?? [];
    if (filter.from && filter.workerId) {
      return (fixtures.week ?? []).filter(
        (row) => (row as { workerId: string }).workerId === filter.workerId,
      );
    }
    if (filter.from) return fixtures.week ?? [];
    if (statuses.length === 1 && statuses[0] === "Submitted") return fixtures.awaiting ?? [];
    if (statuses.length === 1 && statuses[0] === "Locked") return fixtures.paid ?? [];
    return fixtures.active ?? [];
  });
  mocks.fetchOpenTimeEntries.mockResolvedValue(fixtures.open ?? []);
  mocks.fetchOpenTimeEntry.mockImplementation(
    async (workerId: string) => fixtures.openFor?.[workerId] ?? null,
  );
  mocks.fetchTimeClockEntries.mockImplementation(
    async (args: { workerId: string }) => fixtures.entriesFor?.[args.workerId] ?? [],
  );
  mocks.fetchPayrollExports.mockResolvedValue([]);
  mocks.fetchTimesheet.mockImplementation(async (id: string) =>
    fixtures.detail && (fixtures.detail as { id: string }).id === id ? fixtures.detail : null,
  );
  mocks.clockOut.mockResolvedValue({ id: "tce_open_1", clockedOutAt: NOW, paidMinutes: 360 });

  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <TimeAttendanceConsole />
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  vi.spyOn(Date, "now").mockReturnValue(NOW * 1000);
});

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
  vi.clearAllMocks();
});

describe("TimeAttendanceConsole", () => {
  it("heads the page with what is waiting, who is punched in, and the week so far", async () => {
    renderConsole({
      awaiting: [
        sheet(),
        sheet({
          id: "tsh_2",
          workerId: BEN.id,
          worker: BEN,
          totalMinutes: 2700,
          regularMinutes: 2400,
          overtimeMinutes: 300,
        }),
      ],
      week: [
        sheet({ id: "tsh_w1", status: "Open", periodStart: WEEK_START, regularMinutes: 1200 }),
        sheet({
          id: "tsh_w2",
          status: "Open",
          workerId: BEN.id,
          periodStart: WEEK_START,
          regularMinutes: 1200,
          overtimeMinutes: 60,
          paidLeaveMinutes: 480,
        }),
      ],
      unpaid: [sheet({ id: "tsh_u", status: "Approved" })],
      open: [
        openEntry(),
        openEntry({
          id: "tce_open_2",
          workerId: BEN.id,
          clockedInAt: NOW - HOUR,
          worker: { ...BEN, profilePicUrl: "", fleetCode: null },
        }),
      ],
    });

    expect(
      await screen.findByLabelText("Awaiting approval", { selector: "span" }),
    ).toHaveTextContent("2");
    expect(screen.getByText("85h across 2 people")).toBeInTheDocument();
    expect(
      await screen.findByLabelText("On the clock now", { selector: "span" }),
    ).toHaveTextContent("2");
    expect(screen.getByText("Longest running: Ada Byrne, 6h")).toBeInTheDocument();
    const week = await screen.findByRole("img", { name: /This week's hours/ });
    expect(week).toHaveAccessibleName("This week's hours: Regular 40h, Overtime 1h, Leave 8h");
    expect(
      await screen.findByLabelText("Approved, not paid", { selector: "span" }),
    ).toHaveTextContent("1");
    expect(screen.getByText("40h waiting for a run")).toBeInTheDocument();
  });

  // The board answers the question a manager walks in with, before a worker
  // is picked: who is still here, and who has been here longest.
  it("lists everyone on the clock, longest first, and punches them out from the row", async () => {
    const user = userEvent.setup();
    renderConsole({
      open: [
        openEntry({
          id: "tce_open_2",
          workerId: BEN.id,
          clockedInAt: NOW - HOUR,
          worker: { ...BEN, profilePicUrl: "", fleetCode: null },
        }),
        openEntry(),
      ],
    });

    const board = await screen.findByRole("region", { name: "On the clock now" });
    const rows = await within(board).findAllByRole("listitem");
    expect(within(rows[0]).getByText("Ada Byrne")).toBeInTheDocument();
    expect(within(rows[0]).getByLabelText("Ada Byrne running time")).toHaveTextContent("6h");
    expect(within(rows[0]).getByText("SOUTH")).toBeInTheDocument();
    expect(within(rows[1]).getByText("Ben Cole")).toBeInTheDocument();
    expect(within(rows[1]).getByLabelText("Ben Cole running time")).toHaveTextContent("1h");

    await user.click(within(board).getByRole("button", { name: "Clock out Ben Cole" }));
    expect(mocks.clockOut).toHaveBeenCalledWith({ workerId: BEN.id });
  });

  it("flags a punch that has run past twelve hours", async () => {
    renderConsole({ open: [openEntry({ clockedInAt: NOW - 13 * HOUR })] });

    const board = await screen.findByRole("region", { name: "On the clock now" });
    expect(await within(board).findByText("1 past 12h")).toBeInTheDocument();
  });

  it("opens somebody's clock straight from the board", async () => {
    const user = userEvent.setup();
    renderConsole({
      open: [openEntry()],
      openFor: {
        [ADA.id]: {
          id: "tce_open_1",
          workerId: ADA.id,
          clockedInAt: NOW - 6 * HOUR,
          source: "Clock",
          note: null,
        },
      },
    });

    const board = await screen.findByRole("region", { name: "On the clock now" });
    expect(screen.queryByRole("region", { name: "Clock" })).not.toBeInTheDocument();

    await user.click(
      await within(board).findByRole("button", { name: "Open the clock for Ada Byrne" }),
    );

    const clock = await screen.findByRole("region", { name: "Clock" });
    expect(within(clock).getByText("On the clock")).toBeInTheDocument();
    expect(within(clock).getByText("6h")).toBeInTheDocument();
    expect(screen.getByLabelText("Worker")).toHaveValue(ADA.id);
    expect(mocks.fetchOpenTimeEntry).toHaveBeenCalledWith(ADA.id, expect.anything());
  });

  it("groups the punch history by day with a subtotal per day", async () => {
    const user = userEvent.setup();
    const sunday = WEEK_START + 8 * HOUR;
    const monday = WEEK_START + DAY + 8 * HOUR;
    renderConsole({
      entriesFor: {
        [ADA.id]: [
          {
            id: "e1",
            workerId: ADA.id,
            timesheetId: null,
            source: "Clock",
            clockedInAt: sunday,
            clockedOutAt: sunday + 8 * HOUR,
            breakMinutes: 30,
            paidMinutes: 450,
            note: null,
            editReason: null,
            version: 1,
          },
          {
            id: "e2",
            workerId: ADA.id,
            timesheetId: null,
            source: "Manual",
            clockedInAt: sunday + 10 * HOUR,
            clockedOutAt: sunday + 12 * HOUR,
            breakMinutes: 0,
            paidMinutes: 120,
            note: null,
            editReason: "Forgot to punch",
            version: 1,
          },
          {
            id: "e3",
            workerId: ADA.id,
            timesheetId: null,
            source: "Clock",
            clockedInAt: monday,
            clockedOutAt: monday + 3 * HOUR,
            breakMinutes: 0,
            paidMinutes: 180,
            note: null,
            editReason: null,
            version: 1,
          },
        ],
      },
    });

    await screen.findByRole("region", { name: "On the clock now" });
    await user.type(screen.getByLabelText("Worker"), ADA.id);

    const history = await screen.findByRole("region", { name: "Last two weeks" });
    const days = await within(history).findAllByRole("region");
    expect(days).toHaveLength(2);
    expect(within(days[0]).getByLabelText("Day total")).toHaveTextContent("3h");
    expect(within(days[1]).getByLabelText("Day total")).toHaveTextContent("9h 30m · 30m break");
    expect(within(days[1]).getByText("Manual")).toBeInTheDocument();
    expect(within(history).getByText("3 punches · 12h 30m")).toBeInTheDocument();
  });

  // A week sent back used to vanish: it was neither "open" nor "awaiting". It
  // is somebody's unfinished week and has to be findable.
  it("counts the queue segments and keeps sent-back weeks with the open ones", async () => {
    const user = userEvent.setup();
    renderConsole({
      awaiting: [sheet(), sheet({ id: "tsh_2", workerId: BEN.id, worker: BEN })],
      active: [
        sheet(),
        sheet({ id: "tsh_2", workerId: BEN.id, worker: BEN }),
        sheet({ id: "tsh_3", status: "Open", periodStart: WEEK_START }),
        sheet({
          id: "tsh_4",
          status: "Rejected",
          workerId: BEN.id,
          worker: BEN,
          decisionNote: "Tuesday is missing",
        }),
        sheet({ id: "tsh_5", status: "Approved" }),
      ],
    });

    const tab = await screen.findByRole("tab", { name: /Timesheets/ });
    await waitFor(() => expect(tab).toHaveTextContent("2"));
    await user.click(tab);

    const control = await screen.findByRole("radiogroup", { name: "Timesheet status" });
    expect(within(control).getByRole("radio", { name: /Awaiting approval/ })).toHaveTextContent(
      "2",
    );
    expect(within(control).getByRole("radio", { name: /^Open/ })).toHaveTextContent("2");
    expect(within(control).getByRole("radio", { name: /Approved/ })).toHaveTextContent("1");

    await user.click(within(control).getByRole("radio", { name: /^Open/ }));
    const list = await screen.findByRole("list", { name: "Open" });
    const rows = within(list).getAllByRole("listitem");
    expect(rows).toHaveLength(2);
    expect(within(list).getByText("Sent back")).toBeInTheDocument();
    expect(within(list).getByText(/Tuesday is missing/)).toBeInTheDocument();
  });

  // Somebody watching their week wants the number that matters: how much is
  // left before it tips into overtime.
  it("meters the picked worker's week against their overtime threshold", async () => {
    const user = userEvent.setup();
    renderConsole({
      week: [
        sheet({
          id: "tsh_ada_week",
          status: "Open",
          periodStart: WEEK_START,
          periodEnd: WEEK_START + 7 * DAY,
          regularMinutes: 1935,
          totalMinutes: 1935,
          submittedAt: null,
        }),
      ],
    });

    await screen.findByRole("region", { name: "On the clock now" });
    await user.type(screen.getByLabelText("Worker"), ADA.id);

    const clock = await screen.findByRole("region", { name: "Clock" });
    const meter = await within(clock).findByRole("meter", { name: "This week" });
    expect(meter).toHaveAttribute("aria-valuenow", "81");
    expect(within(clock).getByText(/7h 45m before overtime/)).toBeInTheDocument();
  });

  it("draws each day's punches on a 24-hour track", async () => {
    const user = userEvent.setup();
    const sunday = WEEK_START + 8 * HOUR;
    renderConsole({
      entriesFor: {
        [ADA.id]: [
          {
            id: "e1",
            workerId: ADA.id,
            timesheetId: null,
            source: "Clock",
            clockedInAt: sunday,
            clockedOutAt: sunday + 4 * HOUR,
            breakMinutes: 0,
            paidMinutes: 240,
            note: null,
            editReason: null,
            version: 1,
          },
          {
            id: "e2",
            workerId: ADA.id,
            timesheetId: null,
            source: "Clock",
            clockedInAt: sunday + 5 * HOUR,
            clockedOutAt: sunday + 8 * HOUR,
            breakMinutes: 0,
            paidMinutes: 180,
            note: null,
            editReason: null,
            version: 1,
          },
        ],
      },
    });

    await screen.findByRole("region", { name: "On the clock now" });
    await user.type(screen.getByLabelText("Worker"), ADA.id);

    const history = await screen.findByRole("region", { name: "Last two weeks" });
    const track = await within(history).findByRole("img", { name: /^Punches on/ });
    expect(track.querySelectorAll("[data-slot='punch-span']")).toHaveLength(2);
  });

  // An approver triages by age: the week that has waited longest is the one
  // somebody is chasing.
  it("says how long each handed-over week has been waiting", async () => {
    const user = userEvent.setup();
    renderConsole({
      awaiting: [sheet()],
      active: [
        sheet({ submittedAt: WEEK_START }),
        sheet({ id: "tsh_2", workerId: BEN.id, worker: BEN, submittedAt: NOW - 4 * DAY - HOUR }),
      ],
    });

    await user.click(await screen.findByRole("tab", { name: /Timesheets/ }));
    const list = await screen.findByRole("list", { name: "Awaiting approval" });
    const rows = within(list).getAllByRole("listitem");
    expect(within(rows[0]).getByText(/Waiting 1 day/)).toBeInTheDocument();
    expect(within(rows[1]).getByText(/Waiting 4 days/)).toBeInTheDocument();
  });

  it("lays a week out as a time card when it is opened", async () => {
    const user = userEvent.setup();
    const start = WEEK_START - 7 * DAY;
    renderConsole({
      awaiting: [sheet()],
      active: [sheet()],
      detail: {
        ...sheet(),
        entries: [
          {
            id: "d1",
            workerId: ADA.id,
            source: "Clock",
            clockedInAt: start + 8 * HOUR,
            clockedOutAt: start + 16 * HOUR,
            breakMinutes: 30,
            paidMinutes: 450,
            note: null,
            editReason: null,
            version: 1,
          },
          {
            id: "d2",
            workerId: ADA.id,
            source: "Manual",
            clockedInAt: start + DAY + 9 * HOUR,
            clockedOutAt: start + DAY + 12 * HOUR,
            breakMinutes: 0,
            paidMinutes: 180,
            note: null,
            editReason: "Forgot to punch",
            version: 1,
          },
        ],
      },
    });

    await user.click(await screen.findByRole("tab", { name: /Timesheets/ }));
    const list = await screen.findByRole("list", { name: "Awaiting approval" });
    await user.click(within(list).getByRole("button", { name: /Ada Byrne/ }));

    const card = await screen.findByRole("group", { name: "Time card" });
    const days = within(card).getAllByRole("group");
    expect(days).toHaveLength(7);
    expect(days[0]).toHaveAccessibleName(/Sun/);
    expect(days[0]).toHaveTextContent("7h 30m");
    expect(days[1]).toHaveTextContent("3h");
    expect(days[2]).toHaveTextContent("—");
  });
});
