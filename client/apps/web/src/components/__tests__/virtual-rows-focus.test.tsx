import { stubLayout, stubScrollTo } from "@/test/layout";
import { act, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { VirtualRows } from "../virtual-rows";

let restore: (() => void)[] = [];
beforeEach(() => {
  restore = [stubLayout(320), stubScrollTo()];
});
afterEach(() => {
  for (const undo of restore) undo();
});

const rows = Array.from({ length: 200 }, (_, index) => ({
  key: `row-${index}`,
  render: () => <button type="button">Row {index}</button>,
}));

describe("VirtualRows and keyboard focus", () => {
  it("keeps the row holding focus mounted, and focused, when it scrolls out of the window", async () => {
    const { container } = render(<VirtualRows rows={rows} estimateSize={40} initialHeight={320} />);
    const first = screen.getByRole("button", { name: "Row 0" });
    act(() => first.focus());

    const scroller = container.firstElementChild as HTMLElement;
    await act(async () => {
      scroller.scrollTop = 1_000_000;
      scroller.dispatchEvent(new Event("scroll"));
      await new Promise((resolve) => setTimeout(resolve, 20));
    });

    expect(screen.getByText("Row 199")).toBeInTheDocument();
    expect(screen.queryByText("Row 1")).toBeNull();
    expect(screen.getByRole("button", { name: "Row 0" })).toBe(document.activeElement);
  });
});
