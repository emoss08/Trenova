import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import type { TariffInputDeclaration } from "@trenova/shared/types/tariff";
import { describe, expect, it } from "vitest";
import {
  humanizeTariffName,
  sharedTariffInputs,
  tariffInputIssues,
  tariffInputText,
  tariffInputValuesSchemaFor,
} from "../tariff-inputs";

const t: TranslateFn = ((message: string, ...args: unknown[]) =>
  message.replace(/\{(\d+)\}/g, (_, index: string) => String(args[Number(index)]))) as TranslateFn;

function declaration(overrides: Partial<TariffInputDeclaration>): TariffInputDeclaration {
  return {
    name: "rate",
    doc: null,
    type: "Money/mi",
    descriptor: { kind: "quantity", dimension: "Money/Distance", unit: "$/mi" },
    required: true,
    range: null,
    ...overrides,
  };
}

const rate = declaration({});
const minimum = declaration({
  name: "minimumCharge",
  type: "Money",
  descriptor: { kind: "money" },
  required: false,
  default: { $q: "450", u: "$" },
});
const breaks = declaration({
  name: "breaks",
  type: "List<Break>",
  descriptor: {
    kind: "list",
    item: {
      kind: "record",
      name: "std.ltl.Break",
      fields: [
        { name: "from", doc: null, type: { kind: "quantity", dimension: "Weight", unit: "lb" } },
        { name: "minimum", doc: null, type: { kind: "optional", inner: { kind: "money" } } },
      ],
    },
  },
});

describe("humanizeTariffName", () => {
  it("reads a camelCase input name as words", () => {
    expect(humanizeTariffName("extraDropFee")).toBe("Extra drop fee");
    expect(humanizeTariffName("rate")).toBe("Rate");
    expect(humanizeTariffName("residentialZip3s")).toBe("Residential zip3s");
    expect(humanizeTariffName("maxFSCPercent")).toBe("Max FSC percent");
  });
});

describe("tariffInputText", () => {
  it("labels an input with its doc's first sentence and describes it with the rest", () => {
    expect(
      tariffInputText({
        name: "fuelPeg",
        doc: "Fuel is priced off this peg. One step per 5 cents.",
      }),
    ).toEqual({ label: "Fuel is priced off this peg", description: "One step per 5 cents." });
  });

  it("labels an input with no doc by its name", () => {
    expect(tariffInputText({ name: "minimumCharge", doc: null })).toEqual({
      label: "Minimum charge",
      description: null,
    });
  });
});

describe("sharedTariffInputs", () => {
  it("keeps only the inputs every tariff declares with the same type", () => {
    const perKm = declaration({
      descriptor: { kind: "quantity", dimension: "Money/Weight", unit: "$/cwt" },
    });
    expect(
      sharedTariffInputs([
        [rate, minimum],
        [minimum, perKm],
      ]).map((d) => d.name),
    ).toEqual(["minimumCharge"]);
  });

  it("drops a default the tariffs disagree on", () => {
    const otherDefault = { ...minimum, default: { $q: "500", u: "$" } };
    expect(sharedTariffInputs([[minimum], [otherDefault]])[0]?.default).toBeUndefined();
    expect(sharedTariffInputs([[minimum], [minimum]])[0]?.default).toEqual({ $q: "450", u: "$" });
  });

  it("shares nothing when there are no tariffs", () => {
    expect(sharedTariffInputs([])).toEqual([]);
  });
});

describe("tariffInputIssues", () => {
  it("accepts values that fit, and inputs left to their default", () => {
    expect(tariffInputIssues([rate, minimum], { rate: { $q: "2.35", u: "$/km" } }, t)).toEqual([]);
  });

  it("reports a value in another dimension at its input", () => {
    expect(tariffInputIssues([rate], { rate: { $q: "2.35", u: "$/cwt" } }, t)).toEqual([
      { path: ["rate"], message: "Must be a quantity in a unit of Money/Distance" },
    ]);
  });

  it("reports a problem inside a list at the row and field", () => {
    const values = {
      rate: { $q: "1", u: "$/mi" },
      breaks: [{ from: { $q: "0", u: "lb" }, minimum: null }, { from: { $q: "500", u: "mi" } }],
    };
    expect(tariffInputIssues([rate, breaks], values, t)).toEqual([
      { path: ["breaks", 1, "from"], message: "Must be a quantity in a unit of Weight" },
    ]);
  });

  it("requires an input with no default only when asked and nothing is inherited", () => {
    expect(tariffInputIssues([rate], {}, t)).toEqual([]);
    expect(tariffInputIssues([rate], {}, t, { requireUnset: true })).toEqual([
      { path: ["rate"], message: "Required: nothing else sets it" },
    ]);
    expect(
      tariffInputIssues([rate], {}, t, {
        requireUnset: true,
        inherited: { rate: { $q: "2", u: "$/mi" } },
      }),
    ).toEqual([]);
  });

  it("accepts money per pallet, which the module writes in that unit", () => {
    const perPallet = declaration({
      name: "palletFee",
      type: "Money/pallet",
      descriptor: { kind: "money" },
    });
    expect(tariffInputIssues([perPallet], { palletFee: { $q: "5", u: "$/pallet" } }, t)).toEqual(
      [],
    );
    expect(tariffInputIssues([perPallet], { palletFee: { $q: "5", u: "$/mi" } }, t)).toEqual([
      { path: ["palletFee"], message: "Must be an amount of money" },
    ]);
  });

  it("reports a value for an input the tariff does not declare", () => {
    expect(tariffInputIssues([rate], { rate: { $q: "1", u: "$/mi" }, rat: true }, t)).toEqual([
      { path: ["rat"], message: "This tariff has no input named rat" },
    ]);
  });
});

describe("tariffInputValuesSchemaFor", () => {
  it("turns each issue into a zod issue at its path", () => {
    const result = tariffInputValuesSchemaFor([rate, minimum], t, { requireUnset: true }).safeParse(
      { minimumCharge: { $q: "450", u: "%" } },
    );

    expect(result.success).toBe(false);
    expect(result.error?.issues.map((issue) => [issue.path, issue.message])).toEqual([
      [["rate"], "Required: nothing else sets it"],
      [["minimumCharge"], "Must be an amount of money"],
    ]);
  });
});
