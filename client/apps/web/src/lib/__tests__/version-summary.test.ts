import type { RateAgreementVersion } from "@trenova/shared/types/rate";
import { describe, expect, it } from "vitest";
import { describeVersion } from "../version-summary";

const CHARGE_ID = "acc_01K2ZX8Y4M5N6P7Q8R9S0T1V2W";

function version(overrides: Partial<RateAgreementVersion>): RateAgreementVersion {
  return {
    versionNumber: 2,
    effectiveFrom: 1_700_000_000,
    changeMessage: "",
    ...overrides,
  } as RateAgreementVersion;
}

describe("describeVersion", () => {
  it("prefers the change message when one was written", () => {
    const v = version({
      changeMessage: "Annual renegotiation",
      changeSummary: { currency: { from: "USD", to: "CAD" } },
    });
    expect(describeVersion(v)).toBe("Annual renegotiation");
  });

  it("humanizes header field paths instead of echoing camelCase keys", () => {
    const v = version({
      changeSummary: {
        agreementEffectiveFrom: { from: 1, to: 2 },
        defaultMinCharge: { from: "100", to: "150" },
      },
    });
    const text = describeVersion(v);
    expect(text).toContain("Effective from");
    expect(text).toContain("Default minimum charge");
    expect(text).not.toContain("agreementEffectiveFrom");
  });

  it("names an accessorial term change by the charge's code, never its id", () => {
    const v = version({
      changeSummary: {
        [`accessorialTerms.${CHARGE_ID}.amount`]: { from: "25", to: "40" },
      },
      accessorialNames: { [CHARGE_ID]: "DETENTION" },
    });
    const text = describeVersion(v);
    expect(text).toContain("DETENTION");
    expect(text).not.toContain(CHARGE_ID);
  });

  it("reads an added accessorial as an addition", () => {
    const v = version({
      changeSummary: {
        [`accessorialTerms.${CHARGE_ID}`]: { to: { amount: "75" } },
      },
      accessorialNames: { [CHARGE_ID]: "TONU" },
    });
    expect(describeVersion(v)).toContain("Added TONU accessorial");
  });

  it("falls back to a plain phrase when the charge cannot be named", () => {
    const v = version({
      changeSummary: {
        [`accessorialTerms.${CHARGE_ID}`]: { from: { amount: "75" } },
      },
    });
    const text = describeVersion(v);
    expect(text).toContain("Removed accessorial");
    expect(text).not.toContain(CHARGE_ID);
  });

  it("collapses fuel binding sub-fields into one phrase", () => {
    const v = version({
      changeSummary: {
        "fuelTerms.fuelSurchargeProgramId": { from: "fsp_1", to: "fsp_2" },
        "fuelTerms.capAmount": { from: null, to: "500" },
      },
    });
    expect(describeVersion(v)).toBe("Fuel terms");
  });

  it("shows a dash when nothing is recorded", () => {
    expect(describeVersion(version({}))).toBe("—");
  });
});
