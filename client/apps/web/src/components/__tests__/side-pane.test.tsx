import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { SidePane } from "../side-pane";

describe("SidePane", () => {
  it("is a named region at its open width while open", () => {
    render(
      <SidePane open aria-label="Move 1042" width="26rem">
        <button type="button">Assign</button>
      </SidePane>,
    );

    const pane = screen.getByRole("complementary", { name: "Move 1042" });
    expect(pane).not.toHaveAttribute("inert");
    expect(pane).toHaveAttribute("aria-hidden", "false");
    expect(pane.style.getPropertyValue("--side-pane-width")).toBe("26rem");
    expect(pane).toHaveClass("w-(--side-pane-width)");
  });

  it("folds to nothing and takes its content out of reach when closed", () => {
    render(
      <SidePane open={false} aria-label="Inspector">
        <button type="button">Assign</button>
      </SidePane>,
    );

    const pane = screen.getByRole("complementary", { hidden: true });
    expect(pane).toHaveAttribute("inert");
    expect(pane).toHaveAttribute("aria-hidden", "true");
    expect(pane).toHaveClass("w-0");
    expect(screen.queryByRole("button", { name: "Assign" })).not.toBeInTheDocument();
  });

  it("is the dispatch inspector's width unless told otherwise", () => {
    render(<SidePane open aria-label="Inspector" />);

    expect(
      screen
        .getByRole("complementary", { name: "Inspector" })
        .style.getPropertyValue("--side-pane-width"),
    ).toBe("25rem");
  });
});
