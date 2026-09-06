import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { PTOMonthCalendar } from "../pto-month-calendar";

const { fetchUpcomingWorkerPTO } = vi.hoisted(() => ({ fetchUpcomingWorkerPTO: vi.fn() }));
const dialogProps = vi.hoisted(() => ({ last: null as Record<string, unknown> | null }));
const permissionState = vi.hoisted(() => ({ create: true }));

vi.mock("@/lib/queries/worker", () => ({ fetchUpcomingWorkerPTO }));

vi.mock("@/lib/queries", () => ({
  queries: {
    worker: {
      listUpcomingPTO: { _def: ["worker", "list-upcoming-pto"] },
      ptoChartData: { _def: ["worker", "pto-chart-data"] },
    },
  },
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: (_resource: string, operation: number) => ({
    allowed: operation === 2 ? permissionState.create : true,
    isLoading: false,
  }),
}));

vi.mock("@trenova/shared/stores/auth-store", () => ({
  useAuthStore: (selector: (state: { user: { timezone: string } }) => unknown) =>
    selector({ user: { timezone: "America/New_York" } }),
}));

vi.mock("../../pto-form-dialog", () => ({
  PTOFormDialog: (props: Record<string, unknown>) => {
    dialogProps.last = props;
    return <div role="dialog">request-form</div>;
  },
}));

vi.mock("../../requested/upcoming-pto-content", () => ({
  UpcomingPTOContent: ({ pto }: { pto: { id: string } }) => <div>actions-for-{pto.id}</div>,
}));

const local = (y: number, m: number, d: number) => new Date(y, m, d).getTime() / 1000;
const march = { startDate: local(2026, 2, 1), endDate: local(2026, 2, 31) + 86_399 };

const rows = [
  {
    id: "wrkpto_a",
    status: "Approved",
    type: "Vacation",
    startDate: local(2026, 2, 5),
    endDate: local(2026, 2, 10),
    reason: "Beach",
    worker: { firstName: "Ada", lastName: "Lovelace" },
  },
  {
    id: "wrkpto_b",
    status: "Requested",
    type: "Sick",
    startDate: local(2026, 2, 6),
    endDate: local(2026, 2, 6),
    reason: "Dentist",
    worker: { firstName: "Grace", lastName: "Hopper" },
  },
  {
    id: "wrkpto_c",
    status: "Rejected",
    type: "Personal",
    startDate: local(2026, 2, 12),
    endDate: local(2026, 2, 12),
    reason: "Nope",
    worker: { firstName: "Linus", lastName: "T" },
  },
];

function renderCalendar(onMonthChange = vi.fn()) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={queryClient}>
      <PTOMonthCalendar filters={{ ...march, type: "Vacation" }} onMonthChange={onMonthChange} />
    </QueryClientProvider>,
  );
  return { onMonthChange };
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
  dialogProps.last = null;
  permissionState.create = true;
});

describe("PTOMonthCalendar", () => {
  it("queries the month covered by the filters with the active filter values", async () => {
    fetchUpcomingWorkerPTO.mockResolvedValue({ results: rows, count: 3, next: null, prev: null });
    renderCalendar();

    await screen.findAllByTestId("pto-span-wrkpto_a");
    expect(fetchUpcomingWorkerPTO).toHaveBeenCalledExactlyOnceWith(
      {
        filter: { limit: 200 },
        type: "Vacation",
        startDate: local(2026, 2, 1),
        endDate: local(2026, 2, 31) + 86_399,
        workerId: undefined,
        fleetCodeId: undefined,
        timezone: "America/New_York",
      },
      { signal: expect.any(AbortSignal) },
    );
  });

  it("renders approved and requested spans across week rows and hides decided-against ones", async () => {
    fetchUpcomingWorkerPTO.mockResolvedValue({ results: rows, count: 3, next: null, prev: null });
    renderCalendar();

    const spans = await screen.findAllByTestId("pto-span-wrkpto_a");
    expect(spans).toHaveLength(2);
    expect(screen.getByTestId("pto-span-wrkpto_b")).toBeInTheDocument();
    expect(screen.queryByTestId("pto-span-wrkpto_c")).not.toBeInTheDocument();
    expect(screen.getByTestId("whos-out-strip")).toBeInTheDocument();
  });

  it("moves months from the arrows and the keyboard", async () => {
    fetchUpcomingWorkerPTO.mockResolvedValue({ results: [], count: 0, next: null, prev: null });
    const { onMonthChange } = renderCalendar();

    fireEvent.click(await screen.findByRole("button", { name: "Next month" }));
    expect(onMonthChange).toHaveBeenLastCalledWith(local(2026, 3, 1), local(2026, 3, 30) + 86_399);

    fireEvent.keyDown(await screen.findByRole("grid"), { key: "ArrowLeft" });
    expect(onMonthChange).toHaveBeenLastCalledWith(local(2026, 1, 1), local(2026, 1, 28) + 86_399);
  });

  it("opens the request dialog prefilled with the dragged range", async () => {
    fetchUpcomingWorkerPTO.mockResolvedValue({ results: [], count: 0, next: null, prev: null });
    renderCalendar();

    fireEvent.mouseDown(await screen.findByTestId("pto-day-2026-03-09"));
    fireEvent.mouseEnter(screen.getByTestId("pto-day-2026-03-11"));
    expect(screen.getByTestId("pto-day-2026-03-10")).toHaveAttribute("data-selected", "true");
    fireEvent.mouseUp(window);

    await waitFor(() => expect(screen.getByRole("dialog")).toBeInTheDocument());
    expect(dialogProps.last?.defaultRange).toEqual({
      start: local(2026, 2, 9),
      end: local(2026, 2, 11),
    });
  });

  it("does not start a selection without create permission", async () => {
    fetchUpcomingWorkerPTO.mockResolvedValue({ results: [], count: 0, next: null, prev: null });
    permissionState.create = false;
    renderCalendar();

    fireEvent.mouseDown(await screen.findByTestId("pto-day-2026-03-09"));
    fireEvent.mouseUp(window);
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });
});
