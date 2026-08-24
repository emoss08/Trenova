import { describe, expect, it } from "vitest";
import {
  assignableOptions,
  decidingValue,
  hasSellPrice,
  marginTone,
  offerWindowLabel,
  optionIsPriced,
  shopOptionNote,
  shopStrategyExplanation,
  shopStrategyLabel,
  THIN_MARGIN_PERCENT,
} from "../shop";
import type { MarginVerdict, ShopOption, ShopResult } from "../../types/rate";

function verdict(overrides: Partial<MarginVerdict> = {}): MarginVerdict {
  return {
    amount: 600,
    percent: 30,
    payPercent: 70,
    floorApplies: false,
    belowFloor: false,
    ceilingApplies: false,
    abovePayCeiling: false,
    explanation: "",
    ...overrides,
  };
}

function option(overrides: Partial<ShopOption> = {}): ShopOption {
  return {
    carrierId: "car_1",
    carrierName: "Acme Freight",
    rank: 1,
    guideRank: 1,
    offerTtlSeconds: 900,
    outcome: "Rated",
    cost: 1400,
    currency: "USD",
    margin: verdict(),
    note: "",
    ...overrides,
  } as ShopOption;
}

function result(overrides: Partial<ShopResult> = {}): ShopResult {
  return {
    strategy: "LeastCost",
    options: [option()],
    warnings: [],
    sellTotal: 2000,
    ...overrides,
  } as ShopResult;
}

describe("shopStrategyLabel", () => {
  it("names each strategy in the words somebody picking one thinks in", () => {
    expect(shopStrategyLabel("LeastCost")).toBe("Cheapest");
    expect(shopStrategyLabel("BestMargin")).toBe("Best margin");
    expect(shopStrategyLabel("GuideRank")).toBe("Routing guide order");
    expect(shopStrategyLabel("FastestAccept")).toBe("Fastest to accept");
  });

  it("explains that best margin is not the same order as cheapest", () => {
    expect(shopStrategyExplanation("BestMargin")).toContain("not the same order");
  });
});

describe("optionIsPriced", () => {
  it("accepts every outcome that produced an amount", () => {
    expect(optionIsPriced(option({ outcome: "Rated" }))).toBe(true);
    expect(optionIsPriced(option({ outcome: "FormulaFallback" }))).toBe(true);
    expect(optionIsPriced(option({ outcome: "ManualOverride" }))).toBe(true);
  });

  // An unpriced carrier is shown so somebody can see it was considered, but it
  // can never be chosen — assigning it would settle the load for nothing.
  it("rejects the outcomes that produced no amount", () => {
    expect(optionIsPriced(option({ outcome: "NoRateFound" }))).toBe(false);
    expect(optionIsPriced(option({ outcome: "Error" }))).toBe(false);
  });
});

describe("assignableOptions", () => {
  it("keeps only the carriers a contract actually priced", () => {
    const shopped = result({
      options: [
        option({ carrierId: "car_priced", outcome: "Rated" }),
        option({ carrierId: "car_silent", outcome: "NoRateFound" }),
      ],
    });

    expect(assignableOptions(shopped).map((each) => each.carrierId)).toEqual(["car_priced"]);
  });

  it("has nothing to assign before a shopping run has happened", () => {
    expect(assignableOptions(undefined)).toEqual([]);
  });
});

describe("marginTone", () => {
  // A breach is the only state where somebody has a decision to make, so it is
  // the one that has to stand out.
  it("marks a margin below the contract's floor as a breach", () => {
    expect(marginTone(verdict({ percent: 5, belowFloor: true }), true)).toBe("breach");
  });

  it("marks carrier pay above the contract's ceiling as a breach too", () => {
    expect(marginTone(verdict({ percent: 20, abovePayCeiling: true }), true)).toBe("breach");
  });

  it("marks a thin margin even where no contract set a floor", () => {
    expect(marginTone(verdict({ percent: THIN_MARGIN_PERCENT - 1 }), true)).toBe("thin");
  });

  it("treats a margin exactly at the display threshold as healthy", () => {
    expect(marginTone(verdict({ percent: THIN_MARGIN_PERCENT }), true)).toBe("healthy");
  });

  // Quoting cost before quoting the customer is ordinary, and colouring it as a
  // problem would train people to ignore the colour.
  it("is neutral rather than alarming when the load has no sell price yet", () => {
    expect(marginTone(verdict({ percent: 0 }), false)).toBe("unknown");
  });

  it("does not call a loss thin when a contract already named it a breach", () => {
    expect(marginTone(verdict({ percent: -20, belowFloor: true }), true)).toBe("breach");
  });
});

describe("shopOptionNote", () => {
  // The server knows why the option ranked where it did. Paraphrasing here
  // would put two explanations of one decision in front of the same person.
  it("prefers whatever the server said", () => {
    const note = "margin of 5% is below the 15% this contract requires";

    expect(shopOptionNote(option({ note }), true)).toBe(note);
  });

  it("says why an unpriced carrier cannot be used", () => {
    expect(shopOptionNote(option({ outcome: "NoRateFound" }), true)).toContain("No carrier");
  });

  it("says why margin is blank when the load has no sell price", () => {
    expect(shopOptionNote(option(), false)).toContain("no sell price");
  });

  it("says nothing about an ordinary priced option", () => {
    expect(shopOptionNote(option(), true)).toBe("");
  });
});

describe("decidingValue", () => {
  it("shows the cost when ranking by cost", () => {
    expect(decidingValue(option({ cost: 1400 }), "LeastCost")).toBe("1400");
  });

  it("shows the margin share when ranking by margin", () => {
    expect(decidingValue(option({ margin: verdict({ percent: 30 }) }), "BestMargin")).toBe("30%");
  });

  it("shows the guide's own rank when keeping its order", () => {
    expect(decidingValue(option({ guideRank: 2 }), "GuideRank")).toBe("#2");
  });

  // A carrier from an explicit shortlist has no guide rank, and printing "#0"
  // would read as a rank rather than as an absence.
  it("shows nothing for a carrier that came from no guide", () => {
    expect(decidingValue(option({ guideRank: 0 }), "GuideRank")).toBe("—");
  });

  it("shows the offer window when ranking by how fast a carrier can accept", () => {
    expect(decidingValue(option({ offerTtlSeconds: 900 }), "FastestAccept")).toBe("15 min");
  });
});

describe("offerWindowLabel", () => {
  it("reads short windows in minutes", () => {
    expect(offerWindowLabel(900)).toBe("15 min");
  });

  it("reads long windows in hours", () => {
    expect(offerWindowLabel(5400)).toBe("1.5 hr");
  });

  it("treats an hour as hours rather than sixty minutes", () => {
    expect(offerWindowLabel(3600)).toBe("1 hr");
  });

  // A guide that never set a window has not set one. Printing "0 min" would
  // look like an offer that has already expired.
  it("reads an unset window as unknown, not as expired", () => {
    expect(offerWindowLabel(0)).toBe("—");
  });
});

describe("hasSellPrice", () => {
  it("is true when the load has been priced for the customer", () => {
    expect(hasSellPrice(result({ sellTotal: 2000 }))).toBe(true);
  });

  // A shipment quoted at nothing is not one sold for nothing, and every margin
  // shown against it would read as a hundred percent.
  it("is false when the sell total is zero", () => {
    expect(hasSellPrice(result({ sellTotal: 0 }))).toBe(false);
  });

  it("is false when the sell total is absent entirely", () => {
    expect(hasSellPrice(result({ sellTotal: null }))).toBe(false);
  });

  it("is false before a shopping run has happened", () => {
    expect(hasSellPrice(undefined)).toBe(false);
  });
});
