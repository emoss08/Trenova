import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import {
  tariffInputDeclarationsSchema,
  type TariffInputDeclaration,
} from "@trenova/shared/types/tariff";
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
    constraint: null,
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

describe("tariffInputIssues with constraints", () => {
  const bounded = declaration({
    constraint: {
      range: { min: { $q: "0.5", u: "$/mi" }, max: { $q: "15", u: "$/mi" }, maxInclusive: true },
    },
  });

  it("accepts both ends of an inclusive range", () => {
    expect(tariffInputIssues([bounded], { rate: { $q: "0.50", u: "$/mi" } }, t)).toEqual([]);
    expect(tariffInputIssues([bounded], { rate: { $q: "15", u: "$/mi" } }, t)).toEqual([]);
  });

  it("refuses a value past either end with the bounds", () => {
    expect(tariffInputIssues([bounded], { rate: { $q: "235", u: "$/mi" } }, t)).toEqual([
      { path: ["rate"], message: "Must be from $0.50/mi to $15.00/mi" },
    ]);
    expect(tariffInputIssues([bounded], { rate: { $q: "0.49", u: "$/mi" } }, t)).toHaveLength(1);
  });

  it("reports each item of a list input outside the bounds at its index", () => {
    const stops = declaration({
      name: "stops",
      type: "List<Int>",
      descriptor: { kind: "list", item: { kind: "int" } },
      constraint: { range: { min: { $n: "0" }, max: { $n: "10" }, maxInclusive: false } },
    });
    expect(
      tariffInputIssues([stops], { stops: [{ $n: "2" }, { $n: "10" }, { $n: "-1" }] }, t),
    ).toEqual([
      { path: ["stops", 1], message: "Must be at least 0 and below 10" },
      { path: ["stops", 2], message: "Must be at least 0 and below 10" },
    ]);
  });

  it("leaves a value of the wrong type to the type check alone", () => {
    expect(tariffInputIssues([bounded], { rate: { $q: "1", u: "lb" } }, t)).toHaveLength(1);
  });
});

// The inputs tariff.check answers for `input open: Time = @08:00 where @00:00..=@23:59`,
// `input close: Time? = none`, `input zip: Text = "30301" where "30000"..="39999"` and
// `input cls: FreightClass = Class70 where Class50..=Class100`, as the crate's
// testdata/tariff/check/{time,text-ranges,ordered-enums}.json record them.
const newKinds = tariffInputDeclarationsSchema.parse([
  {
    constraint: {
      range: { max: { $time: "23:59:00" }, maxInclusive: true, min: { $time: "00:00:00" } },
    },
    default: { $time: "08:00:00" },
    descriptor: { kind: "time" },
    doc: "When the dock opens.",
    name: "open",
    range: { from: 42, to: 46 },
    required: false,
    type: "Time",
  },
  {
    constraint: null,
    default: null,
    descriptor: { inner: { kind: "time" }, kind: "optional" },
    doc: null,
    name: "close",
    range: { from: 90, to: 95 },
    required: false,
    type: "Time?",
  },
  {
    constraint: { range: { max: "39999", maxInclusive: true, min: "30000" } },
    default: "30301",
    descriptor: { kind: "text" },
    doc: null,
    name: "zip",
    range: { from: 17, to: 20 },
    required: false,
    type: "Text",
  },
  {
    constraint: { range: { max: "Class100", maxInclusive: true, min: "Class50" } },
    default: "Class70",
    descriptor: {
      kind: "enum",
      name: "FreightClass",
      ordered: true,
      variants: [
        "Class50",
        "Class55",
        "Class60",
        "Class65",
        "Class70",
        "Class77_5",
        "Class85",
        "Class92_5",
        "Class100",
        "Class110",
        "Class125",
        "Class150",
        "Class175",
        "Class200",
        "Class250",
        "Class300",
        "Class400",
        "Class500",
      ],
    },
    doc: null,
    name: "cls",
    range: { from: 17, to: 20 },
    required: false,
    type: "FreightClass",
  },
]);

describe("tariffInputIssues for times, text ranges and ordered enums", () => {
  it("reads every new kind of input tariff.check sends", () => {
    expect(newKinds.map((input) => input.descriptor.kind)).toEqual([
      "time",
      "optional",
      "text",
      "enum",
    ]);
    expect(newKinds[1]?.default).toBeNull();
  });

  it("accepts values inside each range", () => {
    expect(
      tariffInputIssues(
        newKinds,
        { open: { $time: "23:59" }, close: null, zip: "35004", cls: "Class100" },
        t,
      ),
    ).toEqual([]);
  });

  it("refuses a value outside a text range or an ordered enum's range as it is typed", () => {
    expect(tariffInputIssues(newKinds, { zip: "40000", cls: "Class110" }, t)).toEqual([
      { path: ["zip"], message: "Must be from 30000 to 39999" },
      { path: ["cls"], message: "Must be from Class50 to Class100" },
    ]);
  });

  it("refuses a time of day in another shape, before its bounds", () => {
    expect(tariffInputIssues(newKinds, { open: "08:00", close: { $time: "8:00" } }, t)).toEqual([
      { path: ["open"], message: "Must be a time of day" },
      { path: ["close"], message: "Must be a time of day" },
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
