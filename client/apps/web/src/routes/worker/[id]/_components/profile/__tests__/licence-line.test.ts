import { translate } from "@trenova/shared/i18n/runtime";
import { describe, expect, it } from "vitest";
import { licenceSegments } from "../licence-line";

const APR_18_2030 = 1_902_700_800;

const read = (input: Parameters<typeof licenceSegments>[0]) =>
  licenceSegments(input, translate).map(({ code, value, note }) =>
    [code, value, note].filter(Boolean).join(" "),
  );

describe("the license line", () => {
  it("reads the CDL the way the card abbreviates it", () => {
    expect(
      read({
        stateAbbreviation: "IL",
        cdlClass: "A",
        endorsement: "X",
        restrictions: "",
        licenseExpiry: APR_18_2030,
      }),
    ).toEqual(["IL", "CLASS A", "END X Tanker + Hazmat", "RESTR none", "EXP 04/18/2030"]);
  });

  it("writes restrictions in capitals and dashes what is not filled in", () => {
    expect(
      read({
        stateAbbreviation: null,
        cdlClass: "B",
        endorsement: null,
        restrictions: " l, z ",
        licenseExpiry: null,
      }),
    ).toEqual(["—", "CLASS B", "END —", "RESTR L, Z", "EXP —"]);
  });
});
