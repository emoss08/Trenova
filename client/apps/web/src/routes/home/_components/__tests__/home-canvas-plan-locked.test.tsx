import type { HomeWidget } from "@/lib/graphql/home-layout";
import { act, cleanup, render, screen } from "@testing-library/react";
import { lazy } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { HomeCanvas } from "../home-canvas";
import { liveMetricsOf, type HomeData } from "../use-home-data";

const mounted: string[] = [];

vi.mock("../widget-registry", () => {
  const widget = (name: string) =>
    lazy(() =>
      Promise.resolve({
        default: () => {
          mounted.push(name);
          return <p>{name} widget</p>;
        },
      }),
    );
  const components: Record<string, ReturnType<typeof widget>> = {
    unassigned: widget("Unassigned"),
    "ar-snapshot": widget("Receivables"),
  };
  return { WIDGET_COMPONENTS: components };
});
vi.mock("../add-widget-dialog", () => ({ AddWidgetDialog: () => null }));
vi.mock("../widget-config-dialog", () => ({ WidgetConfigDialog: () => null }));
vi.mock("@trenova/shared/hooks/use-plan-feature", () => ({
  usePlanFeature: (feature: string) =>
    feature === "tms.accounting"
      ? { available: false, featureName: "Accounting", addOnName: "Accounting & Cash" }
      : { available: true, featureName: "", addOnName: "" },
}));

type Watcher = { enter: () => void };
let watchers: Watcher[] = [];

class FakeIntersectionObserver {
  private readonly callback: IntersectionObserverCallback;

  constructor(callback: IntersectionObserverCallback) {
    this.callback = callback;
  }

  observe(target: Element) {
    watchers.push({
      enter: () =>
        this.callback(
          [{ target, isIntersecting: true } as IntersectionObserverEntry],
          this as unknown as IntersectionObserver,
        ),
    });
  }

  disconnect() {}

  unobserve() {}
}

async function scrollIntoView(index: number) {
  await act(async () => {
    watchers[index].enter();
    await Promise.resolve();
  });
}

const accountingLock = {
  feature: "tms.accounting",
  featureName: "Accounting",
  addOn: "ACC",
  addOnName: "Accounting & Cash",
};

function widget(overrides: Partial<HomeWidget> & { id: string; key: string }): HomeWidget {
  return {
    label: overrides.key,
    title: null,
    w: 6,
    h: 5,
    planLock: null,
    lockedMetrics: [],
    config: {},
    ...overrides,
  } as unknown as HomeWidget;
}

function renderCanvas(widgets: HomeWidget[], editing = false) {
  return render(
    <main style={{ overflowY: "auto" }}>
      <HomeCanvas
        widgets={widgets}
        data={{} as HomeData}
        catalog={undefined}
        catalogLoading={false}
        editing={editing}
        onChange={() => {}}
      />
    </main>,
  );
}

beforeEach(() => {
  mounted.length = 0;
  watchers = [];
  vi.stubGlobal("IntersectionObserver", FakeIntersectionObserver);
});

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

describe("HomeCanvas plan-locked widgets", () => {
  it("leaves a widget the plan leaves out off Home entirely, with no card and no frame", async () => {
    const { container } = renderCanvas([
      widget({ id: "widget_1", key: "unassigned" }),
      widget({
        id: "widget_2",
        key: "ar-snapshot",
        label: "Receivables snapshot",
        planLock: accountingLock,
      }),
    ]);

    await scrollIntoView(0);

    expect(await screen.findByText("Unassigned widget")).toBeInTheDocument();
    expect(screen.queryByText("Receivables snapshot")).not.toBeInTheDocument();
    expect(screen.queryByText("Not on your plan")).not.toBeInTheDocument();
    expect(container.querySelector('[data-slot="plan-locked"]')).toBeNull();
  });

  it("leaves off a single costing figure and a detention watch the server marked off the plan", () => {
    const detentionLock = {
      feature: "tms.detention",
      featureName: "Detention",
      addOn: "RTE",
      addOnName: "Rating & Revenue Recovery",
    };
    const { container } = renderCanvas([
      widget({
        id: "widget_1",
        key: "kpi",
        label: "Metric",
        planLock: accountingLock,
        config: { metric: "marginPercent" } as HomeWidget["config"],
      }),
      widget({
        id: "widget_2",
        key: "detention-watch",
        label: "Detention watch",
        planLock: detentionLock,
      }),
    ]);

    expect(screen.getByText("Your home screen is empty")).toBeInTheDocument();
    expect(screen.queryByText("Detention watch")).not.toBeInTheDocument();
    expect(container.querySelector('[data-slot="plan-locked"]')).toBeNull();
    expect(mounted).toEqual([]);
  });

  it("leaves off a strip whose every measure the plan leaves out", () => {
    renderCanvas([
      widget({
        id: "widget_1",
        key: "kpi-row",
        lockedMetrics: [{ metric: "arTotalOpen", label: "Open receivables", lock: accountingLock }],
        config: { metrics: ["arTotalOpen"] } as HomeWidget["config"],
      }),
    ]);

    expect(screen.getByText("Your home screen is empty")).toBeInTheDocument();
  });

  it("shows the locked card naming the add-on in the editor, so the person can remove it", () => {
    renderCanvas(
      [
        widget({
          id: "widget_1",
          key: "ar-snapshot",
          label: "Receivables snapshot",
          planLock: accountingLock,
        }),
      ],
      true,
    );

    expect(screen.getByText("Receivables snapshot")).toBeInTheDocument();
    expect(screen.getByText("Part of Accounting & Cash")).toBeInTheDocument();
    expect(screen.getByText("Not on your plan")).toBeInTheDocument();
  });

  it("never mounts a locked widget, so none of its data is asked for", async () => {
    renderCanvas([
      widget({ id: "widget_1", key: "unassigned" }),
      widget({ id: "widget_2", key: "ar-snapshot", planLock: accountingLock }),
    ]);

    expect(watchers, "only the live tile is watched for scrolling into view").toHaveLength(1);
    await scrollIntoView(0);
    await screen.findByText("Unassigned widget");

    expect(new Set(mounted)).toEqual(new Set(["Unassigned"]));
    expect(screen.queryByText("Receivables widget")).not.toBeInTheDocument();
  });

  it("uses the widget's own title for the locked card when it has one", () => {
    renderCanvas(
      [
        widget({
          id: "widget_1",
          key: "ar-snapshot",
          title: "Cash position",
          planLock: accountingLock,
        }),
      ],
      true,
    );

    expect(screen.getAllByText("Cash position").length).toBeGreaterThan(0);
    expect(screen.getByText("Part of Accounting & Cash")).toBeInTheDocument();
  });
});

describe("liveMetricsOf", () => {
  it("reads nothing for a widget the plan leaves out", () => {
    expect(
      liveMetricsOf(
        widget({
          id: "w",
          key: "kpi",
          planLock: accountingLock,
          config: { metric: "arTotalOpen" } as HomeWidget["config"],
        }),
      ),
    ).toEqual([]);
  });

  it("drops the locked measures of a strip and keeps the rest", () => {
    expect(
      liveMetricsOf(
        widget({
          id: "w",
          key: "kpi-row",
          lockedMetrics: [
            { metric: "arTotalOpen", label: "Open receivables", lock: accountingLock },
          ],
          config: { metrics: ["activeShipments", "arTotalOpen"] } as HomeWidget["config"],
        }),
      ),
    ).toEqual(["activeShipments"]);
  });
});
