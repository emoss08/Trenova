import { stubLayout } from "@/test/layout";
import { act, cleanup, render, screen, within } from "@testing-library/react";
import { useEffect, useRef } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { VirtualRows, type VirtualRowsHandle } from "../virtual-rows";

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

  it("windows its rows inside a masked scroll area when given a mask", () => {
    const { container } = render(
      <VirtualRows
        rows={rows(200)}
        estimateSize={40}
        initialHeight={320}
        overscan={4}
        mask="card"
        aria-label="Resources"
      />,
    );

    const area = container.querySelector('[data-slot="scroll-area"]');
    expect(area).not.toBeNull();
    expect(container.querySelectorAll('[data-slot="scroll-area-mask"]').length).toBeGreaterThan(0);
    expect(within(area as HTMLElement).getByLabelText("Resources")).toBeInTheDocument();
    expect(screen.getAllByTestId("row").length).toBeLessThan(40);
    expect(screen.queryByText("Row 199")).not.toBeInTheDocument();
  });
});

/**
 * Opening a message far down a Triage lane scrolls the list to it. The rows
 * that mount there are measured as they attach, and a row above the viewport
 * that turns out taller than its estimate moves the scroll position to keep
 * the open row still. That move must not ask React to flush while React is
 * still committing the rows that caused it.
 */
describe("VirtualRows measuring rows it scrolled to", () => {
  function ScrollsTo({ target }: { target: string }) {
    const handle = useRef<VirtualRowsHandle>(null);
    useEffect(() => {
      handle.current?.scrollToKey(target);
    }, [target]);

    return (
      <VirtualRows
        rows={rows(300)}
        estimateSize={40}
        initialHeight={320}
        handleRef={handle}
        className="h-80"
      />
    );
  }

  it("never flushes from inside a commit", async () => {
    restoreLayout();
    restoreLayout = stubLayout(64);
    const errors = vi.spyOn(console, "error").mockImplementation(() => {});

    try {
      const { container } = render(<ScrollsTo target="row-120" />);
      const scroller = container.firstElementChild as HTMLElement;
      await act(async () => {
        scroller.scrollTop = 120 * 40;
        scroller.dispatchEvent(new Event("scroll"));
        await new Promise((resolve) => setTimeout(resolve, 20));
      });
      await act(async () => {
        await new Promise((resolve) => setTimeout(resolve, 20));
      });

      expect(screen.getByText("Row 120")).toBeInTheDocument();
      const flushes = errors.mock.calls.filter((call) => String(call[0]).includes("flushSync"));
      expect(flushes).toEqual([]);
    } finally {
      errors.mockRestore();
    }
  });
});
