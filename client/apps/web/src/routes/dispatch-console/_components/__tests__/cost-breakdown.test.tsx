import type { DispatchCandidate } from "@/lib/graphql/dispatch-console";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { CostBreakdown } from "../cost-breakdown";
import { formatCents } from "../cost-vocabulary";

afterEach(cleanup);

// DispatchCandidate as dispatch_console.graphqls defines it: cost lines in cents.
function candidate(overrides: Partial<DispatchCandidate> = {}): DispatchCandidate {
  return {
    workerId: "wrk_1",
    workerName: "Dana Ortiz",
    moveId: "smv_1",
    costCents: 14_250,
    costLines: [
      { component: "emptyMiles", cents: 9_800, soft: false },
      { component: "idleWait", cents: 3_000, soft: false },
      { component: "haulFit", cents: 3_000, soft: true },
      { component: "endMarket", cents: -1_550, soft: false },
    ],
    deadheadMiles: 49,
    minutesOfSlack: 30,
    slackKnown: true,
    hosKnown: true,
    deliveryLateMinutes: 0,
    restMs: 0,
    ...overrides,
  } as DispatchCandidate;
}

describe("CostBreakdown", () => {
  it("states what the pairing costs and what the move is worth", () => {
    render(<CostBreakdown candidate={candidate()} coveragePriceCents={90_000} />);

    expect(screen.getByText("$143")).toBeInTheDocument();
    expect(screen.getByText("of $900 the move is worth")).toBeInTheDocument();
  });

  it("lists each cost with the fact behind it, and a credit as a negative line", () => {
    render(<CostBreakdown candidate={candidate()} />);

    expect(screen.getByText("Empty miles")).toBeInTheDocument();
    expect(screen.getByText("49 mi")).toBeInTheDocument();
    expect(screen.getByText("$98")).toBeInTheDocument();
    expect(screen.getByText("Where they end up")).toBeInTheDocument();
    expect(screen.getByText("−$16")).toBeInTheDocument();
  });

  it("keeps preferences apart from the costs that decide coverage", () => {
    render(<CostBreakdown candidate={candidate()} />);

    const preferences = screen.getByText("Preferences").parentElement;
    expect(preferences).toHaveTextContent("Haul fit");
    expect(preferences).not.toHaveTextContent("Empty miles");
  });

  it("says when a pairing costs nothing extra", () => {
    render(<CostBreakdown candidate={candidate({ costCents: 0, costLines: [] })} />);

    expect(screen.getByText("This pairing costs nothing extra.")).toBeInTheDocument();
  });

  it("subtracts what the best driver pays in preferences, so the total is the plan's cost", () => {
    render(
      <CostBreakdown
        candidate={candidate()}
        coveragePriceCents={90_000}
        preferenceFloorCents={2_500}
      />,
    );

    expect(
      screen.getByText("Less what the best driver pays in preferences").nextElementSibling,
    ).toHaveTextContent("−$25");
    expect(screen.getByText("Cost to cover").nextElementSibling).toHaveTextContent(
      formatCents(candidate().costCents - 2_500),
    );
  });

  it("shows no preference line for a move no plan has priced", () => {
    render(<CostBreakdown candidate={candidate()} />);

    expect(screen.queryByText("Less what the best driver pays in preferences")).toBeNull();
  });
});
