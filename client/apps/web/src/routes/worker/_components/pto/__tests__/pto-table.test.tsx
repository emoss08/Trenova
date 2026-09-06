import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import PTODataTable from "../pto-table";

const { bulkWorkerPTOAction, toastInfo, toastSuccess, toastWarning } = vi.hoisted(() => ({
  bulkWorkerPTOAction: vi.fn(),
  toastInfo: vi.fn(),
  toastSuccess: vi.fn(),
  toastWarning: vi.fn(),
}));

const permissionState = vi.hoisted(() => ({
  approve: true,
  reject: true,
  cancel: true,
}));

const rows = [
  {
    id: "wrkpto_requested",
    status: "Requested",
    type: "Vacation",
    startDate: 1_767_225_600,
    endDate: 1_767_398_400,
    reason: "Family trip",
    days: "3.00",
    autoApproved: true,
    balanceAfterDays: "7.50",
    createdAt: 1_767_000_000,
    worker: { id: "wrk_1", firstName: "Ada", lastName: "Lovelace" },
  },
  {
    id: "wrkpto_rejected",
    status: "Rejected",
    type: "Sick",
    startDate: 1_767_225_600,
    endDate: 1_767_312_000,
    reason: "Flu",
    rejectionReason: "No coverage",
    rejector: { id: "usr_1", name: "Dana Dispatcher" },
    createdAt: 1_767_000_100,
    worker: { id: "wrk_2", firstName: "Grace", lastName: "Hopper" },
  },
];

const useDataTableQueryMock = vi.hoisted(() =>
  vi.fn(() => ({
    data: { results: [] as unknown[], count: 0 },
    isLoading: false,
    isError: false,
    error: null,
  })),
);

vi.mock("@/lib/graphql/worker-mutations", () => ({
  bulkWorkerPTOAction,
  rejectWorkerPTO: vi.fn(),
  cancelWorkerPTO: vi.fn(),
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermissions: () => ({
    canRead: true,
    canCreate: true,
    canUpdate: true,
    canExport: true,
    canImport: true,
    isLoading: false,
  }),
  usePermission: (_resource: string, operation: number) => {
    const bit = {
      256: permissionState.approve,
      512: permissionState.reject,
      32768: permissionState.cancel,
    };
    return { allowed: bit[operation as 256 | 512 | 32768] ?? true, isLoading: false };
  },
}));

vi.mock("@/hooks/data-table/use-data-table-query", () => ({
  useDataTableQuery: useDataTableQueryMock,
}));

vi.mock("@/lib/queries", () => ({
  queries: {
    tableConfiguration: {
      default: () => ({ queryKey: ["tableConfig-default"], queryFn: () => null }),
      all: () => ({
        queryKey: ["tableConfig-all"],
        queryFn: () => ({ results: [], count: 0 }),
      }),
    },
    worker: {
      listUpcomingPTO: { _def: ["worker", "list-upcoming-pto"] },
      ptoChartData: { _def: ["worker", "pto-chart-data"] },
    },
  },
}));

vi.mock("../pto-form-dialog", () => ({
  PTOFormDialog: ({ open, pto }: { open: boolean; pto?: { id?: string } | null }) =>
    open ? <div role="dialog">{pto?.id ? `edit:${pto.id}` : "request-pto-form"}</div> : null,
}));

vi.mock("@trenova/shared/hooks/use-debounce", () => ({
  useDebounce: <T,>(value: T): T => value,
}));

vi.mock("sonner", () => ({
  toast: { info: toastInfo, success: toastSuccess, warning: toastWarning, error: vi.fn() },
}));

function renderTable() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 }, mutations: { retry: false } },
  });
  const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries");
  render(
    <QueryClientProvider client={queryClient}>
      <NuqsTestingAdapter hasMemory resetUrlUpdateQueueOnMount={false}>
        <PTODataTable />
      </NuqsTestingAdapter>
    </QueryClientProvider>,
  );
  return { invalidateSpy };
}

async function selectAllRows() {
  const checkboxes = await screen.findAllByLabelText("Select row");
  for (const checkbox of checkboxes) {
    fireEvent.click(checkbox);
  }
  return checkboxes.length;
}

beforeEach(() => {
  useDataTableQueryMock.mockReturnValue({
    data: { results: rows, count: rows.length },
    isLoading: false,
    isError: false,
    error: null,
  });
  permissionState.approve = true;
  permissionState.reject = true;
  permissionState.cancel = true;
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("PTODataTable bulk actions", () => {
  it("shows Approve, Reject and Cancel in the dock once rows are selected", async () => {
    renderTable();

    expect(screen.queryByRole("button", { name: /^approve$/i })).not.toBeInTheDocument();
    const count = await selectAllRows();
    expect(count).toBe(2);

    expect(await screen.findByText("2 selected")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /^approve$/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /^reject$/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /^cancel$/i })).toBeInTheDocument();
  });

  it("approves only the eligible rows with one bulk call after confirmation", async () => {
    bulkWorkerPTOAction.mockResolvedValue({
      successCount: 1,
      failureCount: 0,
      results: [{ ptoId: "wrkpto_requested", success: true, error: "" }],
    });
    const { invalidateSpy } = renderTable();
    await selectAllRows();

    fireEvent.click(await screen.findByRole("button", { name: /^approve$/i }));

    const dialog = await screen.findByRole("alertdialog");
    expect(dialog).toHaveTextContent("Approve 1 PTO request");
    expect(dialog).toHaveTextContent("1 selected request is not eligible and will be skipped");
    expect(bulkWorkerPTOAction).not.toHaveBeenCalled();

    fireEvent.click(screen.getByRole("button", { name: /^approve$/i, hidden: false }));

    await waitFor(() => {
      expect(bulkWorkerPTOAction).toHaveBeenCalledExactlyOnceWith({
        ptoIds: ["wrkpto_requested"],
        action: "Approve",
      });
    });
    await waitFor(() => {
      expect(toastSuccess).toHaveBeenCalledWith("Approved 1 PTO request (1 ineligible skipped)");
    });
    await waitFor(() => {
      expect(invalidateSpy).toHaveBeenCalledWith({
        queryKey: ["worker-pto-list"],
        refetchType: "all",
      });
    });
  });

  it("tells the user when nothing in the selection can be rejected", async () => {
    useDataTableQueryMock.mockReturnValue({
      data: { results: [rows[1]], count: 1 },
      isLoading: false,
      isError: false,
      error: null,
    });
    renderTable();
    await selectAllRows();

    fireEvent.click(await screen.findByRole("button", { name: /^reject$/i }));

    expect(toastInfo).toHaveBeenCalledExactlyOnceWith("Only requested PTO can be rejected.");
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    expect(bulkWorkerPTOAction).not.toHaveBeenCalled();
  });

  it("opens the reason dialog scoped to eligible ids for a bulk reject", async () => {
    renderTable();
    await selectAllRows();

    fireEvent.click(await screen.findByRole("button", { name: /^reject$/i }));

    const dialog = await screen.findByRole("dialog");
    expect(dialog).toHaveTextContent("Reject this PTO request");
    expect(dialog).toHaveTextContent("1 selected request is not eligible and will be skipped");
  });

  it("hides selection entirely when the user holds none of the decision permissions", () => {
    permissionState.approve = false;
    permissionState.reject = false;
    permissionState.cancel = false;
    renderTable();

    expect(screen.queryByLabelText("Select row")).not.toBeInTheDocument();
  });

  it("opens the request form from the toolbar Add Record button", async () => {
    renderTable();

    fireEvent.click(await screen.findByRole("button", { name: /add record/i }));

    expect(await screen.findByRole("dialog")).toHaveTextContent("request-pto-form");
  });

  it("shows stored days, the auto-approved badge, and the balance after booking", async () => {
    renderTable();

    expect(await screen.findByText("Auto")).toBeInTheDocument();
    expect(screen.getByText("7.50")).toBeInTheDocument();
  });

  it("renders the decision column with the reviewer and their reason", async () => {
    renderTable();

    expect(await screen.findByText("Dana Dispatcher")).toBeInTheDocument();
    expect(screen.getByText("No coverage")).toBeInTheDocument();
  });
});
