import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { DataTableGraphQLSource } from "@trenova/shared/types/data-table";
import type { Shipment } from "@trenova/shared/types/shipment";
import ShipmentTimeline from "../shipment-timeline";

const { fetchAllRowsMock } = vi.hoisted(() => ({ fetchAllRowsMock: vi.fn(async () => []) }));

vi.mock("@/lib/data-table-export", () => ({ fetchAllRows: fetchAllRowsMock }));
vi.mock("@/hooks/use-user-timezone", () => ({ useUserTimezone: () => "UTC" }));
vi.mock("../../url-state", () => ({ useShipmentBoardUrl: () => [{}, vi.fn()] }));

const queryOptions = { query: "", fieldFilters: [], filterGroups: [], sort: [] };

function source(quickFilters: string[]): DataTableGraphQLSource<Shipment> {
  return {
    document: "query ShipmentBoardTable { shipments { totalCount } }",
    operationName: "ShipmentBoardTable",
    connectionKey: "shipments",
    inputExtraVariables: () => ({ quickFilters }),
  } as unknown as DataTableGraphQLSource<Shipment>;
}

describe("ShipmentTimeline query", () => {
  beforeEach(() => fetchAllRowsMock.mockClear());

  it("fetches again when the board's quick filters change", async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const view = (graphql: DataTableGraphQLSource<Shipment>) => (
      <QueryClientProvider client={client}>
        <ShipmentTimeline graphql={graphql} queryOptions={queryOptions} />
      </QueryClientProvider>
    );

    const { rerender } = render(view(source([])));
    await waitFor(() => expect(fetchAllRowsMock).toHaveBeenCalledTimes(1));

    rerender(view(source(["Late"])));
    await waitFor(() => expect(fetchAllRowsMock).toHaveBeenCalledTimes(2));
  });
});
