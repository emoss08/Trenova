import { beforeEach, describe, expect, it, vi } from "vitest";
import { analytics } from "./analytics";

const getShipmentPageAnalyticsGraphQLMock = vi.hoisted(() => vi.fn());
const analyticsServiceGetMock = vi.hoisted(() => vi.fn());

vi.mock("@/lib/graphql/shipment", () => ({
  getShipmentPageAnalyticsGraphQL: getShipmentPageAnalyticsGraphQLMock,
}));

vi.mock("@/services/api", () => ({
  apiService: {
    analyticService: {
      get: analyticsServiceGetMock,
    },
  },
}));

describe("analytics query keys", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("uses GraphQL for shipment-management analytics with the existing query key", async () => {
    getShipmentPageAnalyticsGraphQLMock.mockResolvedValueOnce({
      page: "shipment-management",
      savedViewCounts: null,
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
      savedViewCounts: null,
      laneHeatmap: {
        cells: [{ count: 18, destination: "South", origin: "Midwest" }],
        total: 18,
        windowDays: 7,
      },
    });
    expect(getShipmentPageAnalyticsGraphQLMock).toHaveBeenCalledWith({
      timezone: expect.any(String),
    });
    expect(analyticsServiceGetMock).not.toHaveBeenCalled();
  });

  it("keeps api-key-management analytics on REST", async () => {
    analyticsServiceGetMock.mockResolvedValueOnce({ page: "api-key-management" });

    const query = analytics.get("api-key-management");
    const response = await query.queryFn();

    expect(query.queryKey).toEqual(["analytics", "api-key-management"]);
    expect(response).toEqual({ page: "api-key-management" });
    expect(analyticsServiceGetMock).toHaveBeenCalledWith({
      page: "api-key-management",
      timezone: expect.any(String),
    });
    expect(getShipmentPageAnalyticsGraphQLMock).not.toHaveBeenCalled();
  });
});
