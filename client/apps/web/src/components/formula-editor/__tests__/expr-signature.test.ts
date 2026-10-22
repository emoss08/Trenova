import { rulesSync } from "@trenova/shared/rules/runtime";
import { describe, expect, it } from "vitest";
import { signatureAt, signatureParts } from "../expr-signature";
import { knownFor } from "./formula-fixtures";

const rules = rulesSync();
const known = knownFor();

describe("signatureAt", () => {
  it("finds the catalogued call around the cursor and the argument being typed", () => {
    const help = signatureAt(rules, "round(totalWeight, 2)", 19, known);
    expect(help).toMatchObject({ active: 1, at: 5 });
    expect(help?.fn.signature).toBe("round(x, decimals?)");
  });

  it("follows the innermost call, built-ins included", () => {
    const source = "sum(filter(stops, .weight > 1))";
    const help = signatureAt(rules, source, source.indexOf(".weight"), known);
    expect(help).toMatchObject({ active: 1, at: source.indexOf("(stops") });
    expect(help?.fn.name).toBe("filter");
  });

  it("offers nothing outside a call, or in a call the catalog does not have", () => {
    expect(signatureAt(rules, "totalWeight * 2", 3, known)).toBeNull();
    expect(signatureAt(rules, "bogus(1, 2)", 7, known)).toBeNull();
  });
});

describe("signatureParts", () => {
  it("splits a signature into its parameters", () => {
    expect(signatureParts("round(x, decimals?)")).toEqual({
      name: "round",
      params: ["x", "decimals?"],
    });
    expect(signatureParts('text startsWith "prefix"')).toEqual({
      name: 'text startsWith "prefix"',
      params: [],
    });
  });
});
