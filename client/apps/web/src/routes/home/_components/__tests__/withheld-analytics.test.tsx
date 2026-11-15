import { render, screen, within } from "@testing-library/react";
import type { ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { describe, expect, it } from "vitest";
import type { HomeWidget } from "@/lib/graphql/home-layout";
import { LaneHeatmap } from "@/components/work-modules/lane-heatmap";
import {
  defaultAnalyticsData,
  mergeShipmentAnalyticsWithDefaults,
  type ShipmentAnalyticsData,
} from "@/lib/shipment-analytics";
import { resolveMetric } from "../metric-values";
import type { HomeData } from "../use-home-data";
import { DetentionWatchWidget, TomorrowsPickupsWidget } from "../widgets/work-widgets";

function homeData(analytics: Partial<ShipmentAnalyticsData>): HomeData {
  return {
    shipmentAnalytics: { ...defaultAnalyticsData, ...analytics },
    shipmentAnalyticsLoading: false,
    shipmentAnalyticsReady: true,
    attention: undefined,
    attentionLoading: false,
    receivables: undefined,
    receivablesLoading: false,
  };
}

function widget(key: string): HomeWidget {
  return {
    id: "wgt_01",
    key,
    label: key,
    title: "",
    w: 1,
    h: 1,
    planLock: null,
    lockedMetrics: [],
    config: {},
  } as unknown as HomeWidget;
}

function Wrapper({ children }: { children: ReactNode }) {
  return <MemoryRouter>{children}</MemoryRouter>;
}

// A shipment reader who cannot read customers, locations or workers sees each one's reference
// view; a value their role's ceiling withholds arrives as null and is drawn as hidden, never as
// an empty name or "Unassigned".
describe("withheld shipment analytics", () => {
  it("draws a withheld detention customer as hidden", () => {
    render(
      <DetentionWatchWidget
        widget={widget("detention-watch")}
        data={homeData({
          detentionWatchlist: {
            items: [
              { shipmentId: "SHP-1", customer: null, dwellLabel: "3h 10m", tone: "danger" },
              {
                shipmentId: "SHP-2",
                customer: "Acme Foods",
                dwellLabel: "2h 05m",
                tone: "warning",
              },
            ],
          },
        })}
      />,
      { wrapper: Wrapper },
    );

    expect(screen.getByText("Acme Foods")).toBeInTheDocument();
    expect(screen.getAllByText("Hidden")).toHaveLength(1);
  });

  it("never fills a withheld detention watchlist with the sample rows", () => {
    expect(
      mergeShipmentAnalyticsWithDefaults({ detentionWatchlist: null }).detentionWatchlist,
    ).toBeNull();
    expect(
      mergeShipmentAnalyticsWithDefaults({
        detentionWatchlist: { items: null },
      } as unknown as Parameters<typeof mergeShipmentAnalyticsWithDefaults>[0]).detentionWatchlist,
      "a reader who cannot read detention gets the list withheld",
    ).toBeNull();
  });

  it("draws nothing for a detention watchlist the server did not read", () => {
    const { container } = render(
      <DetentionWatchWidget
        widget={widget("detention-watch")}
        data={homeData({ detentionWatchlist: null })}
      />,
      { wrapper: Wrapper },
    );

    expect(container).toBeEmptyDOMElement();
    expect(screen.queryByText(/You don't have access/)).toBeNull();
    expect(screen.queryByText("Nothing dwelling past threshold.")).toBeNull();
  });

  it("keeps the skeleton while the detention watchlist is still loading", () => {
    const { container } = render(
      <DetentionWatchWidget
        widget={widget("detention-watch")}
        data={{ ...homeData({ detentionWatchlist: null }), shipmentAnalyticsLoading: true }}
      />,
      { wrapper: Wrapper },
    );

    expect(container).not.toBeEmptyDOMElement();
    expect(screen.getByText("Detention watch")).toBeInTheDocument();
  });

  it("tells a withheld driver apart from an unassigned pickup", () => {
    render(
      <TomorrowsPickupsWidget
        widget={widget("tomorrows-pickups")}
        data={homeData({
          tomorrowsPickups: {
            date: "2026-05-08",
            pickups: [
              {
                shipmentId: "shp_1",
                proNumber: "PRO-1",
                pickupWindowStart: 1_778_230_800,
                customer: "Acme Foods",
                origin: null,
                destination: "DC-CHI",
                driver: null,
                status: "confirmed",
              },
              {
                shipmentId: "shp_2",
                proNumber: "PRO-2",
                pickupWindowStart: 1_778_234_400,
                customer: "Acme Foods",
                origin: "TERM-LA",
                destination: "DC-CHI",
                driver: "",
                status: "unassigned",
              },
            ],
          },
        })}
      />,
      { wrapper: Wrapper },
    );

    const withheld = screen.getByText("PRO-1").closest("a");
    const unassigned = screen.getByText("PRO-2").closest("a");
    expect(withheld).not.toBeNull();
    expect(unassigned).not.toBeNull();
    expect(within(withheld as HTMLElement).getAllByText("Hidden")).toHaveLength(2);
    expect(within(withheld as HTMLElement).queryByText("Unassigned")).toBeNull();
    expect(within(unassigned as HTMLElement).getByText("Unassigned")).toBeInTheDocument();
    expect(within(unassigned as HTMLElement).queryByText("Hidden")).toBeNull();
  });

  it("shows the margin as hidden without the margins field group", () => {
    const hidden = resolveMetric(
      "marginPercent",
      homeData({
        profitability: {
          ...defaultAnalyticsData.profitability,
          avgMarginPct: null,
          unprofitableCount: null,
        },
      }),
    );
    expect(hidden?.display).toBe("Hidden");
    expect(hidden?.sub, "how many loads lost money is a margin figure too").toBeUndefined();

    const shown = resolveMetric(
      "marginPercent",
      homeData({
        profitability: {
          ...defaultAnalyticsData.profitability,
          avgMarginPct: 18.46,
          unprofitableCount: 3,
        },
      }),
    );
    expect(shown?.display).toBe("18.5%");
    expect(shown?.sub).toBe("3 unprofitable loads");
  });

  // The costing figures are computed from the costing setup, which belongs to Accounting & Cash.
  // A plan without it answers a locked card: the figures are named by their add-on, never drawn
  // as the zeroes the card carries.
  it("locks the margin and the cost per mile when the plan leaves out costing", () => {
    const data = homeData({
      profitability: {
        ...defaultAnalyticsData.profitability,
        planDenial: {
          feature: "tms.accounting",
          featureName: "Accounting",
          addOn: "ACC",
          addOnName: "Accounting & Cash",
        },
      },
    });

    for (const key of ["marginPercent", "costPerMile"]) {
      const metric = resolveMetric(key, data);
      expect(metric?.lockedFeature, key).toBe("tms.accounting");
      expect(metric?.display, key).toBe("");
    }
  });

  it("keeps the revenue per mile when only costing is locked", () => {
    const data = homeData({
      profitability: {
        ...defaultAnalyticsData.profitability,
        planDenial: {
          feature: "tms.accounting",
          featureName: "Accounting",
          addOn: "ACC",
          addOnName: "Accounting & Cash",
        },
      },
    });

    expect(resolveMetric("revenuePerMile", data)?.lockedFeature).toBeUndefined();
  });

  it("keeps a withheld lane heatmap apart from one that has not loaded", () => {
    expect(mergeShipmentAnalyticsWithDefaults({ laneHeatmap: null }).laneHeatmap).toBeNull();
    expect(mergeShipmentAnalyticsWithDefaults({}).laneHeatmap).toEqual(
      defaultAnalyticsData.laneHeatmap,
    );
  });

  it("keeps a withheld customer mix apart from one that has not loaded", () => {
    expect(mergeShipmentAnalyticsWithDefaults({ customerMix: null }).customerMix).toBeNull();
    expect(mergeShipmentAnalyticsWithDefaults({}).customerMix).toEqual(
      defaultAnalyticsData.customerMix,
    );
  });

  it("draws a withheld lane heatmap as hidden", () => {
    render(<LaneHeatmap data={null} />);

    expect(screen.getByText("Hidden")).toBeInTheDocument();
    expect(screen.queryByText("No lane activity")).toBeNull();
  });

  // The widget that shows the heatmap draws the one frame and the one title, so the
  // heatmap brings neither: a second card inside the tile was a frame within a frame.
  it("draws the lane heatmap with no card or heading of its own", () => {
    const { container } = render(
      <LaneHeatmap
        data={{
          windowDays: 7,
          total: 53,
          cells: [{ origin: "South", destination: "South", count: 53 }],
        }}
      />,
    );

    expect(container.querySelector("section, header, footer")).toBeNull();
    expect(screen.queryByRole("heading")).toBeNull();
    expect(screen.getByText("Busiest lane: South → South, 53 loads")).toBeInTheDocument();
  });
});
