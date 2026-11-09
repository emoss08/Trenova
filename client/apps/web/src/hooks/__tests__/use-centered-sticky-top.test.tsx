import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useCenteredStickyTop } from "../use-centered-sticky-top";

beforeEach(() => {
  vi.stubGlobal(
    "ResizeObserver",
    class {
      observe() {}
      disconnect() {}
    },
  );
});

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
});

function Paper({ height }: { height: number }) {
  const ref = useCenteredStickyTop<HTMLElement>();
  return (
    <div style={{ overflowY: "auto" }} data-testid="scroller">
      <article
        ref={(node) => {
          if (node) Object.defineProperty(node, "offsetHeight", { value: height });
          ref.current = node;
        }}
        aria-label="paper"
      />
    </div>
  );
}

function setScrollerHeight(height: number) {
  Object.defineProperty(HTMLElement.prototype, "clientHeight", {
    configurable: true,
    get() {
      return this.dataset.testid === "scroller" ? height : 0;
    },
  });
}

describe("useCenteredStickyTop", () => {
  it("holds the element in the middle of its scroll area", () => {
    setScrollerHeight(1000);
    render(<Paper height={800} />);

    expect(screen.getByLabelText("paper").style.top).toBe("100px");
  });

  it("never sits closer to the top than the minimum when the element is taller than the view", () => {
    setScrollerHeight(600);
    render(<Paper height={900} />);

    expect(screen.getByLabelText("paper").style.top).toBe("16px");
  });
});
