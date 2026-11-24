import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { SettlementHoldBadge } from "../settlement-hold-badge";

afterEach(cleanup);

describe("SettlementHoldBadge", () => {
  it("reads On hold with a dot and says why in its title", () => {
    const { container } = render(
      <SettlementHoldBadge reason="RateDispute" note="Waiting on the corrected rate" />,
    );

    const badge = screen.getByText("On hold").closest("[title]");
    expect(badge).toHaveAttribute("title", "On hold: Rate dispute, Waiting on the corrected rate");
    expect(container.querySelector('[data-slot="badge-dot"]')).not.toBeNull();
  });

  it("names the reason alone when there is no note", () => {
    render(<SettlementHoldBadge reason="MissingPaperwork" note="" />);

    expect(screen.getByText("On hold").closest("[title]")).toHaveAttribute(
      "title",
      "On hold: Missing paperwork",
    );
  });

  it("says it is held without a reason the reader may not see", () => {
    render(<SettlementHoldBadge reason={null} note="" />);

    expect(screen.getByText("On hold").closest("[title]")).toHaveAttribute(
      "title",
      "On hold until a person releases it",
    );
  });
});
