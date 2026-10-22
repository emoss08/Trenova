import { rateAgreementSchema } from "@trenova/shared/types/rate";
import type { TariffInputDeclaration } from "@trenova/shared/types/tariff";
import { describe, expect, it } from "vitest";
import {
  agreementSharedInputs,
  rateAgreementTemplateIds,
  refineRateAgreementInputs,
} from "../rate-agreement-inputs";

const rate: TariffInputDeclaration = {
  name: "rate",
  doc: null,
  type: "Money/mi",
  descriptor: { kind: "quantity", dimension: "Money/Distance", unit: "$/mi" },
  required: true,
  range: null,
};
const minimumCharge: TariffInputDeclaration = {
  name: "minimumCharge",
  doc: null,
  type: "Money",
  descriptor: { kind: "money" },
  required: false,
  default: { $q: "450", u: "$" },
  range: null,
};
const perCwt: TariffInputDeclaration = {
  ...rate,
  type: "Money/cwt",
  descriptor: { kind: "quantity", dimension: "Money/Weight", unit: "$/cwt" },
};

const inputs = new Map([
  ["ft_mile", [rate, minimumCharge]],
  ["ft_ltl", [perCwt, minimumCharge]],
]);

function agreement(overrides: Record<string, unknown>) {
  return {
    code: "ACME-2026",
    name: "Acme",
    customerId: "cus_1",
    effectiveFrom: 1_700_000_000,
    ...overrides,
  };
}

function lane(formulaTemplateId: string, laneInputs: Record<string, unknown> = {}) {
  return { formulaTemplateId, inputs: laneInputs, effectiveFrom: 1_700_000_000 };
}

function issues(value: Record<string, unknown>): [string, string][] {
  const result = rateAgreementSchema
    .superRefine(refineRateAgreementInputs(inputs))
    .safeParse(agreement(value));
  return result.success
    ? []
    : result.error.issues.map((issue) => [issue.path.join("."), issue.message]);
}

describe("rateAgreementTemplateIds", () => {
  it("names each lane's tariff once, in lane order", () => {
    expect(
      rateAgreementTemplateIds([lane("ft_mile"), lane("ft_ltl"), lane("ft_mile"), lane("")]),
    ).toEqual(["ft_mile", "ft_ltl"]);
  });
});

describe("agreementSharedInputs", () => {
  it("offers the agreement only what every lane's tariff declares alike", () => {
    expect(
      agreementSharedInputs([lane("ft_mile"), lane("ft_ltl")], inputs).declarations.map(
        (d) => d.name,
      ),
    ).toEqual(["minimumCharge"]);
  });

  it("waits for every lane's tariff rather than offering a wrong list", () => {
    expect(agreementSharedInputs([lane("ft_mile"), lane("ft_unknown")], inputs)).toEqual({
      declarations: [],
      complete: false,
    });
  });
});

describe("refineRateAgreementInputs", () => {
  it("requires a lane's input with no default that the agreement does not set either", () => {
    expect(issues({ rules: [lane("ft_mile")] })).toEqual([
      ["rules.0.inputs.rate", "Required: nothing else sets it"],
    ]);
  });

  it("lets the agreement's value satisfy a lane", () => {
    expect(
      issues({
        rules: [lane("ft_mile")],
        inputs: { rate: { $q: "2.35", u: "$/mi" } },
      }),
    ).toEqual([]);
  });

  it("reports a lane value that does not fit its tariff at the lane's field", () => {
    expect(
      issues({ rules: [lane("ft_mile"), lane("ft_ltl", { rate: { $q: "2", u: "$/mi" } })] }),
    ).toEqual([
      ["rules.0.inputs.rate", "Required: nothing else sets it"],
      ["rules.1.inputs.rate", "Must be a quantity in a unit of Money/Weight"],
    ]);
  });

  it("refuses an agreement value some lane's tariff does not declare alike", () => {
    expect(
      issues({
        rules: [
          lane("ft_mile", { rate: { $q: "2", u: "$/mi" } }),
          lane("ft_ltl", { rate: { $q: "30", u: "$/cwt" } }),
        ],
        inputs: { rate: { $q: "2", u: "$/mi" }, minimumCharge: { $q: "500", u: "$" } },
      }),
    ).toEqual([["inputs.rate", "Not every lane's tariff has an input named rate of this type"]]);
  });

  it("leaves a lane whose tariff has not loaded to the server", () => {
    expect(issues({ rules: [lane("ft_unknown", { anything: true })] })).toEqual([]);
  });
});
