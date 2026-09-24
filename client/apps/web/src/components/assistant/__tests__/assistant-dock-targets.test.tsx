import { cleanup, render, screen, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { AssistantDockTargets } from "../assistant-dock-targets";
import { AssistantLauncher } from "../assistant-launcher";

afterEach(cleanup);

function targets() {
  const overlay = screen.getByTestId("assistant-dock-targets");

  return {
    overlay,
    corner: (dock: string) => overlay.querySelector(`[data-dock="${dock}"]`) as HTMLElement,
  };
}

/**
 * A drag with no targets reads as free placement, and a launcher let go in
 * the middle of an edge jumping to a corner looks like a bug. While it is
 * being dragged, every place it can go is drawn, and the one it will go to
 * is marked and named.
 */
describe("AssistantDockTargets", () => {
  it("draws all four corners", () => {
    render(<AssistantDockTargets active="bottom-right" />);

    const { corner } = targets();
    for (const dock of ["bottom-right", "bottom-left", "top-right", "top-left"]) {
      expect(corner(dock)).not.toBeNull();
    }
  });

  it("marks only the corner it will land in", () => {
    render(<AssistantDockTargets active="bottom-left" />);

    const { corner } = targets();
    expect(corner("bottom-left")).toHaveAttribute("data-active");
    expect(corner("bottom-right")).not.toHaveAttribute("data-active");
    expect(corner("top-left")).not.toHaveAttribute("data-active");
  });

  it("names the corner it will land in, and no other", () => {
    render(<AssistantDockTargets active="top-right" />);

    const { overlay, corner } = targets();
    expect(within(corner("top-right")).getByText("Top right")).toBeInTheDocument();
    expect(within(overlay).queryByText("Bottom right")).toBeNull();
  });

  // It is a picture of where the drag will end, for the pointer; the menu is
  // how the same choice is made without one.
  it("is hidden from assistive technology", () => {
    render(<AssistantDockTargets active="bottom-right" />);

    expect(targets().overlay).toHaveAttribute("aria-hidden");
  });
});

describe("AssistantLauncher at rest", () => {
  it("draws no targets until it is dragged", () => {
    render(<AssistantLauncher pendingCount={0} onClick={vi.fn()} onMove={vi.fn()} />);

    expect(screen.queryByTestId("assistant-dock-targets")).toBeNull();
  });
});
