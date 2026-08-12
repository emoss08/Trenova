import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type { ColumnDef } from "@trenova/shared/types/data-table";
import type { OrganizationCapabilities } from "@trenova/shared/types/organization-capability";
import type { Shipment } from "@trenova/shared/types/shipment";
import type { User } from "@trenova/shared/types/user";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import { afterEach, describe, expect, it, vi } from "vitest";
import { CommandCenterTable } from "../command-center-table";
import { resolveCommandCenterViewMode } from "../url-state";

vi.mock("../timeline", () => ({
  default: () => <div data-testid="command-center-timeline" />,
}));

vi.mock("@/lib/graphql/shipment", () => ({
  listShipmentsGraphQL: async () => ({
    results: [],
    count: 0,
    pageInfo: { endCursor: null, hasNextPage: false },
  }),
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

const hybrid: OrganizationCapabilities = {
  brokerageEnabled: true,
  assetOperationsEnabled: true,
};
const brokerageOnly: OrganizationCapabilities = {
  brokerageEnabled: true,
  assetOperationsEnabled: false,
};

function signIn(capabilities: OrganizationCapabilities) {
  useAuthStore.setState({
    user: {
      currentOrganizationId: "org_01",
      memberships: [
        {
          userId: "usr_01",
          organizationId: "org_01",
          isDefault: true,
          organization: { id: "org_01", name: "Brokerage Co", ...capabilities },
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
    accessorFn: () => null,
    cell: () => null,
  } as unknown as ColumnDef<Shipment>,
];

function renderTable(searchParams: string) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

  return render(
    <NuqsTestingAdapter searchParams={searchParams}>
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

describe("resolveCommandCenterViewMode", () => {
  it("keeps the timeline for an organization that employs drivers", () => {
    expect(resolveCommandCenterViewMode("timeline", hybrid)).toBe("timeline");
    expect(resolveCommandCenterViewMode("table", hybrid)).toBe("table");
  });

  it("resolves a bookmarked timeline back to the table for a brokerage", () => {
    expect(resolveCommandCenterViewMode("timeline", brokerageOnly)).toBe("table");
    expect(resolveCommandCenterViewMode("table", brokerageOnly)).toBe("table");
  });
});

describe("command center timeline visibility", () => {
  afterEach(() => {
    cleanup();
    useAuthStore.setState({ user: null, isAuthenticated: false });
  });

  it("offers the timeline view to an organization that employs drivers", async () => {
    signIn(hybrid);

    renderTable("?mode=timeline");

    expect(screen.getByRole("button", { name: "Timeline" })).toBeInTheDocument();
    expect(await screen.findByTestId("command-center-timeline")).toBeInTheDocument();
  });

  it("removes the option entirely for a brokerage rather than leaving an empty tab", () => {
    signIn(brokerageOnly);

    renderTable("?mode=table");

    expect(screen.queryByRole("button", { name: "Timeline" })).toBeNull();
    expect(screen.queryByRole("group", { name: "View mode" })).toBeNull();
  });

  it("leaves the timeline unmounted on a bookmarked timeline URL", async () => {
    signIn(brokerageOnly);

    renderTable("?mode=timeline");

    // The table is what a lazy timeline chunk would have replaced; waiting for it
    // proves the timeline is never mounted rather than merely not mounted yet.
    expect(await screen.findByRole("table")).toBeInTheDocument();
    expect(screen.queryByTestId("command-center-timeline")).toBeNull();
  });
});
