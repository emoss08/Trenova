import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { DeskColumns, DeskRailFold, DeskShell } from "../desk-shell";

/**
 * A folded column is gone, not merely thin: at zero width its links and
 * fields would otherwise still be in the tab order and read out by a screen
 * reader, so the fold takes the column out of both and the unfold puts it
 * back.
 */
describe("the Desk's folds", () => {
  it("takes the rail out of the tab order and the accessibility tree when folded", () => {
    const { rerender } = render(
      <DeskRailFold open={false}>
        <a href="/desk/t/a">A conversation</a>
      </DeskRailFold>,
    );

    const rail = document.querySelector("aside");
    expect(rail).toHaveAttribute("inert");
    expect(rail).toHaveAttribute("aria-hidden", "true");
    expect(rail).toHaveAttribute("data-state", "closed");

    rerender(
      <DeskRailFold open>
        <a href="/desk/t/a">A conversation</a>
      </DeskRailFold>,
    );
    expect(rail).not.toHaveAttribute("inert");
    expect(rail).toHaveAttribute("aria-hidden", "false");
    expect(screen.getByRole("link", { name: "A conversation" })).toBeInTheDocument();
  });

  it("does the same for the workspace", () => {
    const { rerender } = render(
      <DeskColumns conversation={<p>Thread</p>} workspace={<p>Work</p>} workspaceOpen={false} />,
    );

    const workspace = document.querySelector('[data-state="closed"]');
    expect(workspace).toHaveAttribute("inert");
    expect(screen.queryByRole("complementary", { name: "Workspace" })).toBeNull();

    rerender(<DeskColumns conversation={<p>Thread</p>} workspace={<p>Work</p>} workspaceOpen />);
    expect(screen.getByRole("complementary", { name: "Workspace" })).not.toHaveAttribute("inert");
  });
});

/**
 * The strip's left edge is where the rail comes back from. With the rail
 * open on a wide screen the rail's own header holds the fold, so the strip
 * hides its control there; folded, the control stands first.
 */
describe("the Desk's strip", () => {
  it("offers the rail at its left edge once the rail is folded", () => {
    const onShowRail = vi.fn();
    const { rerender } = render(
      <DeskShell rail={null} railOpen onShowRail={onShowRail} title="Today">
        <p>Room</p>
      </DeskShell>,
    );

    const show = screen.getByRole("button", { name: "Show the rail" });
    expect(show).toHaveClass("lg:hidden");

    rerender(
      <DeskShell rail={null} railOpen={false} onShowRail={onShowRail} title="Today">
        <p>Room</p>
      </DeskShell>,
    );
    expect(show).not.toHaveClass("lg:hidden");
    expect(show).toHaveAttribute("aria-keyshortcuts", "Meta+B Control+B");
    fireEvent.click(show);
    expect(onShowRail).toHaveBeenCalledTimes(1);
  });

  it("lights the strip in the agent's accent while it works", () => {
    render(
      <DeskShell
        rail={null}
        railOpen
        onShowRail={() => undefined}
        accent="oklch(0.5 0.1 200)"
        working
      >
        <p>Room</p>
      </DeskShell>,
    );

    const header = document.querySelector("header");
    expect(header).toHaveAttribute("data-working", "true");
    expect(header).toHaveClass("ui-agent-glow");
  });
});
