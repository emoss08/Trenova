import { render } from "@testing-library/react";
import { Table } from "@trenova/shared/components/ui/table";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DataTableTotalsRow } from "../data-table-totals-row";

const observed = new Map<Element, () => void>();
let footerHeight = 0;

// The table's scroll area observes its own elements too, so each callback is
// kept against what it observes and the footer's is the one fired.
class FakeResizeObserver {
  private readonly callback: () => void;
  constructor(callback: () => void) {
    this.callback = callback;
  }
  observe(target: Element) {
    observed.set(target, this.callback);
  }
  disconnect() {
    for (const [target, callback] of observed) {
      if (callback === this.callback) observed.delete(target);
    }
  }
}

function resizeFooter(container: HTMLElement) {
  const footer = container.querySelector("tfoot");
  if (footer) observed.get(footer)?.();
}

function Totals({ shown }: { shown: boolean }) {
  return (
    <Table>
      {shown ? (
        <DataTableTotalsRow columns={[]} totals={undefined} loading={false} summable={new Set()} />
      ) : null}
    </Table>
  );
}

function footerSpace(container: HTMLElement) {
  return container
    .querySelector<HTMLElement>('[data-slot="table-container"]')
    ?.style.getPropertyValue("--dt-foot-h");
}

describe("DataTableTotalsRow", () => {
  beforeEach(() => {
    vi.stubGlobal("ResizeObserver", FakeResizeObserver);
    vi.spyOn(HTMLElement.prototype, "offsetHeight", "get").mockImplementation(function (
      this: HTMLElement,
    ) {
      return this.tagName === "TFOOT" ? footerHeight : 0;
    });
  });
  afterEach(() => {
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
    observed.clear();
    footerHeight = 0;
  });

  it("reserves its height at the bottom of the table so the scroll fade ends above it", () => {
    footerHeight = 41;
    const { container } = render(<Totals shown />);

    expect(footerSpace(container)).toBe("41px");
  });

  it("keeps the reserved space in step when the row grows", () => {
    footerHeight = 41;
    const { container } = render(<Totals shown />);

    footerHeight = 55;
    resizeFooter(container);

    expect(footerSpace(container)).toBe("55px");
  });

  it("gives the space back when the totals row is taken away", () => {
    footerHeight = 41;
    const { container, rerender } = render(<Totals shown />);

    rerender(<Totals shown={false} />);

    expect(footerSpace(container)).toBe("");
  });
});
