import { render } from "@testing-library/react";
import { Table } from "@trenova/shared/components/ui/table";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DataTableTotalsRow } from "../data-table-totals-row";

let resize: (() => void) | null = null;
let footerHeight = 0;

// The table's scroll area observes its viewport as well, after the footer, so
// the callback kept is the one watching the footer, not the last one made.
class FakeResizeObserver {
  callback: () => void;
  constructor(callback: () => void) {
    this.callback = callback;
  }
  observe(target: Element) {
    if (target.tagName === "TFOOT") resize = this.callback;
  }
  disconnect() {}
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
    resize = null;
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
    resize?.();

    expect(footerSpace(container)).toBe("55px");
  });

  it("gives the space back when the totals row is taken away", () => {
    footerHeight = 41;
    const { container, rerender } = render(<Totals shown />);

    rerender(<Totals shown={false} />);

    expect(footerSpace(container)).toBe("");
  });
});
