import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import IftaJurisdictionMileageTable from "../ifta-jurisdiction-mileage-table";

const useDataTableQueryMock = vi.hoisted(() =>
  vi.fn(() => ({
    data: { results: [] as unknown[], count: 0 },
    isLoading: false,
    isError: false,
    error: null,
  })),
);

vi.mock("../ifta-jurisdiction-mileage-panel", () => ({
  IftaJurisdictionMileagePanel: () => null,
}));

vi.mock("../delete-ifta-mileage-entry-dialog", () => ({
  DeleteIftaMileageEntryDialog: () => null,
}));

vi.mock("@/lib/ifta-jurisdiction-options", () => ({
  useIftaJurisdictionOptions: () => ({ jurisdictions: [], groups: [], byId: new Map() }),
  jurisdictionFilterOptions: () => [],
  jurisdictionLabel: (jurisdiction: { code: string; name: string }) =>
    `${jurisdiction.code} — ${jurisdiction.name}`,
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
      all: () => ({ queryKey: ["tableConfig-all"], queryFn: () => ({ results: [], count: 0 }) }),
    },
  },
}));

vi.mock("@trenova/shared/hooks/use-debounce", () => ({
  useDebounce: <T,>(value: T): T => value,
}));

function renderTable() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } });
  render(
    <QueryClientProvider client={client}>
      <NuqsTestingAdapter hasMemory resetUrlUpdateQueueOnMount={false}>
        <IftaJurisdictionMileageTable />
      </NuqsTestingAdapter>
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  useDataTableQueryMock.mockReturnValue({
    data: { results: [], count: 0 },
    isLoading: false,
    isError: false,
    error: null,
  });
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("IftaJurisdictionMileageTable empty state", () => {
  it("says most miles arrive on their own and manual entries are the exception", async () => {
    renderTable();

    expect(await screen.findByText("No manual miles yet")).toBeInTheDocument();
    expect(
      screen.getByText(
        "Most miles come from shipment moves automatically. Enter miles here only for travel the system did not see.",
      ),
    ).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /clear filters/i })).not.toBeInTheDocument();
  });
});
