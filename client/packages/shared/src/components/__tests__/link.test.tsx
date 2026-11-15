import { createTestRouter, TestRouter } from "@trenova/shared/testing/test-router";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { Link } from "../link";

function renderAt(path: string, ui: React.ReactNode) {
  render(<TestRouter router={createTestRouter(undefined, [path])}>{ui}</TestRouter>);
}

afterEach(cleanup);

describe("Link", () => {
  it("is current on the exact address it opens", () => {
    renderAt("/desk", <Link to="/desk">Today</Link>);

    expect(screen.getByRole("link", { name: "Today" })).toHaveAttribute("aria-current", "page");
  });

  it("is not current on the pages under it", () => {
    renderAt("/desk/scheduled", <Link to="/desk">Today</Link>);

    expect(screen.getByRole("link", { name: "Today" })).not.toHaveAttribute("aria-current");
  });

  it("adds no class of its own when current", () => {
    renderAt(
      "/desk",
      <Link to="/desk" className="rail-row">
        Today
      </Link>,
    );

    expect(screen.getByRole("link", { name: "Today" })).toHaveAttribute("class", "rail-row");
  });

  it("is current on the pages under it when the navigation asks for that", () => {
    renderAt(
      "/dash/loads/asg_1",
      <Link to="/dash/loads" activeOptions={{ exact: false }}>
        Loads
      </Link>,
    );

    expect(screen.getByRole("link", { name: "Loads" })).toHaveAttribute("data-status", "active");
  });
});
