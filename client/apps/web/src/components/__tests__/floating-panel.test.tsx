import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { FloatingPanel, FloatingPanelHeader } from "../floating-panel";

describe("FloatingPanel", () => {
  it("is a named region at the shipment panel's width unless told otherwise", () => {
    render(<FloatingPanel label="Brief and activity">Body</FloatingPanel>);

    const panel = screen.getByRole("complementary", { name: "Brief and activity" });

    expect(panel).toHaveTextContent("Body");
    expect(panel.style.getPropertyValue("--floating-panel-width")).toBe("372px");
  });

  it("takes the width its content is drawn for", () => {
    render(
      <FloatingPanel label="Brief" width={400}>
        Body
      </FloatingPanel>,
    );

    expect(
      screen
        .getByRole("complementary", { name: "Brief" })
        .style.getPropertyValue("--floating-panel-width"),
    ).toBe("400px");
  });

  it("says it is busy while it stands in for a panel that is loading", () => {
    render(
      <FloatingPanel aria-busy label="Loading the brief and activity">
        Shapes
      </FloatingPanel>,
    );

    expect(
      screen.getByRole("complementary", { name: "Loading the brief and activity" }),
    ).toHaveAttribute("aria-busy", "true");
  });
});

describe("FloatingPanelHeader", () => {
  it("shows what the panel holds beside one button that closes it", async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    render(
      <FloatingPanelHeader onClose={onClose}>
        <h2>Brief</h2>
      </FloatingPanelHeader>,
    );

    expect(screen.getByRole("heading", { name: "Brief" })).toBeInTheDocument();
    expect(screen.getAllByRole("button")).toHaveLength(1);

    await user.click(screen.getByRole("button", { name: "Close panel" }));

    expect(onClose).toHaveBeenCalledTimes(1);
  });
});
