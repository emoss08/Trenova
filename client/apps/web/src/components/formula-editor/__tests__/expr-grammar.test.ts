import { describe, expect, it } from "vitest";
import { checkExpression } from "../expr-check";
import { formatExpression } from "../expr-format";
import { parseExpression, stringValue, tokenize, type Node } from "../expr-parser";
import { buildKnownIdentifiers } from "../known-identifiers";
import { describedShipment } from "./described-shipment";

const known = buildKnownIdentifiers(describedShipment, [{ name: "baseRate", type: "Number" }]);

function only(expression: string): Node {
  const program = parseExpression(expression);
  expect(program.diagnostics).toEqual([]);
  const statement = program.statements.at(-1);
  if (!statement) throw new Error("no statement");
  return statement.value;
}

function fixesFor(expression: string, code: string): string[] {
  return checkExpression(expression, known)
    .diagnostics.filter((d) => d.code === code)
    .flatMap((d) => d.fixes)
    .flatMap((fix) => (fix.kind === "replace" ? fix.changes.map((c) => c.insert) : []));
}

describe("numbers as the engine's lexer reads them", () => {
  it("reads separators, hex, octal, binary and a trailing dot", () => {
    expect(only("10_000")).toMatchObject({ kind: "number", value: 10000, whole: true });
    expect(only("0x1F")).toMatchObject({ kind: "number", value: 31, whole: true });
    expect(only("0o17")).toMatchObject({ kind: "number", value: 15, whole: true });
    expect(only("0b101")).toMatchObject({ kind: "number", value: 5, whole: true });
    expect(only("1.")).toMatchObject({ kind: "number", value: 1, whole: false });
    expect(only("2.5e3")).toMatchObject({ kind: "number", value: 2500, whole: false });
  });

  it("leaves a range after a whole number alone", () => {
    expect(tokenize("1..3").tokens.map((token) => token.text)).toEqual(["1", "..", "3", ""]);
  });
});

describe("strings as the engine reads them", () => {
  it("reads escapes in quoted strings and none in backtick strings", () => {
    expect(stringValue('"a\\nb"')).toBe("a\nb");
    expect(stringValue("'it\\'s'")).toBe("it's");
    expect(stringValue('"\\u00e9t\\x41"')).toBe("étA");
    expect(stringValue("`a\\n`")).toBe("a\\n");
    expect(stringValue("`a``b`")).toBe("a`b");
  });

  it("knows a doubled backtick does not close a raw string", () => {
    expect(parseExpression("len(`a``b`)").diagnostics).toEqual([]);
    expect(parseExpression("len(`a``)").diagnostics.map((d) => d.message)).toContain(
      "This text is never closed",
    );
  });
});

describe("the grammar the editor used to refuse", () => {
  it("reads non-ASCII names, named pointers and optional indexes", () => {
    expect(only("let tarifa_año = 2; tarifa_año")).toMatchObject({
      kind: "ident",
      name: "tarifa_año",
    });
    const reduced = only("reduce([1, 2], #acc + #index, 0)");
    expect(reduced.kind).toBe("call");
    expect(only("stops?.[0]")).toMatchObject({ kind: "index", optional: true });
  });

  it("scopes a let inside parentheses, an if block and a predicate", () => {
    expect(only("(let a = 2; a * 3)")).toMatchObject({
      kind: "sequence",
      lets: [{ name: "a" }],
      grouped: 1,
    });
    const analysis = checkExpression("(let a = 2; a * 3) + (let a = 4; a)", known);
    expect(analysis.diagnostics).toEqual([]);
  });

  it("takes a trailing comma in a call", () => {
    expect(only("max(1, 2,)")).toMatchObject({ kind: "call", args: [{}, {}] });
  });
});

describe("one-click fixes for what the engine refuses", () => {
  it("adds the parentheses a piped function needs", () => {
    expect(parseExpression("totalWeight | round").diagnostics[0]?.fix?.insert).toBe("round()");
  });

  it("maps the list for a sum given a predicate", () => {
    expect(fixesFor("sum(stops, .weight)", "type")).toEqual(["sum(map(stops, .weight))"]);
  });

  it("takes the whole part of a decimal field before '%'", () => {
    expect(fixesFor("totalDistance % 100", "type")).toEqual(["int(totalDistance)"]);
  });
});

describe("while the engine's schema is loading", () => {
  const loading = buildKnownIdentifiers(undefined, [{ name: "baseRate", type: "Number" }]);

  it("calls no name unknown, and says names are not checked yet", () => {
    const analysis = checkExpression('baseRate * totalDistance + lookup("zones", 1)', loading);
    expect(analysis.pending).toBe(true);
    expect(analysis.diagnostics).toEqual([]);
  });

  it("still reports what is wrong with the text itself", () => {
    const analysis = checkExpression("baseRate *", loading);
    expect(analysis.diagnostics.map((d) => d.code)).toEqual(["syntax"]);
  });
});

describe("formatting the new grammar", () => {
  it.each([
    "(let a = 2; a * 3)",
    "reduce([1, 2, 3], #acc + #, 0)",
    "stops?.[0] == nil ? 1 : 0",
    "len(filter([1, 2, 3], {let x = #; x > 1}))",
    "if isRush { let a = 2; a } else { 0 }",
    "0x1F + 10_000 + 1.",
  ])("keeps %s as written", (source) => {
    const result = formatExpression(source);
    expect(result).toEqual({ ok: true, text: source, changed: false });
  });
});
