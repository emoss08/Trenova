import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { WidgetCount } from "../widget-shell";

describe("WidgetCount", () => {
  afterEach(() => {
    cleanup();
  });

  it("shows a counted figure as it is", () => {
    render(<WidgetCount value={42} />);

    expect(screen.getByText("42")).not.toHaveAttribute("title");
  });

  // An estimate is never below ten thousand, so the pill reads "999+" and says
  // the figure is approximate rather than showing a number that reads as counted.
  it("says an estimated figure is approximate", () => {
    render(<WidgetCount value={1_250_000} exact={false} />);

    const pill = screen.getByText("999+");
    expect(pill).toHaveAttribute("title", "About 1.3M");
  });
});
