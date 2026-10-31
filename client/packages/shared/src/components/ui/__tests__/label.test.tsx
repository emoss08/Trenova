import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { Label } from "@trenova/shared/components/ui/label";

afterEach(cleanup);

describe("Label", () => {
  // `font-mono` is Geist Mono, kept for figures that align in a column. A label is text, so
  // it draws in the body font; with `font-mono` every form label in the app read as code.
  it("draws in the body font, not the figure font", () => {
    render(<Label htmlFor="base-url">Base URL</Label>);

    expect(screen.getByText("Base URL").className.split(/\s+/)).not.toContain("font-mono");
  });

  it("names the control it is for", () => {
    render(
      <>
        <Label htmlFor="base-url">Base URL</Label>
        <input id="base-url" />
      </>,
    );

    expect(screen.getByLabelText("Base URL").id).toBe("base-url");
  });
});
