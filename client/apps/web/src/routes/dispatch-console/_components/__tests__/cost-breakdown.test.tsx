import { formatCentsWhole } from "@trenova/shared/lib/utils";
import type { DispatchCandidate, DispatchEndMarket } from "@/lib/graphql/dispatch-console";
import { cleanup, render, screen } from "@testing-library/react";
import { formatUnixDayTimeRange, setUserDatePreferences } from "@trenova/shared/lib/date";
import { afterEach, describe, expect, it } from "vitest";
import { CostBreakdown } from "../cost-breakdown";

afterEach(() => {
  cleanup();
  setUserDatePreferences({});
});

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
      formatCentsWhole(candidate().costCents - 2_500),
    );
  });

  it("shows no preference line for a move no plan has priced", () => {
    render(<CostBreakdown candidate={candidate()} />);

    expect(screen.queryByText("Less what the best driver pays in preferences")).toBeNull();
  });
});

// DispatchEndMarket as dispatch_console.graphqls defines it: the forecast fields are set only
// when the source is Forecast, and valueCents is the drop value less the stay value.
const WINDOW_START = 1_793_793_600;
const WINDOW_END = WINDOW_START + 12 * 3_600;

const forecast: DispatchEndMarket = {
  source: "Forecast",
  valueCents: 1_550,
  dropValueCents: 1_550,
  stayValueCents: 0,
  stayAreaLabel: null,
  areaLabel: "Dallas, TX",
  windowStart: WINDOW_START,
  windowEnd: WINDOW_END,
  typicalLow: 3,
  typicalHigh: 5,
  booked: 1,
  driversExpected: 1,
  customers: [
    { customerId: "cus_1", name: "Acme Foods", expectedLoads: 3 },
    { customerId: "cus_2", name: "Red River Grain", expectedLoads: 1.4 },
  ],
};

const history: DispatchEndMarket = {
  source: "History",
  valueCents: 1_550,
  dropValueCents: 1_550,
  stayValueCents: 0,
  stayAreaLabel: null,
  areaLabel: null,
  windowStart: null,
  windowEnd: null,
  typicalLow: null,
  typicalHigh: null,
  booked: null,
  driversExpected: null,
  customers: [],
};

// toHaveTextContent collapses the element's whitespace, and Intl puts narrow spaces in a time.
const collapsed = (text: string) => text.replace(/\s+/g, " ");

function endMarketLine() {
  return screen.getByText("Where they end up").closest("li");
}

describe("CostBreakdown end market reason", () => {
  it("says what freight the forecast expects near the drop, from whom, and who else is expected", () => {
    setUserDatePreferences({ timezone: "America/Chicago" });

    render(<CostBreakdown candidate={candidate({ endMarket: forecast })} />);

    const line = endMarketLine();
    expect(line).toHaveTextContent(
      collapsed(
        `Near Dallas, TX: 3–5 loads usually appear ${formatUnixDayTimeRange(WINDOW_START, WINDOW_END)}.`,
      ),
    );
    expect(line).toHaveTextContent(
      "Mostly from Acme Foods (about 3) and Red River Grain (about 1.4).",
    );
    expect(line).toHaveTextContent("1 booked; 1 other driver expected.");
  });

  it("says up to the high end when the usual range starts at none", () => {
    setUserDatePreferences({ timezone: "America/Chicago" });

    render(
      <CostBreakdown
        candidate={candidate({
          endMarket: { ...forecast, typicalLow: 0, typicalHigh: 2, customers: [] },
        })}
      />,
    );

    expect(endMarketLine()).toHaveTextContent(
      collapsed(
        `Near Dallas, TX: up to 2 loads usually appear ${formatUnixDayTimeRange(WINDOW_START, WINDOW_END)}.`,
      ),
    );
  });

  it("says a single expected count and no other driver without a range or customers", () => {
    setUserDatePreferences({ timezone: "America/Chicago" });

    render(
      <CostBreakdown
        candidate={candidate({
          endMarket: {
            ...forecast,
            typicalLow: 2,
            typicalHigh: 2,
            booked: 2,
            driversExpected: 0,
            customers: [],
          },
        })}
      />,
    );

    const line = endMarketLine();
    expect(line).toHaveTextContent(
      collapsed(
        `Near Dallas, TX: 2 loads usually appear ${formatUnixDayTimeRange(WINDOW_START, WINDOW_END)}.`,
      ),
    );
    expect(line).toHaveTextContent("2 booked; no other driver expected.");
    expect(line).not.toHaveTextContent("Mostly from");
  });

  it("says a credit learned from past plans is what drivers free there have been worth", () => {
    render(<CostBreakdown candidate={candidate({ endMarket: history })} />);

    expect(endMarketLine()).toHaveTextContent(
      "Drivers free here have been worth this to past plans.",
    );
    expect(screen.queryByText(/usually appear/)).toBeNull();
  });

  it("says what staying was worth and what the move adds on top of it", () => {
    render(
      <CostBreakdown
        candidate={candidate({
          costLines: [{ component: "endMarket", cents: -1_500, soft: false }],
          endMarket: {
            ...forecast,
            valueCents: 1_500,
            dropValueCents: 4_000,
            stayValueCents: 2_500,
            stayAreaLabel: "Fort Worth, TX",
          },
        })}
      />,
    );

    const line = endMarketLine();
    expect(line).toHaveTextContent("−$15");
    expect(line).toHaveTextContent(
      "Staying near Fort Worth, TX was worth $25, so the move adds $15.",
    );
    expect(line).toHaveTextContent("1 booked; 1 other driver expected.");
  });

  it("says the move costs the plan when staying was worth more than ending at the drop", () => {
    render(
      <CostBreakdown
        candidate={candidate({
          costLines: [{ component: "endMarket", cents: 1_200, soft: false }],
          endMarket: {
            ...forecast,
            valueCents: -1_200,
            dropValueCents: 1_800,
            stayValueCents: 3_000,
            stayAreaLabel: "Houston, TX",
          },
        })}
      />,
    );

    const line = endMarketLine();
    expect(line).toHaveTextContent("$12");
    expect(line).not.toHaveTextContent("−$12");
    expect(line).toHaveTextContent(
      "Staying near Houston, TX was worth $30, more than ending here, so the move costs $12.",
    );
  });

  it("says staying was worth as much as the drop when the move nets nothing", () => {
    render(
      <CostBreakdown
        candidate={candidate({
          costLines: [{ component: "endMarket", cents: 0, soft: false }],
          endMarket: {
            ...forecast,
            valueCents: 0,
            dropValueCents: 2_000,
            stayValueCents: 2_000,
            stayAreaLabel: "Houston, TX",
          },
        })}
      />,
    );

    expect(endMarketLine()).toHaveTextContent(
      "Staying near Houston, TX was worth $20, as much as ending here.",
    );
  });

  it("says where they are when the forecast does not name the area the driver would leave", () => {
    render(
      <CostBreakdown
        candidate={candidate({
          endMarket: {
            ...forecast,
            valueCents: 1_500,
            dropValueCents: 4_000,
            stayValueCents: 2_500,
            stayAreaLabel: null,
          },
        })}
      />,
    );

    expect(endMarketLine()).toHaveTextContent(
      "Staying where they are was worth $25, so the move adds $15.",
    );
  });

  it("says nothing about staying when staying was worth nothing", () => {
    render(<CostBreakdown candidate={candidate({ endMarket: forecast })} />);

    expect(endMarketLine()).not.toHaveTextContent(/Staying/);
  });

  it("keeps a credit learned from past plans to its one sentence", () => {
    render(
      <CostBreakdown
        candidate={candidate({
          endMarket: {
            ...history,
            valueCents: 1_500,
            dropValueCents: 4_000,
            stayValueCents: 2_500,
          },
        })}
      />,
    );

    expect(endMarketLine()).toHaveTextContent(
      "Drivers free here have been worth this to past plans.",
    );
    expect(endMarketLine()).not.toHaveTextContent(/Staying/);
  });

  it("gives no reason when the move leaves the driver nowhere the plan values", () => {
    render(<CostBreakdown candidate={candidate({ endMarket: null })} />);

    expect(screen.getByText("Where they end up")).toBeInTheDocument();
    expect(screen.queryByText(/past plans|usually appear/)).toBeNull();
  });
});
