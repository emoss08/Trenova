import { rulesSync } from "@trenova/shared/rules/runtime";
import { tariffCheck, tariffTest } from "@trenova/shared/rules/tariff";
import { EMPTY_TARIFF_CONTEXT } from "@trenova/shared/types/tariff";
import { describe, expect, it } from "vitest";
import { headerMismatch, libraryPath, starterLibrary } from "../starter";

describe("the starter library", () => {
  it("checks clean as a library and every test it holds passes", () => {
    const rules = rulesSync();
    const source = starterLibrary("accessorials");
    const check = tariffCheck(rules, { source, kind: "library", context: EMPTY_TARIFF_CONTEXT });
    expect(check.diagnostics).toEqual([]);
    expect(check.header).toEqual({ kind: "library", name: "org.accessorials" });

    const run = tariffTest(rules, { source, context: EMPTY_TARIFF_CONTEXT });
    expect(run.error).toBeNull();
    expect(run.tests.map((test) => [test.name, test.status])).toEqual([
      ["the flat fee defaults to $150", "passed"],
    ]);
  });

  it("names itself org.example until the library has a valid name", () => {
    expect(starterLibrary("").startsWith('library "org.example"\n')).toBe(true);
    expect(starterLibrary("Not Valid").startsWith('library "org.example"\n')).toBe(true);
    expect(starterLibrary("zoneRates").startsWith('library "org.zoneRates"\n')).toBe(true);
  });
});

describe("headerMismatch", () => {
  it("is true only when the header names another path than org.<name>", () => {
    expect(headerMismatch("org.accessorials", "accessorials")).toBe(false);
    expect(headerMismatch("accessorials", "accessorials")).toBe(true);
    expect(headerMismatch("org.fees", "accessorials")).toBe(true);
    expect(headerMismatch("org.fees", " fees ")).toBe(false);
  });

  it("says nothing while there is no header or the name is not a valid one yet", () => {
    expect(headerMismatch(null, "accessorials")).toBe(false);
    expect(headerMismatch(undefined, "accessorials")).toBe(false);
    expect(headerMismatch("org.fees", "")).toBe(false);
    expect(headerMismatch("org.fees", "Fees")).toBe(false);
  });
});

describe("libraryPath", () => {
  it("is org. and the name", () => {
    expect(libraryPath("accessorials")).toBe("org.accessorials");
  });
});
