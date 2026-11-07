import type { DispatchMovePlanPrice } from "@/lib/graphql/dispatch-console";
import { cleanup, render, screen, within } from "@testing-library/react";
import { formatRelativeTime } from "@trenova/shared/i18n/format";
import { afterEach, describe, expect, it } from "vitest";
import { PlanPricePanel } from "../plan-price-panel";

afterEach(cleanup);

const GENERATED_AT = 1_800_000_000;

// Written from DispatchMovePlanPrice in dispatch_console.graphqls: workerId is set when the
// plan covers the move, uncoveredReason when it does not.
function price(overrides: Partial<DispatchMovePlanPrice> = {}): DispatchMovePlanPrice {
  return {
    workerId: "wrk_1",
    workerName: "Ana Ruiz",
    costCents: 14_200,
    preferenceFloorCents: 0,
    coveragePriceCents: 90_000,
    driverValueCents: 3_100,
    uncoveredReason: null,
    alternatives: [],
    generatedAt: GENERATED_AT,
    ...overrides,
  };
}

function uncovered(overrides: Partial<DispatchMovePlanPrice>): DispatchMovePlanPrice {
  return price({ workerId: null, driverValueCents: 0, ...overrides });
}

function renderPanel(value: DispatchMovePlanPrice, nowSeconds = GENERATED_AT + 30) {
  render(<PlanPricePanel price={value} nowSeconds={nowSeconds} />);
}

describe("PlanPricePanel on a move the plan covers", () => {
  it("says what the move is worth, who the plan chose and what they cost", () => {
    renderPanel(price());

    expect(screen.getByText("Worth $900 to cover")).toBeInTheDocument();
    expect(screen.getByText("Planned driver").nextElementSibling).toHaveTextContent("Ana Ruiz");
    expect(screen.getByText("Cost to cover").nextElementSibling).toHaveTextContent("$142");
  });

  it("says what the planned driver is worth to the whole plan", () => {
    renderPanel(price());

    expect(screen.getByText("Ana Ruiz is worth $31 to the plan")).toBeInTheDocument();
  });

  it("leaves out the driver's worth when losing them would cost the plan nothing", () => {
    renderPanel(price({ driverValueCents: 0 }));

    expect(screen.queryByText(/to the plan/)).toBeNull();
  });

  it("lists every other driver with the exact extra cost of choosing them, in the plan's order", () => {
    renderPanel(
      price({
        alternatives: [
          { workerId: "wrk_2", workerName: "Ben Cho", extraCents: 1_200 },
          { workerId: "wrk_3", workerName: "Cy Diaz", extraCents: 4_800 },
        ],
      }),
    );

    const items = within(screen.getByRole("list", { name: "Other drivers" })).getAllByRole(
      "listitem",
    );
    expect(items.map((item) => item.textContent)).toEqual([
      "Choosing Ben Cho instead adds $12",
      "Choosing Cy Diaz instead adds $48",
    ]);
  });

  it("draws no list when the plan weighed nobody else", () => {
    renderPanel(price({ alternatives: [] }));

    expect(screen.queryByRole("list", { name: "Other drivers" })).toBeNull();
  });
});

describe("PlanPricePanel on a move the plan left uncovered", () => {
  it.each([
    {
      reason: "NotWorthIt" as const,
      workerName: "Dana Ortiz",
      headline: "Not worth covering",
      sentence:
        "The cheapest driver, Dana Ortiz, costs $610, more than the $450 it is worth to cover.",
    },
    {
      reason: "NotWorthIt" as const,
      workerName: "",
      headline: "Not worth covering",
      sentence: "Every driver costs more than the $450 it is worth to cover.",
    },
    {
      reason: "Outbid" as const,
      workerName: "Dana Ortiz",
      headline: "Drivers worth more elsewhere",
      sentence:
        "Its drivers are worth more on other moves. The cheapest, Dana Ortiz, would cost $610 against the $450 it is worth to cover.",
    },
    {
      reason: "Outbid" as const,
      workerName: "",
      headline: "Drivers worth more elsewhere",
      sentence: "Its drivers are worth more on other moves than the $450 it is worth to cover.",
    },
    {
      reason: "Disqualified" as const,
      workerName: "Dana Ortiz",
      headline: "Nearest drivers disqualified",
      sentence: "The nearest driver, Dana Ortiz, can't take it.",
    },
    {
      reason: "Disqualified" as const,
      workerName: "",
      headline: "Nearest drivers disqualified",
      sentence: "The nearest drivers can't take it.",
    },
    {
      reason: "BeyondDeadhead" as const,
      workerName: "Dana Ortiz",
      headline: "Past the empty-mile limit",
      sentence: "The nearest driver, Dana Ortiz, is past the empty-mile limit.",
    },
    {
      reason: "BeyondDeadhead" as const,
      workerName: "",
      headline: "Past the empty-mile limit",
      sentence: "The nearest drivers are past the empty-mile limit.",
    },
    {
      reason: "LocationUnknown" as const,
      workerName: "",
      headline: "No driver location known",
      sentence: "No driver near the pickup has a known location.",
    },
    {
      reason: "NoCandidate" as const,
      workerName: "",
      headline: "No eligible driver",
      sentence: "No driver could take it.",
    },
  ])(
    "says why for $reason with driver '$workerName'",
    ({ reason, workerName, headline, sentence }) => {
      renderPanel(
        uncovered({
          uncoveredReason: reason,
          workerName,
          costCents: 61_000,
          coveragePriceCents: 45_000,
        }),
      );

      expect(screen.getByText(headline)).toBeInTheDocument();
      expect(screen.getByText(sentence)).toBeInTheDocument();
      expect(screen.queryByText(/Worth .* to cover$/)).toBeNull();
      expect(screen.queryByText("Planned driver")).toBeNull();
    },
  );

  it("still says the move was left uncovered when the plan gave no reason", () => {
    renderPanel(uncovered({ uncoveredReason: null, workerName: "" }));

    expect(screen.getByText("Left uncovered")).toBeInTheDocument();
    expect(screen.getByText("The last plan left it uncovered.")).toBeInTheDocument();
  });
});

describe("PlanPricePanel age", () => {
  it("says the plan priced the move just now within the last minute", () => {
    renderPanel(price(), GENERATED_AT + 30);

    expect(screen.getByText("Priced just now")).toBeInTheDocument();
  });

  it("says how long ago an older plan priced the move", () => {
    renderPanel(price(), GENERATED_AT + 7_200);

    expect(screen.getByText(`Priced ${formatRelativeTime(-7_200, "narrow")}`)).toBeInTheDocument();
  });
});
