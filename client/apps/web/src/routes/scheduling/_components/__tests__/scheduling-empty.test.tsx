import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
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
vi.mock("../shift-template-dialog", () => ({
  ShiftTemplateDialog: ({ open }: { open: boolean }) =>
    open ? <div data-testid="shift-dialog" /> : null,
}));
vi.mock("@number-flow/react", () => ({
  default: ({ value, className, ...rest }: { value: number; className?: string }) => (
    <span className={className} {...rest}>
      {value}
    </span>
  ),
}));

const DAY = 86_400;
const WEEK_START = Date.UTC(2026, 8, 6) / 1000;

function day(index: number) {
  return {
    date: WEEK_START + index * DAY,
    state: "Scheduled",
    scheduled: true,
    startMinute: 480,
    durationMinutes: 480,
    preference: null,
    assignmentCount: 0,
    isConflict: false,
  };
}

function row(name: string) {
  const days = Array.from({ length: 7 }, (_, index) => day(index));
  return {
    workerId: `wrk_${name.split(" ")[0].toLowerCase()}`,
    name,
    fleetCode: "SOUTH",
    fleetColor: "#2563eb",
    shiftCode: "DAYS",
    shiftName: "Days",
    shiftColor: null,
    days,
    scheduledDays: 7,
    conflicts: 0,
  };
}

function rota(rows: ReturnType<typeof row>[]) {
  return {
    weekStart: WEEK_START,
    weekEnd: WEEK_START + 7 * DAY,
    weeks: 1,
    rows,
    scheduledDays: rows.length * 7,
    conflicts: 0,
  };
}

function renderConsole(fixtures: { rows?: ReturnType<typeof row>[]; templates?: unknown[] } = {}) {
  mocks.fetchRota.mockResolvedValue(rota(fixtures.rows ?? []));
  mocks.fetchShiftTemplates.mockResolvedValue(fixtures.templates ?? []);
  mocks.fetchShiftSwapRequests.mockResolvedValue([]);
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <SchedulingConsole />
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("scheduling empty states", () => {
  it("draws the board it will become when nobody is on it", async () => {
    renderConsole();

    expect(await screen.findByRole("heading", { name: "Nobody on the board" })).toBeInTheDocument();
    expect(screen.getByText(/Schedule tab/)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Clear filters" })).not.toBeInTheDocument();
  });

  // The team toggle and the fleet picker can empty the week on their own;
  // the sketch's button drops both rather than telling the reader to widen.
  it("offers to clear the team filter when it is what emptied the board", async () => {
    const user = userEvent.setup();
    renderConsole();

    await screen.findByRole("heading", { name: "Nobody on the board" });
    const team = screen.getByRole("button", { name: "My team" });
    await user.click(team);
    expect(team).toHaveAttribute("aria-pressed", "true");

    await user.click(await screen.findByRole("button", { name: "Clear filters" }));
    expect(team).toHaveAttribute("aria-pressed", "false");
  });

  it("offers to clear a search that hides everyone on the board", async () => {
    const user = userEvent.setup();
    renderConsole({ rows: [row("Ada Byrne")] });

    expect(await screen.findByText("Ada Byrne")).toBeInTheDocument();
    await user.type(screen.getByLabelText("Find on the board"), "zzz");
    expect(await screen.findByRole("heading", { name: "Nobody matches that" })).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Clear filters" }));
    expect(await screen.findByText("Ada Byrne")).toBeInTheDocument();
  });

  it("offers the first shift pattern from the shifts tab", async () => {
    const user = userEvent.setup();
    renderConsole();

    await user.click(await screen.findByRole("tab", { name: /Shifts/ }));
    expect(await screen.findByRole("heading", { name: "No shifts yet" })).toBeInTheDocument();
    const adds = screen.getAllByRole("button", { name: "Add a shift" });
    await user.click(adds[adds.length - 1]);
    expect(screen.getByTestId("shift-dialog")).toBeInTheDocument();
  });

  it("draws the swap queue it will become when nothing is waiting", async () => {
    const user = userEvent.setup();
    renderConsole();

    await user.click(await screen.findByRole("tab", { name: /Swaps/ }));
    expect(await screen.findByRole("heading", { name: "Nothing waiting" })).toBeInTheDocument();
  });
});
