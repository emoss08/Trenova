import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type {
  LanePricingGroup,
  RateAgreement,
  RateAgreementAccessorial,
  RateAgreementFuelBinding,
  RateAgreementRule,
} from "@trenova/shared/types/rate";
import { describe, expect, it } from "vitest";
import {
  accessorialTerms,
  agreementReferences,
  agreementStage,
  decimal,
  fuelTerms,
  guardrails,
  laneTerms,
  lifecycleSteps,
  pricingGroups,
  tariffTerms,
  termSpan,
  termState,
} from "../rate-agreement-card";

/** Fills a message's numbered places the way the catalog would, plurals read as their `other` arm. */
const t = ((message: string, ...args: unknown[]) =>
  message
    .replace(/\{(\d+), plural, one \{[^}]*\} other \{([^}]*)\}\}/gu, (_, index, other: string) =>
      other.replace("#", String(args[Number(index)])),
    )
    .replace(/\{(\d+)\}/gu, (_, index) => String(args[Number(index)]))) as TranslateFn;

const DAY = 86_400;
/** Oct 11, 2026 12:00 UTC. */
const NOW = 1_791_720_000;

function lane(overrides: Partial<RateAgreementRule> = {}): RateAgreementRule {
  return {
    id: "rule_1",
    label: "",
    status: "Active",
    originScopeType: "Any",
    originScopeValue: "",
    originCity: "",
    destinationScopeType: "Any",
    destinationScopeValue: "",
    destinationCity: "",
    direction: "Directional",
    formulaTemplateId: "ft_mile",
    inputs: { rate: { $q: "2.1", u: "$/mi" } },
    daysOfWeek: 0,
    hazmatOnly: false,
    tempControlOnly: false,
    priority: 0,
    effectiveFrom: NOW - 30 * DAY,
    ...overrides,
  } as RateAgreementRule;
}

function agreement(overrides: Partial<RateAgreement> = {}): RateAgreement {
  return {
    id: "ra_1",
    partyType: "Customer",
    customerId: "cus_1",
    carrierId: null,
    code: "ACME-2026",
    name: "Acme dry van contract",
    description: "",
    agreementType: "Contract",
    status: "Active",
    contractRef: "",
    priority: 0,
    effectiveFrom: NOW - 100 * DAY,
    effectiveTo: NOW + 265 * DAY,
    autoRenew: false,
    renewalNoticeDays: 30,
    currency: "USD",
    inputs: {},
    billToCustomerId: null,
    marginFloorPercent: null,
    maxPayPercentOfSell: null,
    submittedById: "usr_1",
    submittedAt: NOW - 110 * DAY,
    approvedById: "usr_2",
    approvedAt: NOW - 105 * DAY,
    reviewComment: "",
    currentVersionNumber: 3,
    createdAt: NOW - 120 * DAY,
    rules: [lane()],
    accessorials: [],
    fuelBinding: null,
    versions: [],
    ...overrides,
  } as RateAgreement;
}

describe("decimal", () => {
  it("reads a decimal the API sends as a string, and nothing for one that is not there", () => {
    expect(decimal("2.1025")).toBe(2.1025);
    expect(decimal(0)).toBe(0);
    expect(decimal("")).toBeNull();
    expect(decimal(null)).toBeNull();
    expect(decimal("n/a")).toBeNull();
  });
});

describe("agreementStage", () => {
  it("reads only an active agreement as pricing freight, and a held or finished one as stopped", () => {
    expect(agreementStage("Active")).toMatchObject({
      tone: "success",
      mark: "check",
      label: "Active",
    });
    expect(agreementStage("InReview")).toMatchObject({ tone: "progress", label: "In review" });
    expect(agreementStage("Draft")).toMatchObject({ tone: "draft" });
    expect(agreementStage("Suspended")).toMatchObject({ tone: "warn", mark: "pause" });
    expect(agreementStage("Expired")).toMatchObject({ tone: "muted" });
    expect(agreementStage("Archived")).toMatchObject({ tone: "muted", mark: "pause" });
  });
});

describe("termState", () => {
  const term = (overrides: Partial<RateAgreement> = {}) => agreement(overrides);

  it("reads an active term as pricing freight, or inside a notice that needs someone", () => {
    expect(termState(term(), NOW)).toBe("live");
    expect(termState(term({ effectiveTo: null }), NOW)).toBe("live");
    expect(termState(term({ effectiveTo: NOW + 12 * DAY }), NOW)).toBe("notice");
    expect(termState(term({ effectiveTo: NOW + 12 * DAY, autoRenew: true }), NOW)).toBe("live");
  });

  it("reads a term not yet in force, held, or over for what it is", () => {
    expect(termState(term({ effectiveFrom: NOW + DAY }), NOW)).toBe("pending");
    expect(termState(term({ status: "InReview" }), NOW)).toBe("pending");
    expect(termState(term({ status: "Draft" }), NOW)).toBe("pending");
    expect(termState(term({ status: "Suspended" }), NOW)).toBe("held");
    expect(termState(term({ status: "Expired" }), NOW)).toBe("ended");
    expect(termState(term({ effectiveTo: NOW - DAY }), NOW)).toBe("ended");
  });
});

describe("termSpan", () => {
  it("places today along the term, and the renewal notice at its end", () => {
    const span = termSpan(
      { effectiveFrom: NOW - 100 * DAY, effectiveTo: NOW + 100 * DAY, renewalNoticeDays: 20 },
      NOW,
    );
    expect(span).toEqual({ today: 0.5, notice: 0.1, when: "during" });
  });

  it("holds today to the bar's ends before and after the term", () => {
    const term = { effectiveFrom: NOW + DAY, effectiveTo: NOW + 11 * DAY, renewalNoticeDays: 30 };
    expect(termSpan(term, NOW)).toEqual({ today: 0, notice: 1, when: "before" });
    expect(termSpan(term, NOW + 20 * DAY)?.when).toBe("after");
  });

  it("has no bar for a term with no end", () => {
    expect(
      termSpan({ effectiveFrom: NOW, effectiveTo: null, renewalNoticeDays: 30 }, NOW),
    ).toBeNull();
  });
});

describe("lifecycleSteps", () => {
  it("has walked every step once approved, with who took each", () => {
    expect(lifecycleSteps(agreement(), t)).toEqual([
      {
        key: "drafted",
        label: "Drafted",
        at: NOW - 120 * DAY,
        byId: null,
        state: "done",
        tone: null,
      },
      {
        key: "submitted",
        label: "Submitted for review",
        at: NOW - 110 * DAY,
        byId: "usr_1",
        state: "done",
        tone: null,
      },
      {
        key: "approved",
        label: "Approved",
        at: NOW - 105 * DAY,
        byId: "usr_2",
        state: "done",
        tone: null,
      },
    ]);
  });

  it("stands at the draft, or waits on approval while in review", () => {
    const draft = lifecycleSteps(
      agreement({ status: "Draft", submittedAt: null, approvedAt: null }),
      t,
    );
    expect(draft.map((step) => step.state)).toEqual(["now", "todo", "todo"]);

    const review = lifecycleSteps(agreement({ status: "InReview", approvedAt: null }), t);
    expect(review.map((step) => step.state)).toEqual(["done", "now", "todo"]);
  });

  it("adds how an agreement ended", () => {
    const steps = lifecycleSteps(agreement({ status: "Suspended" }), t);
    expect(steps.at(-1)).toMatchObject({
      key: "ended",
      label: "Suspended",
      state: "now",
      tone: "held",
    });
    expect(lifecycleSteps(agreement({ status: "Archived" }), t).at(-1)?.tone).toBe("closed");
  });
});

describe("tariffTerms and laneTerms", () => {
  it("lists the tariff input values a lane sets, named as the form names them and in their unit", () => {
    expect(
      tariffTerms(
        {
          rate: { $q: "2.1025", u: "$/mi" },
          minimumCharge: { $q: "250", u: "$" },
          fuelPercent: { $q: "18", u: "%" },
          breaks: [
            { from: { $q: "0", u: "lb" }, rate: { $q: "42", u: "$/cwt" }, minimum: null },
            { from: { $q: "500", u: "lb" }, rate: { $q: "35.5", u: "$/cwt" }, minimum: null },
          ],
          hazmat: true,
          unset: undefined,
        },
        "USD",
        t,
      ).map((term) => [term.label, term.value]),
    ).toEqual([
      ["Rate", "$2.1025/mi"],
      ["Minimum charge", "$250.00"],
      ["Fuel percent", "18%"],
      ["Breaks", "2 items"],
      ["Hazmat", "Yes"],
    ]);
    expect(tariffTerms({}, "USD", t)).toEqual([]);
  });

  it("shows money in the lane's currency", () => {
    expect(tariffTerms({ minimumCharge: { $q: "250", u: "$" } }, "CAD", t)[0]?.value).toBe(
      "CA$250.00",
    );
  });

  it("states which loads a lane applies to and which way it runs", () => {
    expect(
      laneTerms(
        lane({
          minWeight: 5000,
          maxWeight: 20000,
          maxDistance: 500,
          hazmatOnly: true,
          direction: "Bidirectional",
        }),
        t,
      ),
    ).toEqual(["5,000–20,000 lb", "Up to 500 mi", "Hazmat only", "Both ways"]);
    expect(laneTerms(lane(), t)).toEqual([]);
  });
});

describe("accessorialTerms and fuelTerms", () => {
  const charge = (overrides: Partial<RateAgreementAccessorial>) =>
    ({
      accessorialChargeId: "acc_1",
      method: "Flat",
      amount: 75,
      waived: false,
      autoApply: false,
      condition: null,
      ...overrides,
    }) as RateAgreementAccessorial;

  it("states what each accessorial charges, and what changes that", () => {
    expect(accessorialTerms(charge({}), "USD", t)).toEqual(["$75.00"]);
    expect(
      accessorialTerms(
        charge({
          method: "PerUnit",
          rateUnit: "Hour",
          amount: 45,
          freeUnits: 2,
          maxAmount: 300,
          autoApply: true,
        }),
        "USD",
        t,
      ),
    ).toEqual(["$45.00 per hour", "2 free", "Up to $300.00", "Added automatically"]);
    expect(
      accessorialTerms(
        charge({ autoApply: true, condition: 'condition "Residential"\n\nshipment.hazmat' }),
        "USD",
        t,
      ),
    ).toEqual(["$75.00", "Added automatically", "When its condition holds"]);
    expect(accessorialTerms(charge({ method: "Percentage", amount: 3 }), "USD", t)).toEqual(["3%"]);
    expect(accessorialTerms(charge({ waived: true }), "USD", t)).toEqual(["Waived"]);
  });

  it("states what the agreement does to its fuel program", () => {
    const binding = (overrides: Partial<RateAgreementFuelBinding>) =>
      ({
        fuelSurchargeProgramId: "fsp_1",
        waived: false,
        ...overrides,
      }) as RateAgreementFuelBinding;

    expect(fuelTerms(binding({}), "USD", t)).toEqual(["The program's own terms"]);
    expect(fuelTerms(binding({ pegPriceOverride: 3.5, capAmount: 0.6 }), "USD", t)).toEqual([
      "Peg $3.50",
      "Capped at $0.60",
    ]);
    expect(fuelTerms(binding({ waived: true }), "USD", t)).toEqual(["Fuel surcharge waived"]);
  });
});

describe("guardrails", () => {
  it("lists only the margin kept, and only where one is set", () => {
    expect(guardrails(agreement(), t)).toEqual([]);
    expect(
      guardrails(agreement({ marginFloorPercent: 12.5, maxPayPercentOfSell: 85 }), t).map((row) => [
        row.label,
        row.value,
      ]),
    ).toEqual([
      ["Margin floor", "12.5%"],
      ["Pays at most", "85% of sell"],
    ]);
  });

  it("says a carrier agreement keeps margins without showing them to a role that withholds them", () => {
    expect(
      guardrails(
        agreement({ partyType: "Carrier", marginFloorPercent: null, maxPayPercentOfSell: null }),
        t,
        false,
      ).map((row) => [row.label, row.value]),
    ).toEqual([
      ["Margin floor", "Hidden"],
      ["Pays at most", "Hidden"],
    ]);
    expect(guardrails(agreement({ partyType: "Customer" }), t, false)).toEqual([]);
  });
});

describe("agreementReferences", () => {
  it("gathers every id the card names, once each and sorted", () => {
    const refs = agreementReferences(
      agreement({
        billToCustomerId: "cus_0",
        accessorials: [
          { accessorialChargeId: "acc_2" },
          { accessorialChargeId: "acc_1" },
        ] as RateAgreementAccessorial[],
        fuelBinding: { fuelSurchargeProgramId: "fsp_1" } as RateAgreementFuelBinding,
      }),
      [lane(), lane({ id: "rule_2", formulaTemplateId: "ft_ltl" }), lane({ id: "rule_3" })],
      ["usr_3", null, "usr_1"],
    );

    expect(refs).toEqual({
      templates: ["ft_ltl", "ft_mile"],
      charges: ["acc_1", "acc_2"],
      programs: ["fsp_1"],
      customers: ["cus_0", "cus_1"],
      carriers: [],
      users: ["usr_1", "usr_2", "usr_3"],
    });
  });
});

describe("pricingGroups", () => {
  const group = (overrides: Partial<LanePricingGroup>): LanePricingGroup => ({
    formulaTemplateId: "ft_mile",
    formulaTemplateName: "Per mile",
    currency: "USD",
    lanes: 1,
    ...overrides,
  });

  it("names each tariff that prices lanes, with its share of the lanes, most lanes first", () => {
    const groups = pricingGroups(
      {
        count: 8,
        groups: [
          group({ lanes: 3 }),
          group({ formulaTemplateId: "ft_ltl", formulaTemplateName: "LTL classes", lanes: 5 }),
        ],
      },
      { rows: 5, currency: "USD" },
      t,
    );

    expect(groups).toEqual([
      { key: "ft_ltl:USD", label: "LTL classes", lanes: 5, share: 5 / 8 },
      { key: "ft_mile:USD", label: "Per mile", lanes: 3, share: 3 / 8 },
    ]);
  });

  it("says a currency other than the agreement's", () => {
    const [mile] = pricingGroups(
      { count: 3, groups: [group({ lanes: 3, currency: "CAD" })] },
      { rows: 5, currency: "USD" },
      t,
    );

    expect(mile).toMatchObject({ label: "Per mile · CAD", share: 1 });
  });

  it("folds the tariffs past its rows into one, keeping the lanes they price", () => {
    const groups = pricingGroups(
      {
        count: 10,
        groups: [4, 3, 2, 1].map((lanes) =>
          group({ formulaTemplateId: `ft_${lanes}`, formulaTemplateName: `T${lanes}`, lanes }),
        ),
      },
      { rows: 3, currency: "USD" },
      t,
    );

    expect(groups.map((row) => [row.label, row.lanes])).toEqual([
      ["T4", 4],
      ["T3", 3],
      ["2 other ways", 3],
    ]);
    expect(groups[2]).toMatchObject({ key: "rest", share: 0.3 });
  });

  it("draws every tariff when they fit, and no share of an agreement with no lanes", () => {
    expect(
      pricingGroups({ count: 0, groups: [group({ lanes: 0 })] }, { rows: 1, currency: "USD" }, t),
    ).toEqual([{ key: "ft_mile:USD", label: "Per mile", lanes: 0, share: 0 }]);
  });
});
