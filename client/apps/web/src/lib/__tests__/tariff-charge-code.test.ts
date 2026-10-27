import type { TariffDiagnostic } from "@trenova/shared/types/tariff";
import { describe, expect, it } from "vitest";
import { missingChargeCode } from "../tariff-charge-code";

function finding(source: string, code: string, at: string, nth = 0): TariffDiagnostic {
  let from = -1;
  for (let i = 0; i <= nth; i++) from = source.indexOf(at, from + 1);
  return {
    code,
    severity: "error",
    message: { message: "{0} isn't in the accessorial charge catalog", args: [`\`${at}\``] },
    range: { from, to: from + at.length },
    fixes: [],
    related: [],
    module: null,
    path: null,
  } as unknown as TariffDiagnostic;
}

const SOURCE = [
  'tariff "t"',
  'charge "Fuel surcharge" as FSC = $75',
  'charge "Lumper \\"fee\\"" as LMP = $40',
  'test "t" {',
  "  expect charge(FSC) == $75",
  "}",
].join("\n");

describe("missingChargeCode", () => {
  it("takes the code and the label of the charge that bills it", () => {
    expect(missingChargeCode(finding(SOURCE, "unknown-code", "FSC"), SOURCE)).toEqual({
      code: "FSC",
      description: "Fuel surcharge",
    });
  });

  it("decodes escapes in the label", () => {
    expect(missingChargeCode(finding(SOURCE, "unknown-code", "LMP"), SOURCE)).toEqual({
      code: "LMP",
      description: 'Lumper "fee"',
    });
  });

  it("leaves the description empty for a code a test only reads", () => {
    expect(missingChargeCode(finding(SOURCE, "unknown-code", "FSC", 1), SOURCE)).toEqual({
      code: "FSC",
      description: "",
    });
  });

  it("is null for any other finding", () => {
    expect(missingChargeCode(finding(SOURCE, "inactive-code", "FSC"), SOURCE)).toBeNull();
  });

  it("is null for a code the catalog couldn't store", () => {
    const source = 'tariff "t"\ncharge "Fuel" as fsc = $1';
    expect(missingChargeCode(finding(source, "unknown-code", "fsc"), source)).toBeNull();
  });
});
