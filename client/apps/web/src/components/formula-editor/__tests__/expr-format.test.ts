import { describe, expect, it } from "vitest";
import { formatExpression } from "../expr-format";

const LONG_LETS =
  "let linehaul = mileRate * totalDistance; let weightCharge = max(0, totalWeight - weightFreeLbs) / 100 * weightRatePerCwt; let stopCharge = max(0, totalStops - stopsIncluded) * stopRate; let hazmatCharge = hasHazmat ? hazmatFee : 0; linehaul + weightCharge + stopCharge + hazmatCharge";

function formatted(source: string, width?: number): string {
  const result = formatExpression(source, { width });
  if (!result.ok) throw new Error(`refused: ${result.reason}`);
  return result.text;
}

describe("formatExpression", () => {
  it("puts every statement on its own line", () => {
    expect(formatted(LONG_LETS)).toBe(
      [
        "let linehaul = mileRate * totalDistance;",
        "let weightCharge = max(0, totalWeight - weightFreeLbs) / 100 * weightRatePerCwt;",
        "let stopCharge = max(0, totalStops - stopsIncluded) * stopRate;",
        "let hazmatCharge = hasHazmat ? hazmatFee : 0;",
        "linehaul + weightCharge + stopCharge + hazmatCharge",
      ].join("\n"),
    );
  });

  it("normalizes spacing and keeps the author's parentheses and literals", () => {
    expect(formatted("(baseRate*totalDistance)+(hasHazmat?hazmatFee:0)*1.50")).toBe(
      "(baseRate * totalDistance) + (hasHazmat ? hazmatFee : 0) * 1.50",
    );
    expect(formatted('serviceType.code=="EXP"  and  not   isWeekendPickup')).toBe(
      'serviceType.code == "EXP" and not isWeekendPickup',
    );
  });

  it("breaks a long sum one term per line with the operator leading", () => {
    const source =
      "(mileRate * totalDistance) + (max(0, totalStops - stopsIncluded) * stopRate) + (hasHazmat ? hazmatFee : 0) + (isWeekendPickup ? weekendFee : 0)";
    expect(formatted(source)).toBe(
      [
        "(mileRate * totalDistance)",
        "  + (max(0, totalStops - stopsIncluded) * stopRate)",
        "  + (hasHazmat ? hazmatFee : 0)",
        "  + (isWeekendPickup ? weekendFee : 0)",
      ].join("\n"),
    );
  });

  it("breaks a let after its '=' when the value then fits", () => {
    const source =
      "let fuelSurcharge = max(0, coalesce(fuelPrice, fuelBaseline) - fuelBaseline) * fuelMultiplier;";
    expect(formatted(`${source} fuelSurcharge`, 80)).toBe(
      [
        "let fuelSurcharge =",
        "  max(0, coalesce(fuelPrice, fuelBaseline) - fuelBaseline) * fuelMultiplier;",
        "fuelSurcharge",
      ].join("\n"),
    );
  });

  it("breaks a ternary chain one case per line", () => {
    const source =
      "totalWeight < breakOne ? rateOne * totalWeight : totalWeight < breakTwo ? rateTwo * totalWeight : rateThree * totalWeight";
    expect(formatted(source, 60)).toBe(
      [
        "totalWeight < breakOne ? rateOne * totalWeight",
        "  : totalWeight < breakTwo ? rateTwo * totalWeight",
        "  : rateThree * totalWeight",
      ].join("\n"),
    );
  });

  it("breaks a call one argument per line when nothing shorter fits", () => {
    const source =
      "lookup(accessorialTable, customer.code, serviceType.code, equipmentType.code, origin.state)";
    expect(formatted(source, 50)).toBe(
      [
        "lookup(",
        "  accessorialTable,",
        "  customer.code,",
        "  serviceType.code,",
        "  equipmentType.code,",
        "  origin.state",
        ")",
      ].join("\n"),
    );
  });

  it("keeps a pipe, an infix not and a predicate as written", () => {
    expect(formatted("totalWeight|round(2)")).toBe("totalWeight | round(2)");
    expect(formatted('serviceType.code not in ["EXP","TEAM"]')).toBe(
      'serviceType.code not in ["EXP", "TEAM"]',
    );
    expect(formatted("count(commodities,{.hazmat})>0?hazmatFee:0")).toBe(
      "count(commodities, {.hazmat}) > 0 ? hazmatFee : 0",
    );
    expect(formatted("filter(stops, #.type == 'Delivery')")).toBe(
      "filter(stops, #.type == 'Delivery')",
    );
  });

  it("keeps a blank line the author left between statements", () => {
    expect(formatted("let a = 1;\n\n\nlet b = 2;\na + b")).toBe("let a = 1;\n\nlet b = 2;\na + b");
  });

  it("formats if … else on one line, or one branch per line when long", () => {
    expect(formatted("if hasHazmat {hazmatFee} else {0}")).toBe(
      "if hasHazmat { hazmatFee } else { 0 }",
    );
    expect(
      formatted(
        "if requiresTemperatureControl { reeferRate * totalDistance } else if hasHazmat { hazmatFee } else { 0 }",
        40,
      ),
    ).toBe(
      [
        "if requiresTemperatureControl {",
        "  reeferRate * totalDistance",
        "} else if hasHazmat {",
        "  hazmatFee",
        "} else {",
        "  0",
        "}",
      ].join("\n"),
    );
  });

  it("reports when nothing changed", () => {
    const result = formatExpression("baseRate * totalDistance");
    expect(result).toEqual({ ok: true, text: "baseRate * totalDistance", changed: false });
  });

  it("refuses an empty, a broken, or a commented formula", () => {
    expect(formatExpression("   ")).toEqual({ ok: false, reason: "empty" });
    expect(formatExpression("baseRate * (totalDistance")).toEqual({ ok: false, reason: "syntax" });
    expect(formatExpression("baseRate // per mile\n* totalDistance")).toEqual({
      ok: false,
      reason: "comments",
    });
  });

  it("is stable: formatting its own output changes nothing", () => {
    const once = formatted(LONG_LETS, 60);
    expect(formatExpression(once, { width: 60 })).toEqual({ ok: true, text: once, changed: false });
  });
});
