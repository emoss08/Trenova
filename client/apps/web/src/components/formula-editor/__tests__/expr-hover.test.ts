import { rulesSync } from "@trenova/shared/rules/runtime";
import { describe, expect, it } from "vitest";
import { findHoveredIdentifier, formatHoverValue, renderHoverCard } from "../expr-hover";
import { knownFor } from "./formula-fixtures";

const rules = rulesSync();
const known = knownFor([{ name: "fuelPct", type: "Number", description: "", required: false }]);

function hoverAt(doc: string, at: string, offset = 1) {
  return findHoveredIdentifier(rules, doc, doc.indexOf(at) + offset, known);
}

function card(
  doc: string,
  at: string,
  { value, offset = 1 }: { value?: Parameters<typeof renderHoverCard>[2]; offset?: number } = {},
) {
  const hovered = hoverAt(doc, at, offset);
  if (!hovered) throw new Error(`nothing under ${at}`);
  return renderHoverCard(hovered, known, value);
}

function tags(node: HTMLElement | null): string[] {
  return [...(node?.querySelectorAll(".cm-expr-hover-tag") ?? [])].map(
    (tag) => tag.textContent ?? "",
  );
}

describe("findHoveredIdentifier", () => {
  const doc = "coalesce(weight, 0) * customer.name + fuel";

  it("returns the identifier under the cursor with its span and what it is", () => {
    expect(hoverAt(doc, "weight", 2)).toEqual({
      name: "weight",
      kind: "variable",
      from: doc.indexOf("weight"),
      to: doc.indexOf("weight") + "weight".length,
    });
  });

  it("treats a dotted path as one identifier", () => {
    expect(hoverAt(doc, "customer.name", 10)?.name).toBe("customer.name");
  });

  it("names a function as a call, and ignores operators and whitespace", () => {
    expect(hoverAt(doc, "coalesce")).toMatchObject({ name: "coalesce", kind: "function" });
    expect(findHoveredIdentifier(rules, doc, doc.indexOf("*"), known)).toBeNull();
    expect(hoverAt(doc, "fuel", 2)).toMatchObject({ kind: "unknown" });
  });
});

describe("renderHoverCard", () => {
  it("shows a function's signature, description and example from the catalog", () => {
    const node = card("round(totalWeight, 2)", "round");
    expect(node?.querySelector(".cm-expr-hover-name")?.textContent).toBe("round(x, decimals?)");
    expect(tags(node)).toEqual(["function"]);
    expect(node?.querySelector(".cm-expr-hover-body")?.textContent).toBe(
      "Rounds a number to the given decimal places (default 0, between -12 and 12).",
    );
    expect(node?.querySelector(".cm-expr-hover-example")?.textContent).toBe(
      "round(3.14159, 2) // 3.14",
    );
  });

  it("says why a disabled function is turned off", () => {
    const node = card("now()", "now");
    expect(tags(node)).toEqual(["unknown"]);
    expect(node?.querySelector(".cm-expr-hover-body")?.textContent).toBe(
      "`now()` isn't available in formulas: a formula must price a shipment the same way every time it runs. Use `ratingTime`, the date the contract terms are read at",
    );
  });

  it("describes a field that can be empty, with its last preview value", () => {
    const node = card("weight * 2", "weight", {
      value: { name: "weight", value: null, source: "field" },
    });
    expect(tags(node)).toEqual(["a number", "can be empty"]);
    expect(node?.querySelector(".cm-expr-hover-body")?.textContent).toBe("Declared weight");
    expect(node?.querySelector(".cm-expr-hover-value")?.textContent).toBe("= emptyfrom shipment");
  });

  it("describes an item's field inside a predicate from the list's item fields", () => {
    const node = card("sum(map(stops, .weight ?? 0))", "weight");
    expect(tags(node)).toEqual(["a number", "can be empty"]);
    expect(node?.querySelector(".cm-expr-hover-body")?.textContent).toBe("Weight at the stop");
  });

  it("marks an input, a let and a pointer for what they are", () => {
    expect(tags(card("fuelPct * 2", "fuelPct"))).toEqual(["a number", "input"]);
    expect(tags(card("let a = 1; a * 2", "a *", { offset: 0 }))).toEqual([
      "let, this formula's own",
    ]);
    expect(card("map(stops, #index)", "#index")?.textContent).toBe(
      "#indexThe item's position in the list, from 0",
    );
  });

  it("lists an enum's values", () => {
    expect(card('status == "New" ? 1 : 0', "status")?.textContent).toContain(
      "One of: New, InTransit",
    );
  });
});

describe("formatHoverValue", () => {
  it("shows the value and where it came from", () => {
    expect(formatHoverValue({ name: "weight", value: 12000, source: "field" })).toBe(
      "weight = 12000 (from shipment)",
    );
    expect(formatHoverValue({ name: "rate", value: 2.5, source: "override" })).toBe(
      "rate = 2.5 (engine override)",
    );
    expect(formatHoverValue({ name: "code", value: "ACME", source: "sample" })).toBe(
      'code = "ACME" (sample data)',
    );
    expect(formatHoverValue({ name: "weight", value: null, source: "field" })).toBe(
      "weight = empty (from shipment)",
    );
  });
});
