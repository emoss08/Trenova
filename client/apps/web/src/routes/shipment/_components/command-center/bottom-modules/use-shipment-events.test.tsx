import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useShipmentEventsInfinite } from "./use-shipment-events";

const listShipmentEventsMock = vi.hoisted(() => vi.fn());

vi.mock("@/services/api", () => ({
  apiService: {
    shipmentEventService: {
      list: listShipmentEventsMock,
    },
  },
}));

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
  };
}

describe("useShipmentEventsInfinite", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("keeps the shipment-events query key and pages with the last occurredAt cursor", async () => {
    const queryClient = new QueryClient({
      defaultOptions: {
        queries: {
          retry: false,
        },
      },
    });
    const firstPage = Array.from({ length: 20 }, (_, index) => ({
      id: `se_${index + 1}`,
      occurredAt: 1_800_000_000 - index,
    }));
    listShipmentEventsMock
      .mockResolvedValueOnce(firstPage)
      .mockResolvedValueOnce([]);

    const { result } = renderHook(
      () =>
        useShipmentEventsInfinite({
          shipmentId: "shp_1",
          types: ["StatusChanged", "CommentPosted"],
        }),
      { wrapper: createWrapper(queryClient) },
    );

    await waitFor(() => {
      expect(result.current.isSuccess).toBe(true);
    });

    expect(listShipmentEventsMock).toHaveBeenNthCalledWith(1, {
      shipmentId: "shp_1",
      types: ["CommentPosted", "StatusChanged"],
      limit: 20,
      before: undefined,
    });
    const eventQuery = queryClient
      .getQueryCache()
      .getAll()
      .find((query) => query.queryKey[0] === "shipment-events");
    expect(eventQuery?.queryKey).toEqual([
      "shipment-events",
      {
        shipmentId: "shp_1",
        typesKey: "CommentPosted,StatusChanged",
        limit: 20,
        types: ["CommentPosted", "StatusChanged"],
      },
    ]);

    await act(async () => {
      await result.current.fetchNextPage();
    });

    expect(listShipmentEventsMock).toHaveBeenNthCalledWith(2, {
      shipmentId: "shp_1",
      types: ["CommentPosted", "StatusChanged"],
      limit: 20,
      before: 1_799_999_981,
    });
  });
});
