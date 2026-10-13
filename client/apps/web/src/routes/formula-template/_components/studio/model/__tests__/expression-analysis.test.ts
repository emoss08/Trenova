import { buildKnownIdentifiers } from "@/components/formula-editor/known-identifiers";
import { describe, expect, it } from "vitest";
import { analyzeExpression, inferInputType, keyFromLabel } from "../expression-analysis";

const known = buildKnownIdentifiers(undefined, [
  { name: "fuelSurchargePercent", type: "Number", description: "", required: false },
]);

describe("analyzeExpression", () => {
  it("separates schema reads, input reads and functions, and never counts a let as a read", () => {
    const analysis = analyzeExpression(
      "let linehaul = baseRate * totalDistance;\nmax(linehaul * (1 + fuelSurchargePercent / 100), 250)",
      known,
    );

    expect(analysis.errorCount).toBe(0);
    expect([...analysis.schemaReads]).toEqual(["baseRate", "totalDistance"]);
    expect([...analysis.inputReads]).toEqual(["fuelSurchargePercent"]);
    expect([...analysis.functionCalls]).toEqual(["max"]);
    expect(analysis.reads).toEqual(["fuelSurchargePercent", "baseRate", "totalDistance"]);
    expect([...analysis.bindings]).toEqual(["linehaul"]);
  });

  it("reports an unknown name as a problem the studio can turn into an input", () => {
    const analysis = analyzeExpression("baseRate * weekendFee", known);
    expect(analysis.problems).toHaveLength(1);
    expect(analysis.problems[0]).toMatchObject({ code: "unknown-variable", name: "weekendFee" });
    expect(analysis.errorCount).toBe(1);
  });

  it("treats a blank expression as empty with nothing wrong", () => {
    const analysis = analyzeExpression("   ", known);
    expect(analysis.empty).toBe(true);
    expect(analysis.problems).toEqual([]);
  });
});

describe("inferInputType", () => {
  it("starts flags named is, has, requires or uses as Boolean and everything else as Number", () => {
    expect(inferInputType("isWeekend")).toBe("Boolean");
    expect(inferInputType("hasLiftgate")).toBe("Boolean");
    expect(inferInputType("requiresEscort")).toBe("Boolean");
    expect(inferInputType("usesTolls")).toBe("Boolean");
    expect(inferInputType("weekendFee")).toBe("Number");
    expect(inferInputType("island")).toBe("Number");
  });
});

describe("keyFromLabel", () => {
  it("camel-cases a label into a line key", () => {
    expect(keyFromLabel("Fuel surcharge")).toBe("fuelSurcharge");
    expect(keyFromLabel("Stops after two!")).toBe("stopsAfterTwo");
    expect(keyFromLabel("2nd stop")).toBe("line2ndStop");
    expect(keyFromLabel("  ")).toBe("");
  });
});
