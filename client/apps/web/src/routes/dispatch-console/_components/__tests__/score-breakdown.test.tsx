import type { DispatchScoreFactor } from "@/lib/graphql/dispatch-console";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { ScoreBreakdown } from "../score-breakdown";

afterEach(cleanup);

function factor(overrides: Partial<DispatchScoreFactor>): DispatchScoreFactor {
  return {
    key: "deadhead",
    label: "Empty miles",
    raw: 0.8,
    weight: 3,
    contribution: 24,
    detail: "12 empty miles to the pickup",
    ...overrides,
  } as DispatchScoreFactor;
}

describe("ScoreBreakdown", () => {
  it("shows the score against its scale", () => {
    render(<ScoreBreakdown score={88} factors={[factor({})]} />);

    expect(screen.getByText("88")).toBeInTheDocument();
    expect(screen.getByText("/ 100")).toBeInTheDocument();
  });

  // The whole point of the breakdown is that a dispatcher can audit the ranking. Every
  // factor must show both its label and the sentence explaining it.
  it("renders a readable explanation for every factor", () => {
    render(
      <ScoreBreakdown
        score={70}
        factors={[
          factor({ key: "deadhead", label: "Empty miles", detail: "12 empty miles" }),
          factor({
            key: "hosMargin",
            label: "Hours remaining",
            detail: "6h 00m of clock left after the trip",
            contribution: 18,
          }),
        ]}
      />,
    );

    expect(screen.getByText("Empty miles")).toBeInTheDocument();
    expect(screen.getByText("12 empty miles")).toBeInTheDocument();
    expect(screen.getByText("Hours remaining")).toBeInTheDocument();
    expect(screen.getByText("6h 00m of clock left after the trip")).toBeInTheDocument();
  });

  it("shows each factor's contribution to the total", () => {
    render(<ScoreBreakdown score={70} factors={[factor({ contribution: 24.4 })]} />);

    expect(screen.getByText("+24.4")).toBeInTheDocument();
  });

  // A candidate with no scorable data must say so rather than presenting a bare zero,
  // which would read as a judgement rather than as an absence of information.
  it("explains an empty breakdown instead of showing nothing", () => {
    render(<ScoreBreakdown score={0} factors={[]} />);

    expect(screen.getByText(/not enough data to score this pairing/i)).toBeInTheDocument();
  });
});
