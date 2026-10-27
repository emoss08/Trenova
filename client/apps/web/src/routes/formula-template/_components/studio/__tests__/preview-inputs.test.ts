import type { TariffInputDeclaration } from "@trenova/shared/types/tariff";
import { describe, expect, it } from "vitest";
import { previewInputsResolution } from "../preview-inputs";

const rate: TariffInputDeclaration = {
  name: "rate",
  doc: null,
  type: "Money/mi",
  descriptor: { kind: "quantity", dimension: "Money/Distance", unit: "$/mi" },
  required: false,
  default: { $q: "2.35", u: "$/mi" },
  constraint: {
    range: { min: { $q: "0.5", u: "$/mi" }, max: { $q: "15", u: "$/mi" }, maxInclusive: true },
  },
  range: null,
};

const stops: TariffInputDeclaration = {
  name: "stops",
  doc: null,
  type: "List<Int>",
  descriptor: { kind: "list", item: { kind: "int" } },
  required: false,
  constraint: { range: { min: { $n: "1" }, max: null, maxInclusive: false } },
  range: null,
};

describe("previewInputsResolution", () => {
  it("passes sample values inside the bounds, and leaves unset inputs to their defaults", () => {
    const values = { inputs: { rate: { $q: "3", u: "$/mi" } } };
    expect(previewInputsResolution(values, [rate, stops])).toEqual({ values, errors: {} });
    expect(previewInputsResolution({ inputs: {} }, [rate, stops]).errors).toEqual({});
  });

  it("puts a value outside the bounds on its field, and a list item on its index", () => {
    expect(
      previewInputsResolution(
        { inputs: { rate: { $q: "20", u: "$/mi" }, stops: [{ $n: "2" }, { $n: "0" }] } },
        [rate, stops],
      ),
    ).toEqual({
      values: {},
      errors: {
        inputs: {
          rate: { type: "custom", message: "Must be from $0.50/mi to $15.00/mi" },
          stops: { 1: { type: "custom", message: "Must be at least 1" } },
        },
      },
    });
  });
});
