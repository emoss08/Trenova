import { TestRouter } from "@trenova/shared/testing/test-router";
import type { HomeWidget } from "@/lib/graphql/home-layout";
import { defaultAnalyticsData } from "@/lib/shipment-analytics";
import { cleanup, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, describe, expect, it } from "vitest";
import type { HomeData } from "../use-home-data";
import { KPIRowWidget, KPIWidget } from "../widgets/pulse-widgets";

const accountingLock = {
  feature: "tms.accounting",
  featureName: "Accounting",
  addOn: "ACC",
  addOnName: "Accounting & Cash",
};

function homeData(costingDenied: boolean): HomeData {
  return {
    shipmentAnalytics: {
      ...defaultAnalyticsData,
      profitability: {
        ...defaultAnalyticsData.profitability,
        planDenial: costingDenied ? accountingLock : null,
      },
    },
    shipmentAnalyticsLoading: false,
    shipmentAnalyticsReady: true,
    attention: undefined,
    attentionLoading: false,
    receivables: undefined,
    receivablesLoading: false,
  } as HomeData;
}

function strip(metrics: string[], lockedMetrics: HomeWidget["lockedMetrics"] = []): HomeWidget {
  return {
    id: "wgt_01",
    key: "kpi-row",
    label: "kpi-row",
    title: "Key figures",
    w: 12,
    h: 2,
    planLock: null,
    lockedMetrics,
    config: { metrics },
  } as unknown as HomeWidget;
}

function Wrapper({ children }: { children: ReactNode }) {
  return <TestRouter>{children}</TestRouter>;
}

afterEach(cleanup);

describe("a key figures strip on a plan that leaves some figures out", () => {
  it("draws only the figures on the plan, with no add-on sentence in the gap", () => {
    const { container } = render(
      <KPIRowWidget
        widget={strip(
          ["activeShipments", "arTotalOpen"],
          [{ metric: "arTotalOpen", label: "Open receivables", lock: accountingLock }],
        )}
        data={homeData(false)}
      />,
      { wrapper: Wrapper },
    );

    expect(screen.queryByText("Open receivables")).not.toBeInTheDocument();
    expect(screen.queryByText(/not on your plan/)).not.toBeInTheDocument();
    expect(container.querySelector('[data-slot="plan-locked"]')).toBeNull();
    expect(screen.getByRole("link")).toBeInTheDocument();
  });

  it("drops a costing figure the analytics answer says the plan leaves out", () => {
    render(
      <KPIRowWidget widget={strip(["activeShipments", "marginPercent"])} data={homeData(true)} />,
      { wrapper: Wrapper },
    );

    expect(screen.queryByText("Margin")).not.toBeInTheDocument();
    expect(screen.getAllByRole("link")).toHaveLength(1);
  });

  it("draws nothing, not a request for setup, when every figure is off the plan", () => {
    const { container } = render(
      <KPIRowWidget widget={strip(["marginPercent", "costPerMile"])} data={homeData(true)} />,
      { wrapper: Wrapper },
    );

    expect(container).toBeEmptyDOMElement();
    expect(screen.queryByText("Choose the metrics this strip shows.")).not.toBeInTheDocument();
  });
});

describe("a single metric tile whose figure the analytics answer says the plan leaves out", () => {
  function tile(metric: string): HomeWidget {
    return { ...strip([]), key: "kpi", title: null, config: { metric } } as unknown as HomeWidget;
  }

  it("draws nothing for a costing figure, with no locked card", () => {
    for (const metric of ["marginPercent", "costPerMile"]) {
      const { container, unmount } = render(
        <KPIWidget widget={tile(metric)} data={homeData(true)} />,
        { wrapper: Wrapper },
      );

      expect(container, metric).toBeEmptyDOMElement();
      expect(screen.queryByText(/not on your plan/i)).not.toBeInTheDocument();
      unmount();
    }
  });

  it("draws the figure when the plan includes costing", () => {
    render(<KPIWidget widget={tile("costPerMile")} data={homeData(false)} />, {
      wrapper: Wrapper,
    });

    expect(screen.getByRole("link", { name: "Cost / mile" })).toBeInTheDocument();
  });
});
