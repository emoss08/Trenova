import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { stubLayout } from "@/test/layout";
import ShipmentTimeline from "../shipment-timeline";

const { requestGraphQLMock, setUrlMock, urlState } = vi.hoisted(() => ({
  requestGraphQLMock: vi.fn(),
  setUrlMock: vi.fn(),
  urlState: {
    current: {
      tl: "day",
      tlday: "2026-11-03",
      tlrows: "shipments",
      qf: [{ filter: "Late" }],
    } as Record<string, unknown>,
  },
}));

vi.mock("@trenova/shared/lib/graphql", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@trenova/shared/lib/graphql")>()),
  requestGraphQL: requestGraphQLMock,
}));
vi.mock("@/hooks/use-user-timezone", () => ({ useUserTimezone: () => "UTC" }));
vi.mock("../../url-state", () => ({
  useShipmentBoardUrl: () => [urlState.current, setUrlMock],
}));

const TUE = Date.UTC(2026, 10, 3) / 1000;
const HOUR = 3_600;
const queryOptions = { query: "", fieldFilters: [], filterGroups: [], sort: [] };

/** A ShipmentTimeline response, shaped as the operation selects it. */
function response(nodes: Record<string, unknown>[]) {
  return {
    shipments: {
      edges: nodes.map((node) => ({ node })),
      totalCount: nodes.length,
      totalCountExact: true,
      pageInfo: { hasNextPage: false, endCursor: null },
    },
  };
}

function renderTimeline() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <ShipmentTimeline queryOptions={queryOptions} />
    </QueryClientProvider>,
  );
}

// The sheet is virtualized: a viewport that measures nothing mounts no rows.
let restoreLayout: () => void;
beforeEach(() => {
  requestGraphQLMock.mockReset();
  setUrlMock.mockReset();
  restoreLayout = stubLayout(640, 1280);
});
afterEach(() => {
  cleanup();
  restoreLayout();
});

describe("ShipmentTimeline", () => {
  it("asks the server only for the loads active on the day the URL names, under the board's quick filters", async () => {
    requestGraphQLMock.mockResolvedValue(response([]));

    renderTimeline();

    await waitFor(() => expect(requestGraphQLMock).toHaveBeenCalled());
    const { operationName, variables } = requestGraphQLMock.mock.calls[0][0];
    expect(operationName).toBe("ShipmentTimeline");
    expect(variables.input).toMatchObject({
      activityWindowStart: TUE,
      activityWindowEnd: TUE + 24 * HOUR,
      quickFilters: [{ filter: "Late" }],
      timezone: "UTC",
    });
    expect(await screen.findByText("No loads are active in this range.")).toBeTruthy();
  });

  it("moves to the next day when the dispatcher steps later", async () => {
    requestGraphQLMock.mockResolvedValue(response([]));

    renderTimeline();
    fireEvent.click(await screen.findByRole("button", { name: "Later" }));

    expect(setUrlMock).toHaveBeenCalledWith({ tlday: "2026-11-04" });
  });

  it("shows how late a load runs past its appointment", async () => {
    requestGraphQLMock.mockResolvedValue(
      response([
        {
          id: "shp_1",
          proNumber: "S-24871",
          stage: "Late",
          status: "InTransit",
          tenderStatus: null,
          customer: { id: "cus_1", name: "Marlow Auto Parts" },
          eta: {
            stopId: "stp_d",
            estimatedArrival: TUE + 10.9 * HOUR,
            stops: [
              {
                stopId: "stp_d",
                arrived: false,
                estimatedArrival: TUE + 10.9 * HOUR,
                windowEnd: TUE + 9 * HOUR,
                slackMinutes: -114,
                driveMinutes: 120,
              },
            ],
          },
          moves: [
            {
              id: "mov_1",
              status: "InTransit",
              coverageType: "unassigned",
              stops: [
                {
                  id: "stp_p",
                  type: "Pickup",
                  status: "Completed",
                  scheduledWindowStart: TUE + 6 * HOUR,
                  scheduledWindowEnd: TUE + 7 * HOUR,
                  actualArrival: null,
                  actualDeparture: TUE + 6.5 * HOUR,
                  location: { city: "Laredo", state: { abbreviation: "TX" } },
                },
                {
                  id: "stp_d",
                  type: "Delivery",
                  status: "New",
                  scheduledWindowStart: TUE + 8 * HOUR,
                  scheduledWindowEnd: TUE + 9 * HOUR,
                  actualArrival: null,
                  actualDeparture: null,
                  location: { city: "Dallas", state: { abbreviation: "TX" } },
                },
              ],
            },
          ],
        },
      ]),
    );

    renderTimeline();

    expect(await screen.findByText("Laredo → Dallas")).toBeTruthy();
    expect(screen.getByText(/^1h 54m late · ETA /)).toBeTruthy();
    expect(screen.getByRole("button", { name: "S-24871, Laredo → Dallas" })).toBeTruthy();
  });
});
