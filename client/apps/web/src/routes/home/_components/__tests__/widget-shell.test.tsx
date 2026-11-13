import { cleanup, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it } from "vitest";
import { WidgetEditingProvider, WidgetShell, type WidgetShellProps } from "../widget-shell";

function renderShell(props: Partial<WidgetShellProps> = {}, editing = false) {
  return render(
    <MemoryRouter>
      <WidgetEditingProvider value={editing}>
        <WidgetShell title="Unassigned loads" {...props}>
          <p>Body</p>
        </WidgetShell>
      </WidgetEditingProvider>
    </MemoryRouter>,
  );
}

afterEach(cleanup);

describe("WidgetShell", () => {
  it("is a named panel with its title as the heading and a counted figure beside it", () => {
    renderShell({ count: 1709 });

    const panel = screen.getByRole("region", { name: "Unassigned loads" });
    expect(panel).toContainElement(screen.getByRole("heading", { name: "Unassigned loads" }));
    expect(screen.getByText("1,709")).toBeInTheDocument();
  });

  it("draws no count for an empty list", () => {
    renderShell({ count: 0 });

    expect(screen.queryByText("0")).not.toBeInTheDocument();
  });

  // An estimate is never shown as a counted number: it reads as approximate, in words.
  it("says an estimated figure is approximate, ahead of the widget's own note", () => {
    renderShell({ count: 1_250_000, countExact: false, hint: "$40,000.00 waiting" });

    expect(screen.getByText("About 1.3M · $40,000.00 waiting")).toBeInTheDocument();
    expect(screen.queryByText("1,250,000")).not.toBeInTheDocument();
  });

  it("offers one way through to the full view, named for where it goes", () => {
    renderShell({ href: "/shipment-management/shipments", hrefLabel: "Open dispatch board" });

    expect(screen.getByRole("link", { name: "Open dispatch board" })).toHaveAttribute(
      "href",
      "/shipment-management/shipments",
    );
  });

  it("falls back to View all when the destination is not named", () => {
    renderShell({ href: "/reports" });

    expect(screen.getByRole("link", { name: "View all" })).toHaveAttribute("href", "/reports");
  });

  it("gives up its header and footer while the canvas is being edited, keeping the body", () => {
    renderShell({ count: 3, href: "/reports" }, true);

    expect(screen.getByText("Body")).toBeInTheDocument();
    expect(screen.queryByRole("heading")).not.toBeInTheDocument();
    expect(screen.queryByRole("link")).not.toBeInTheDocument();
  });
});
