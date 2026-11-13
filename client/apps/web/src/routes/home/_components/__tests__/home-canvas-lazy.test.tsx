import type { HomeWidget } from "@/lib/graphql/home-layout";
import { act, cleanup, render, screen } from "@testing-library/react";
import { lazy } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { HomeCanvas } from "../home-canvas";
import type { HomeData } from "../use-home-data";

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
    map: widget("Map"),
  };
  return { WIDGET_COMPONENTS: components };
});
vi.mock("../add-widget-dialog", () => ({ AddWidgetDialog: () => null }));
vi.mock("../widget-config-dialog", () => ({ WidgetConfigDialog: () => null }));

type Watcher = { target: Element; root: Element | null; margin: string; enter: () => void };
let watchers: Watcher[] = [];

class FakeIntersectionObserver {
  private readonly callback: IntersectionObserverCallback;
  private readonly options: IntersectionObserverInit;

  constructor(callback: IntersectionObserverCallback, options: IntersectionObserverInit = {}) {
    this.callback = callback;
    this.options = options;
  }

  observe(target: Element) {
    watchers.push({
      target,
      root: (this.options.root as Element | null) ?? null,
      margin: this.options.rootMargin ?? "",
      enter: () =>
        this.callback(
          [{ target, isIntersecting: true } as IntersectionObserverEntry],
          this as unknown as IntersectionObserver,
        ),
    });
  }

  disconnect() {
    watchers = watchers.filter((watcher) => watcher.enter === undefined);
  }

  unobserve() {}
}

function widget(id: string, key: string): HomeWidget {
  return { id, key, title: null, w: 6, h: 5, config: {} } as unknown as HomeWidget;
}

const widgets = [widget("widget_1", "unassigned"), widget("widget_2", "map")];

function renderCanvas() {
  return render(
    <main style={{ overflowY: "auto" }}>
      <HomeCanvas
        widgets={widgets}
        data={{} as HomeData}
        catalog={undefined}
        catalogLoading={false}
        editing={false}
        onChange={() => {}}
      />
    </main>,
  );
}

async function scrollIntoView(index: number) {
  const watcher = watchers[index];
  await act(async () => {
    watcher.enter();
    await Promise.resolve();
  });
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

describe("HomeCanvas tiles", () => {
  it("draws a skeleton and mounts no widget for a tile that has not come into view", () => {
    renderCanvas();

    expect(screen.getAllByLabelText("Loading widget")).toHaveLength(2);
    expect(mounted).toEqual([]);
  });

  it("watches each tile against the element that scrolls it, a little ahead of the fold", () => {
    renderCanvas();

    expect(watchers).toHaveLength(2);
    expect(watchers.map((watcher) => watcher.root?.tagName)).toEqual(["MAIN", "MAIN"]);
    expect(watchers.map((watcher) => watcher.margin)).toEqual(["240px 0px", "240px 0px"]);
  });

  it("mounts only the widget whose tile came into view", async () => {
    renderCanvas();

    await scrollIntoView(0);

    expect(await screen.findByText("Unassigned widget")).toBeInTheDocument();
    expect(screen.queryByText("Map widget")).not.toBeInTheDocument();
    expect(screen.getAllByLabelText("Loading widget")).toHaveLength(1);
    expect(new Set(mounted)).toEqual(new Set(["Unassigned"]));
  });
});
