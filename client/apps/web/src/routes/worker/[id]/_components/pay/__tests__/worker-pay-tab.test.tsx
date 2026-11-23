import { WorkPaneHost, WorkPaneProvider } from "@/components/work-pane";
import { CLOSED_WORKER_OPEN_STATE } from "@/routes/worker/use-worker-state";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { afterEach, describe, expect, it, vi } from "vitest";
import WorkerPayTab from "../pay-profile/worker-pay-tab";

const mocks = vi.hoisted(() => ({
  fetchEffectiveWorkerPayAssignment: vi.fn(),
  fetchWorkerPayAssignments: vi.fn(),
  fetchWorkerEarningsSummary: vi.fn(),
  fetchCurrentSettlementPeriod: vi.fn(),
  endWorkerPayAssignment: vi.fn(),
}));
const granted = vi.hoisted(() => ({ keys: new Set<string>() }));

vi.mock("@/lib/graphql/driver-settlement", () => ({
  ...mocks,
  assignPayProfileToWorker: vi.fn(),
  fetchPayProfileDetail: vi.fn(),
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: (resource: string, operation: number) => ({
    allowed: granted.keys.has(`${resource}:${operation}`),
    isLoading: false,
  }),
}));

vi.mock("@trenova/shared/components/link", () => ({
  Link: ({ to, children }: { to: string; children: React.ReactNode }) => (
    <a href={to}>{children}</a>
  ),
}));

const assignment = {
  id: "wpa_1",
  workerId: "wrk_1",
  payProfileId: "pp_1",
  effectiveFrom: 1_751_328_000,
  effectiveTo: null,
  splitPercent: "100",
  notes: null,
  rateOverrides: [{ componentId: "c_loaded", rate: "0.62" }],
  payProfile: {
    id: "pp_1",
    name: "OTR mileage, company driver",
    classification: "CompanyDriver",
    guaranteedPeriodMinimumMinor: 120_000,
    components: [
      {
        id: "c_loaded",
        kind: "Linehaul",
        method: "PerLoadedMile",
        description: "Loaded miles",
        rate: "0.60",
        revenueBasis: null,
        bands: [],
        isActive: true,
      },
      {
        id: "c_empty",
        kind: "Linehaul",
        method: "PerEmptyMile",
        description: "Empty miles",
        rate: "0.45",
        revenueBasis: null,
        bands: [],
        isActive: true,
      },
    ],
  },
};

const history = [
  {
    id: "wpa_0",
    workerId: "wrk_1",
    payProfileId: "pp_0",
    effectiveFrom: 1_662_940_800,
    effectiveTo: 1_709_510_400,
    splitPercent: "100",
    rateOverrides: [],
    notes: null,
    version: 1,
    createdAt: 1,
    payProfile: {
      id: "pp_0",
      name: "Local hourly",
      classification: "CompanyDriver",
      components: [],
    },
  },
  {
    ...assignment,
    version: 1,
    createdAt: 2,
    payProfile: {
      id: "pp_1",
      name: "OTR mileage",
      classification: "CompanyDriver",
      components: [],
    },
  },
];

function connection<Row extends { id: string }>(rows: Row[]) {
  return {
    edges: rows.map((node) => ({ cursor: node.id, node })),
    pageInfo: { hasNextPage: false, endCursor: null },
  };
}

function grant(...pairs: [string, number][]) {
  granted.keys = new Set(pairs.map(([resource, operation]) => `${resource}:${operation}`));
}

function renderTab() {
  const onOpen = vi.fn();
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  render(
    <QueryClientProvider client={client}>
      <WorkPaneProvider>
        <WorkPaneHost>
          <WorkerPayTab
            workerId="wrk_1"
            worker={{ id: "wrk_1", firstName: "Maria" } as never}
            open={CLOSED_WORKER_OPEN_STATE}
            onOpen={onOpen}
          />
        </WorkPaneHost>
      </WorkPaneProvider>
    </QueryClientProvider>,
  );
  return onOpen;
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("WorkerPayTab", () => {
  it("marks the driver's override against the profile rate and shows the guaranteed minimum", async () => {
    grant([Resource.DriverPayProfile, Operation.Read]);
    mocks.fetchEffectiveWorkerPayAssignment.mockResolvedValue(assignment);
    mocks.fetchWorkerPayAssignments.mockResolvedValue(connection([...history].reverse()));
    renderTab();

    const override = await screen.findByText("$0.62");
    expect(override.closest("td")?.hasAttribute("data-override")).toBe(true);
    const empty = screen.getAllByText("$0.45").map((cell) => cell.closest("td"));
    expect(empty.some((cell) => cell?.hasAttribute("data-override"))).toBe(false);
    expect(screen.getByText("$1,200.00")).toBeTruthy();
  });

  it("lists the assignment history newest first with the one in force marked current", async () => {
    grant([Resource.DriverPayProfile, Operation.Read]);
    mocks.fetchEffectiveWorkerPayAssignment.mockResolvedValue(assignment);
    mocks.fetchWorkerPayAssignments.mockResolvedValue(connection([...history].reverse()));
    renderTab();

    const table = (await screen.findByText("Local hourly")).closest("tbody") as HTMLElement;
    const rows = within(table).getAllByRole("row");
    expect(within(rows[0]).getByText("OTR mileage")).toBeTruthy();
    expect(within(rows[0]).getByText("Current")).toBeTruthy();
    expect(within(rows[0]).getByText("Still in effect")).toBeTruthy();
    expect(within(rows[1]).getByText("Local hourly")).toBeTruthy();
  });

  it("offers no change or end to a reader who may not assign pay profiles, and reads no earnings without settlement read", async () => {
    grant([Resource.DriverPayProfile, Operation.Read]);
    mocks.fetchEffectiveWorkerPayAssignment.mockResolvedValue(assignment);
    mocks.fetchWorkerPayAssignments.mockResolvedValue(connection([]));
    renderTab();

    await screen.findByText("OTR mileage, company driver");
    expect(screen.queryByRole("button", { name: "Change profile" })).toBeNull();
    expect(screen.queryByText("Owed and held")).toBeNull();
    expect(mocks.fetchWorkerEarningsSummary).not.toHaveBeenCalled();
  });

  it("opens the change in the work pane", async () => {
    grant([Resource.DriverPayProfile, Operation.Assign]);
    mocks.fetchEffectiveWorkerPayAssignment.mockResolvedValue(assignment);
    mocks.fetchWorkerPayAssignments.mockResolvedValue(connection([]));
    const onOpen = renderTab();

    await userEvent.setup().click(await screen.findByRole("button", { name: "Change profile" }));
    expect(onOpen).toHaveBeenCalledWith({
      tab: "pay",
      view: "pay-profile",
      form: "assign-pay-profile",
    });
  });

  it("ends the assignment only after the confirmation", async () => {
    grant([Resource.DriverPayProfile, Operation.Assign]);
    mocks.fetchEffectiveWorkerPayAssignment.mockResolvedValue(assignment);
    mocks.fetchWorkerPayAssignments.mockResolvedValue(connection([]));
    mocks.endWorkerPayAssignment.mockResolvedValue({ id: "wpa_1" });
    renderTab();
    const user = userEvent.setup();

    await user.click(await screen.findByRole("button", { name: "Pay profile options" }));
    await user.click(await screen.findByRole("menuitem", { name: "End assignment" }));
    expect(mocks.endWorkerPayAssignment).not.toHaveBeenCalled();

    const dialog = await screen.findByRole("alertdialog");
    await user.click(within(dialog).getByRole("button", { name: "End assignment" }));
    await waitFor(() =>
      expect(mocks.endWorkerPayAssignment).toHaveBeenCalledWith({
        assignmentId: "wpa_1",
        endDate: expect.any(Number),
      }),
    );
  });

  it("shows what is owed and held to a reader of settlements", async () => {
    grant([Resource.DriverPayProfile, Operation.Read], [Resource.DriverSettlement, Operation.Read]);
    mocks.fetchEffectiveWorkerPayAssignment.mockResolvedValue(null);
    mocks.fetchWorkerPayAssignments.mockResolvedValue(connection([]));
    mocks.fetchWorkerEarningsSummary.mockResolvedValue({
      workerId: "wrk_1",
      accruedEventCount: 6,
      accruedGrossMinor: 218_460,
      outstandingAdvances: 30_000,
      escrowBalanceMinor: 0,
    });
    mocks.fetchCurrentSettlementPeriod.mockResolvedValue({
      periodStart: 1,
      periodEnd: 2,
      payDate: null,
    });
    renderTab();

    expect(await screen.findByText("$2,184.60")).toBeTruthy();
    expect(screen.getByText("6 pay events")).toBeTruthy();
    expect(screen.getByText("Taken from the next settlement")).toBeTruthy();
    expect(screen.getByText("No pay profile assigned")).toBeTruthy();
  });
});
