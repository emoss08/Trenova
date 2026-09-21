import { stubLayout } from "@/test/layout";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { VirtualRows } from "../virtual-rows";

let restoreLayout = () => {};

beforeEach(() => {
  restoreLayout = stubLayout(320);
});

afterEach(() => {
  restoreLayout();
  cleanup();
});

function rows(count: number) {
  return Array.from({ length: count }, (_, index) => ({
    key: `row-${index}`,
    render: () => <span data-testid="row">Row {index}</span>,
  }));
}

/**
 * The tool catalog is forty-nine rows and an agent's chosen list can be all
 * of them. Only the rows near the viewport are in the DOM; the rest are a
 * height.
 */
describe("VirtualRows", () => {
  it("mounts only the rows near the viewport", () => {
    render(<VirtualRows rows={rows(200)} estimateSize={40} initialHeight={320} overscan={4} />);

    const mounted = screen.getAllByTestId("row");
    expect(mounted.length).toBeGreaterThan(0);
    expect(mounted.length).toBeLessThan(40);
    expect(screen.getByText("Row 0")).toBeInTheDocument();
    expect(screen.queryByText("Row 199")).not.toBeInTheDocument();
  });

  it("keeps a short list whole", () => {
    render(<VirtualRows rows={rows(5)} />);

    expect(screen.getAllByTestId("row")).toHaveLength(5);
  });

  it("shows the empty state in place of nothing", () => {
    render(<VirtualRows rows={[]} empty={<p>Nothing here</p>} />);

    expect(screen.getByText("Nothing here")).toBeInTheDocument();
  });
});
