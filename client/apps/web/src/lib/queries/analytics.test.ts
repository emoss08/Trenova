import { beforeEach, describe, expect, it, vi } from "vitest";
import { analytics } from "./analytics";

const getShipmentPageAnalyticsGraphQLMock = vi.hoisted(() => vi.fn());
const getApiKeyAnalyticsGraphQLMock = vi.hoisted(() => vi.fn());

vi.mock("@/lib/graphql/shipment", () => ({
  getShipmentPageAnalyticsGraphQL: getShipmentPageAnalyticsGraphQLMock,
}));

vi.mock("@/lib/graphql/api-key-analytics", () => ({
  getApiKeyAnalyticsGraphQL: getApiKeyAnalyticsGraphQLMock,
}));

describe("analytics query keys", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("uses GraphQL for shipment-management analytics with the existing query key", async () => {
    getShipmentPageAnalyticsGraphQLMock.mockResolvedValueOnce({
      page: "shipment-management",
      laneHeatmap: {
        cells: [{ count: 18, destination: "South", origin: "Midwest" }],
        total: 18,
        windowDays: 7,
      },
    });

    const query = analytics.get("shipment-management");
    const response = await query.queryFn();

    expect(query.queryKey).toEqual(["analytics", "shipment-management"]);
    expect(response).toEqual({
      page: "shipment-management",
      laneHeatmap: {
        cells: [{ count: 18, destination: "South", origin: "Midwest" }],
        total: 18,
        windowDays: 7,
      },
    });
    expect(getShipmentPageAnalyticsGraphQLMock).toHaveBeenCalledWith({
      timezone: expect.any(String),
    });
    expect(getApiKeyAnalyticsGraphQLMock).not.toHaveBeenCalled();
  });

  it("reads api-key-management analytics from GraphQL, forwarding the signal", async () => {
    const figures = {
      totalKeys: { count: 3, newThisMonth: 1 },
      activeKeys: { count: 2, percentOfTotal: 66.7 },
      revokedKeys: { count: 1, percentOfTotal: 33.3 },
      requests30d: { total: 40, sparkline: [{ day: "Jan 2", value: 40 }] },
    };
    getApiKeyAnalyticsGraphQLMock.mockResolvedValueOnce(figures);
    const controller = new AbortController();

    const query = analytics.get("api-key-management");
    const response = await query.queryFn({ signal: controller.signal });

    expect(query.queryKey).toEqual(["analytics", "api-key-management"]);
    expect(response).toEqual(figures);
    expect(getApiKeyAnalyticsGraphQLMock).toHaveBeenCalledWith({
      timezone: expect.any(String),
      signal: controller.signal,
    });
    expect(getShipmentPageAnalyticsGraphQLMock).not.toHaveBeenCalled();
  });
});
