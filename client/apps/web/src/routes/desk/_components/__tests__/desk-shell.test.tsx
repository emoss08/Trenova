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
  it("keeps the strip of places and hides the list when the rail is folded", () => {
    const { rerender } = render(
      <DeskRailFold
        open={false}
        rail={<a href="/desk/t/a">A conversation</a>}
        strip={<a href="/desk">Today</a>}
      />,
    );

    const rail = document.querySelector("aside");
    expect(rail).toHaveAttribute("data-state", "collapsed");
    const list = document.querySelector('a[href="/desk/t/a"]')?.parentElement;
    expect(list).toHaveAttribute("inert");
    expect(list).toHaveAttribute("aria-hidden", "true");
    const strip = document.querySelector('a[href="/desk"]')?.parentElement;
    expect(strip).not.toHaveAttribute("inert");

    rerender(
      <DeskRailFold
        open
        rail={<a href="/desk/t/a">A conversation</a>}
        strip={<a href="/desk">Today</a>}
      />,
    );
    expect(rail).toHaveAttribute("data-state", "open");
    expect(list).not.toHaveAttribute("inert");
    expect(strip).toHaveAttribute("inert");
  });

  it("leaves the workspace out entirely when it is folded, and resizable when open", () => {
    const onResize = vi.fn();
    const { rerender } = render(
      <DeskColumns
        conversation={<p>Thread</p>}
        workspace={<p>Work</p>}
        workspaceOpen={false}
        workspaceSize={42}
        onWorkspaceResize={onResize}
      />,
    );
    expect(screen.queryByRole("complementary", { name: "Workspace" })).toBeNull();
    expect(screen.queryByRole("separator")).toBeNull();

    rerender(
      <DeskColumns
        conversation={<p>Thread</p>}
        workspace={<p>Work</p>}
        workspaceOpen
        workspaceSize={42}
        onWorkspaceResize={onResize}
      />,
    );
    expect(screen.getByRole("complementary", { name: "Workspace" })).toBeInTheDocument();
    expect(screen.getByRole("separator", { name: "Resize the workspace" })).toBeInTheDocument();
  });
});

/**
 * The strip's left edge is where the rail comes back from on a narrow screen,
 * where there is no rail beside the room; on a wide screen the rail's own
 * strip holds the unfold, so the control is kept for narrow screens only.
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
