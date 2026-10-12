import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { CardStatus } from "../record-card";

describe("CardStatus", () => {
  it("colours a pill that asks for a look by its own class", () => {
    render(<CardStatus tone="warn">Open</CardStatus>);

    expect(screen.getByText("Open")).toHaveClass("dk-rec-st", "dk-st-warn");
  });

  // The Desk's hold-to-confirm button is `.dk-hold`, 168px wide at the least
  // with a fill drawn behind it; a pill that shared the class was stretched
  // and filled along with it.
  it("never shares a class with the hold-to-confirm button", () => {
    render(
      <>
        {(["progress", "success", "ink", "warn", "muted", "draft"] as const).map((tone) => (
          <CardStatus key={tone} tone={tone}>
            {tone}
          </CardStatus>
        ))}
      </>,
    );

    for (const pill of document.querySelectorAll(".dk-rec-st")) {
      expect(pill).not.toHaveClass("dk-hold");
    }
  });
});
