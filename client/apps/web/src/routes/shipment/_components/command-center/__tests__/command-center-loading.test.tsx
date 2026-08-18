import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import type { Shipment } from "@trenova/shared/types/shipment";
import type { User } from "@trenova/shared/types/user";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import { afterEach, describe, expect, it, vi } from "vitest";
import { CommandCenterTable } from "../command-center-table";

type ShipmentPage = {
  results: Shipment[];
  count: number;
  pageInfo: { endCursor: string | null; hasNextPage: boolean };
};

const listShipments = vi.hoisted(() => {
  const state = { resolve: (_page: unknown) => {} };
  const fn = () =>
    new Promise((resolve) => {
      state.resolve = resolve;
    });
  return { fn, state };
});

vi.mock("@/lib/graphql/shipment", () => ({
  listShipmentsGraphQL: listShipments.fn,
}));

vi.mock("../use-view-counts", () => ({
  useSavedViewCounts: () => ({}),
}));

vi.mock("@/lib/queries", () => ({
  queries: {
    tableConfiguration: {
      default: () => ({ queryKey: ["table-config-default"], queryFn: async () => null }),
    },
  },
}));

function signIn() {
  useAuthStore.setState({
    user: {
      currentOrganizationId: "org_01",
      memberships: [
        {
          userId: "usr_01",
          organizationId: "org_01",
          isDefault: true,
          organization: {
            id: "org_01",
            name: "Hybrid Co",
            brokerageEnabled: true,
            assetOperationsEnabled: true,
          },
        },
      ],
    } as unknown as User,
    isAuthenticated: true,
  });
}

const columns: ColumnDef<Shipment>[] = [
  {
    id: "proNumber",
    header: "Pro Number",
    accessorFn: (row: Shipment) => row.proNumber,
    cell: ({ row }: { row: { original: Shipment } }) => row.original.proNumber,
  } as unknown as ColumnDef<Shipment>,
];

function renderTable() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

  return render(
    <NuqsTestingAdapter searchParams="?mode=table">
      <QueryClientProvider client={queryClient}>
        <CommandCenterTable
          columns={columns}
          rowActions={[]}
          mandatoryFieldFilters={[]}
          onUploadDocument={vi.fn()}
        />
      </QueryClientProvider>
    </NuqsTestingAdapter>,
  );
}

function resolvePage(page: ShipmentPage) {
  listShipments.state.resolve(page);
}

describe("command center table loading state", () => {
  afterEach(() => {
    cleanup();
    useAuthStore.setState({ user: null, isAuthenticated: false });
  });

  it("shows skeleton rows while the first page is in flight, never the empty state", () => {
    signIn();

    renderTable();

    expect(screen.getAllByTestId("command-center-skeleton-row").length).toBeGreaterThan(0);
    expect(screen.queryByText("No shipments match the current view.")).toBeNull();
  });

  it("replaces the skeletons with data rows once the page resolves", async () => {
    signIn();

    renderTable();

    resolvePage({
      results: [{ id: "shp_01", proNumber: "S-100" } as Shipment],
      count: 1,
      pageInfo: { endCursor: null, hasNextPage: false },
    });

    expect(await screen.findByText("S-100")).toBeInTheDocument();
    expect(screen.queryByTestId("command-center-skeleton-row")).toBeNull();
  });

  it("replaces the skeletons with the empty state when the page resolves with no rows", async () => {
    signIn();

    renderTable();

    resolvePage({
      results: [],
      count: 0,
      pageInfo: { endCursor: null, hasNextPage: false },
    });

    expect(await screen.findByText("No shipments match the current view.")).toBeInTheDocument();
    expect(screen.queryByTestId("command-center-skeleton-row")).toBeNull();
  });
});
