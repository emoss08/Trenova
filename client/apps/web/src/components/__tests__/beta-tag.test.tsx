import { Badge } from "@trenova/shared/components/ui/badge";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { BetaTag } from "../beta-tag";

afterEach(cleanup);

describe("beta tag", () => {
  it("shows the translated label set in capitals, with no icon", () => {
    render(<BetaTag />);

    const tag = screen.getByText("Beta");
    expect(tag.textContent).toBe("Beta");
    expect(tag.classList.contains("uppercase")).toBe(true);
    expect(tag.querySelector("svg")).toBeNull();
  });

  it("draws exactly the subtle neutral badge in capitals", () => {
    render(
      <>
        <BetaTag />
        <Badge variant="neutral" appearance="subtle" className="uppercase">
          Reference
        </Badge>
      </>,
    );

    expect(screen.getByText("Beta").className).toBe(screen.getByText("Reference").className);
  });

  it("keeps the placement class its caller passes", () => {
    render(<BetaTag className="ml-auto" />);

    expect(screen.getByText("Beta").classList.contains("ml-auto")).toBe(true);
  });
});
