import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach } from "vitest";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import ServiceTypeTable from "../service-type-table";

const patchMock = vi.hoisted(() => vi.fn());

const useDataTableQueryMock = vi.hoisted(() =>
  vi.fn(() => ({
    data: {
      results: [
        {
          id: "st_1",
          status: "Active",
          code: "LTL",
          description: "Less than truckload",
          color: "#2563eb",
          createdAt: 1700000000,
        },
      ],
      count: 1,
    },
    isLoading: false,
    isError: false,
    error: null,
  })),
);

vi.mock("../service-type-panel", () => ({
  ServiceTypePanel: () => null,
}));

vi.mock("@/services/api", () => ({
  apiService: {
    serviceTypeService: {
      patch: patchMock,
      bulkUpdateStatus: vi.fn(),
    },
  },
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
  usePermission: () => ({ allowed: true, isLoading: false }),
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
  },
}));

vi.mock("@/hooks/use-debounce", () => ({
  useDebounce: <T,>(value: T): T => value,
}));

function renderServiceTypeTable() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  });
  const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries");
  render(
    <QueryClientProvider client={queryClient}>
      <NuqsTestingAdapter hasMemory resetUrlUpdateQueueOnMount={false}>
        <ServiceTypeTable />
      </NuqsTestingAdapter>
    </QueryClientProvider>,
  );
  return { invalidateSpy };
}

describe("ServiceTypeTable inline cell editing", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("patches the service type and refreshes the list when a description edit commits", async () => {
    patchMock.mockResolvedValue({});
    const { invalidateSpy } = renderServiceTypeTable();

    fireEvent.click(screen.getByLabelText("Edit description"));
    const input = await screen.findByRole("textbox", { name: "Edit description" });
    expect(input).toHaveValue("Less than truckload");

    fireEvent.change(input, { target: { value: "Volume LTL" } });
    fireEvent.keyDown(input, { key: "Enter" });

    await waitFor(() => {
      expect(patchMock).toHaveBeenCalledExactlyOnceWith("st_1", { description: "Volume LTL" });
    });
    await waitFor(() => {
      expect(invalidateSpy).toHaveBeenCalledWith({
        queryKey: ["service-type-list"],
        refetchType: "all",
      });
    });
    expect(screen.queryByRole("textbox", { name: "Edit description" })).not.toBeInTheDocument();
  });

  it("rejects clearing the code and keeps the editor open", async () => {
    renderServiceTypeTable();

    fireEvent.click(screen.getByLabelText("Edit code"));
    const input = await screen.findByRole("textbox", { name: "Edit code" });
    expect(input).toHaveValue("LTL");

    fireEvent.change(input, { target: { value: "" } });
    fireEvent.keyDown(input, { key: "Enter" });

    await waitFor(() => {
      expect(screen.getByRole("textbox", { name: "Edit code" })).toBeInTheDocument();
    });
    expect(patchMock).not.toHaveBeenCalled();
  });

  it("does not offer inline editing on the timestamp column", () => {
    renderServiceTypeTable();
    expect(screen.queryByLabelText("Edit createdAt")).not.toBeInTheDocument();
  });
});
