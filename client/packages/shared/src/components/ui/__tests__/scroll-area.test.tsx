import { act, cleanup, render, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ScrollArea } from "../scroll-area";

/**
 * The environment lays nothing out, so the test says how tall the viewport and its content
 * are and delivers the resize notices a browser would.
 */
type Watcher = { notify: () => void; targets: Set<Element> };
let watchers: Watcher[] = [];

beforeEach(() => {
  watchers = [];
  vi.stubGlobal(
    "ResizeObserver",
    class {
      private watcher: Watcher;
      constructor(callback: ResizeObserverCallback) {
        this.watcher = {
          targets: new Set(),
          notify: () => callback([], this as unknown as ResizeObserver),
        };
        watchers.push(this.watcher);
      }
      observe(target: Element) {
        this.watcher.targets.add(target);
      }
      unobserve(target: Element) {
        this.watcher.targets.delete(target);
      }
      disconnect() {
        this.watcher.targets.clear();
      }
    },
  );
});

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

function renderArea(rows: string[]) {
  const view = render(
    <ScrollArea className="h-40">
      {rows.map((row) => (
        <p key={row}>{row}</p>
      ))}
    </ScrollArea>,
  );
  const viewport = view.container.querySelector<HTMLElement>('[data-slot="scroll-area-viewport"]')!;
  const bar = () => view.container.querySelector('[data-slot="scroll-area-scrollbar"]');
  const measure = (contentHeight: number, viewportHeight: number) => {
    Object.defineProperty(viewport, "scrollHeight", { configurable: true, value: contentHeight });
    Object.defineProperty(viewport, "clientHeight", { configurable: true, value: viewportHeight });
  };
  /** What a browser does when something it was asked to watch changes size. */
  const resized = (element: Element) =>
    act(() => {
      for (const watcher of watchers) {
        if (watcher.targets.has(element)) watcher.notify();
      }
    });

  return { ...view, viewport, bar, measure, resized };
}

describe("a scroll area's bar", () => {
  it("is not shown while everything fits", async () => {
    const area = renderArea(["One"]);
    area.measure(160, 160);

    await area.resized(area.viewport);

    expect(area.bar()).toHaveClass("hidden");
  });

  // The bar used to follow the primitive's own measurement, taken only when the viewport
  // was scrolled or resized: content that grew had no bar until it was scrolled, and
  // content that shrank kept the bar of what had been there.
  it("appears when the content outgrows the viewport, before anything is scrolled", async () => {
    const area = renderArea(["One"]);
    area.measure(160, 160);
    await area.resized(area.viewport);

    area.measure(900, 160);
    await area.resized(area.getByText("One"));

    expect(area.bar()).not.toHaveClass("hidden");
  });

  it("goes when the content shrinks back to fit", async () => {
    const area = renderArea(["One"]);
    area.measure(900, 160);
    await area.resized(area.getByText("One"));
    expect(area.bar()).not.toHaveClass("hidden");

    area.measure(120, 160);
    await area.resized(area.getByText("One"));

    expect(area.bar()).toHaveClass("hidden");
  });

  it("watches what is put in the area after it mounts", async () => {
    const area = renderArea(["One"]);
    area.measure(160, 160);
    await area.resized(area.viewport);

    area.measure(900, 160);
    area.rerender(
      <ScrollArea className="h-40">
        <p>One</p>
        <p>Two</p>
      </ScrollArea>,
    );

    await waitFor(() => expect(area.bar()).not.toHaveClass("hidden"));
    expect(watchers.some((watcher) => watcher.targets.has(area.getByText("Two")))).toBe(true);
  });
});
