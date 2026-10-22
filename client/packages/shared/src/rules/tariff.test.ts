import { EMPTY_TARIFF_CONTEXT, type TariffContext } from "@trenova/shared/types/tariff";
import { describe, expect, it } from "vitest";
import { rulesSync } from "./runtime";
import {
  tariffCatalog,
  tariffCheck,
  tariffComplete,
  tariffDefinition,
  tariffEvaluate,
  tariffFormat,
  tariffHover,
  tariffInlays,
  tariffPrepareRename,
  tariffRename,
  tariffScaffoldTest,
  tariffSignature,
  tariffTest,
  tariffTokens,
} from "./tariff";

const rules = rulesSync();

// A cut of the reference's "A first look" tariff (docs/engineering/formula-language.md): a
// documented rate, a minimum, a coded charge inside an `if`, total clauses and two tests, one of
// which fails on purpose.
const SOURCE = `tariff "Dry van TL"

/// Truckload linehaul with extra drops.

/// Paid per loaded mile.
input rate: Money/mi = $2.35/mi
input minimumCharge: Money = $450
input extraDropFee: Money = $75

let miles = shipment.distance
let drops = shipment.stops.count(s => s.kind == Delivery)

charge "Linehaul" (quantity: miles, rate: rate)

if drops > 1 {
  charge "Extra drops" as XDROP (quantity: drops - 1, rate: extraDropFee)
}

minimum minimumCharge
round to cents

test "a two-drop load bills one extra drop" {
  given shipment.stops = [pickup(), delivery(), delivery()]
  expect charge(XDROP) == $75
}

test "a 100-mile load bills $1" {
  given shipment.distance = 100 mi
  expect total == $1
}
`;

const context: TariffContext = {
  ...EMPTY_TARIFF_CONTEXT,
  codes: [{ code: "XDROP", description: "Extra drop", active: true }],
};

const at = (text: string) => SOURCE.indexOf(text);

describe("tariffCheck", () => {
  it("reads the inputs with their types, docs and wire defaults", () => {
    const check = tariffCheck(rules, { source: SOURCE, kind: "tariff", context });

    expect(check.ok).toBe(true);
    expect(check.header).toEqual({ kind: "tariff", name: "Dry van TL" });
    expect(check.diagnostics).toEqual([]);
    expect(check.inputs.map((input) => ({ ...input, range: undefined }))).toEqual([
      {
        name: "rate",
        doc: "Paid per loaded mile.",
        type: "Money/mi",
        descriptor: { kind: "quantity", dimension: "Money/Distance", unit: "$/mi" },
        required: false,
        default: { $q: "2.35", u: "$/mi" },
      },
      {
        name: "minimumCharge",
        doc: null,
        type: "Money",
        descriptor: { kind: "money" },
        required: false,
        default: { $q: "450", u: "$" },
      },
      {
        name: "extraDropFee",
        doc: null,
        type: "Money",
        descriptor: { kind: "money" },
        required: false,
        default: { $q: "75", u: "$" },
      },
    ]);
  });

  it("outlines inputs, charges with their nesting, total clauses and tests", () => {
    const check = tariffCheck(rules, { source: SOURCE, context });

    expect(
      check.outline?.items.map(({ kind, name, code, depth }) => [kind, name, code, depth]),
    ).toEqual([
      ["header", "Dry van TL", null, 0],
      ["input", "rate", null, 0],
      ["input", "minimumCharge", null, 0],
      ["input", "extraDropFee", null, 0],
      ["charge", "Linehaul", null, 0],
      ["charge", "Extra drops", "XDROP", 1],
      ["total", "minimum", null, 0],
      ["total", "round", null, 0],
      ["test", "a two-drop load bills one extra drop", null, 0],
      ["test", "a 100-mile load bills $1", null, 0],
    ]);
    expect(check.charges.map((charge) => [charge.label, charge.code])).toEqual([
      ["Linehaul", null],
      ["Extra drops", "XDROP"],
    ]);
  });

  it("finds an optional used as a value, with each fix as a source edit in UTF-16 ranges", () => {
    const source = 'tariff "Poids é"\n\nlet w = shipment.weight\ncharge "Linehaul" = $2/cwt * w\n';
    const check = tariffCheck(rules, { source, context });
    const w = source.lastIndexOf("w");

    expect(check.ok).toBe(false);
    const [finding] = check.diagnostics;
    expect(finding).toMatchObject({
      code: "optional",
      severity: "error",
      range: { from: w, to: w + 1 },
      module: null,
      path: null,
    });
    expect(finding?.fixes[0]).toEqual({
      label: { message: "Fall back to {0}", args: ["`w ?? 0 lb`"] },
      changes: [{ range: { from: w, to: w + 1 }, insert: "(w ?? 0 lb)" }],
    });
  });

  it("refuses a code the organisation's catalog does not hold", () => {
    const source = 'tariff "t"\n\ncharge "Fuel" as FSX = $10\n';
    const check = tariffCheck(rules, { source, context });
    expect(check.diagnostics.map((finding) => [finding.code, finding.range])).toEqual([
      ["unknown-code", { from: source.indexOf("FSX"), to: source.indexOf("FSX") + 3 }],
    ]);
  });
});

describe("tariffTest", () => {
  it("runs each test against the sample record and reports the failing expectation's values", () => {
    const run = tariffTest(rules, { source: SOURCE, context });

    expect(run.error).toBeNull();
    expect(run.tests.map((test) => [test.name, test.status])).toEqual([
      ["a two-drop load bills one extra drop", "passed"],
      ["a 100-mile load bills $1", "failed"],
    ]);
    const failed = run.tests[1];
    expect(failed?.failures).toEqual([
      {
        range: { from: at("expect total"), to: at("expect total") + "expect total == $1".length },
        message: { message: "This expectation does not hold" },
        values: [
          { range: { from: at("total == $1"), to: at("total == $1") + 5 }, display: "$450.00" },
        ],
      },
    ]);
    expect(failed?.bill?.total).toEqual({ $q: "450", u: "$" });
  });
});

describe("tariffEvaluate", () => {
  it("rates against the sample record when no record is given, with the trace", () => {
    const evaluation = tariffEvaluate(rules, {
      context,
      record: null,
      trace: true,
      programs: [
        { id: "preview", source: SOURCE, inputs: { minimumCharge: { $q: "500", u: "$" } } },
      ],
    });
    const [result] = evaluation.results;

    expect(result?.status).toBe("rated");
    expect(result?.bill?.lines).toEqual([
      {
        kind: "minimum",
        label: null,
        code: null,
        freight: true,
        amount: { $q: "500", u: "$" },
        quantity: null,
        rate: null,
        range: null,
      },
    ]);
    expect(result?.trace?.[0]).toEqual({
      kind: "bind",
      name: "miles",
      value: { $q: "0", u: "mi" },
      display: "0 mi",
      range: {
        from: at("let miles"),
        to: at("let miles") + "let miles = shipment.distance".length,
      },
    });
  });
});

describe("tariffFormat", () => {
  it("writes the canonical style, and refuses a file with syntax errors", () => {
    expect(tariffFormat(rules, 'tariff "t"\nlet a=1+2\n')).toEqual({
      formatted: 'tariff "t"\nlet a = 1 + 2\n',
      diagnostics: [],
    });
    const refused = tariffFormat(rules, 'tariff "t"\nlet a = 1 +\n');
    expect(refused.formatted).toBeNull();
    expect(refused.diagnostics[0]?.code).toBe("syntax");
  });
});

describe("tariffTokens", () => {
  it("classes every token in UTF-16 ranges", () => {
    const source = 'let t = -5 °F\nlet s = "é 🚚"\n';
    const tokens = tariffTokens(rules, source);
    const text = (from: number, to: number) => source.slice(from, to);

    expect(tokens.map((token) => [text(token.from, token.to), token.kind])).toContainEqual([
      "°F",
      "unit",
    ]);
    expect(tokens.find((token) => token.kind === "text")).toMatchObject({
      from: source.indexOf('"'),
    });
    expect(tokens.at(-1)?.to).toBeLessThanOrEqual(source.length);
  });
});

describe("tariffCatalog", () => {
  it("lists the standard library, methods, units and schema once per runtime", () => {
    const catalog = tariffCatalog(rules);

    expect(tariffCatalog(rules)).toBe(catalog);
    expect(catalog.modules.map((module) => module.path)).toEqual(
      expect.arrayContaining(["std.ltl", "std.fuel", "std.time", "std.accessorials"]),
    );
    expect(catalog.units).toContainEqual(
      expect.objectContaining({ unit: "cwt", dimension: "Weight" }),
    );
    expect(catalog.schema.roots.map((root) => root.name)).toEqual(
      expect.arrayContaining(["shipment", "lane", "customer", "fuel", "rating", "tables"]),
    );
    expect(catalog.keywords).toContain("charge");
  });
});

describe("tariffScaffoldTest", () => {
  it("writes a test block pinning the bill, with a given for each input that differs", () => {
    const scaffold = tariffScaffoldTest(rules, {
      source: SOURCE,
      context,
      record: null,
      inputs: { rate: { $q: "2.1", u: "$/mi" } },
      name: "from a shipment",
    });

    expect(scaffold).toEqual({
      test: 'test "from a shipment" {\n  given input.rate = $2.10/mi\n  expect total == $450.00\n}\n',
      error: null,
      status: null,
      needs: [],
    });
  });
});

describe("the editor operations", () => {
  const cursor = (offset: number) => ({ source: SOURCE, context, kind: "tariff" as const, offset });

  it("completes a record's fields after a dot, sorted by the module", () => {
    const offset = at("shipment.distance") + "shipment.".length;
    const answer = tariffComplete(rules, cursor(offset));
    expect(answer.range).toEqual({ from: offset, to: offset });
    const distance = answer.items.find((item) => item.label === "distance");
    expect(distance).toMatchObject({
      kind: "field",
      insert: "distance",
      snippet: false,
      range: null,
    });
    const sorts = answer.items.map((item) => item.sort ?? "");
    expect([...sorts].sort()).toEqual(sorts);
  });

  it("hovers a binding with its declaration, type and unit", () => {
    expect(tariffHover(rules, cursor(at("miles =") + 1))).toEqual({
      range: { from: at("miles ="), to: at("miles =") + "miles".length },
      contents: {
        signature: "let miles: Distance",
        type: "Distance",
        unit: "mi",
        doc: null,
        value: null,
      },
    });
  });

  it("hovers nothing on blank space", () => {
    expect(tariffHover(rules, cursor(at("\n\nlet miles") + 1))).toBeNull();
  });

  it("helps with a call's signature, naming the active parameter", () => {
    const help = tariffSignature(rules, cursor(at("count(s") + "count(".length));
    expect(help?.signatures.length).toBeGreaterThan(0);
    expect(help?.active).toBe(0);
    expect(help?.activeParameter).toBe(0);
  });

  it("finds a binding's declaration in this file", () => {
    const use = SOURCE.indexOf("miles,");
    expect(tariffDefinition(rules, cursor(use + 1))).toEqual([
      {
        module: null,
        range: expect.objectContaining({ from: at("let miles") }),
        selectionRange: { from: at("miles ="), to: at("miles =") + "miles".length },
      },
    ]);
  });

  it("prepares and applies a rename to every reference", () => {
    const offset = at("drops =") + 1;
    expect(tariffPrepareRename(rules, cursor(offset))).toEqual({
      range: { from: at("drops ="), to: at("drops =") + "drops".length },
      placeholder: "drops",
    });
    const renamed = tariffRename(rules, { ...cursor(offset), newName: "deliveries" });
    expect("edits" in renamed && renamed.edits.length).toBe(3);
  });

  it("refuses to rename a schema root", () => {
    const answer = tariffPrepareRename(rules, cursor(at("shipment.distance") + 2));
    expect("error" in answer).toBe(true);
  });

  it("gives inlay hints with their punctuation", () => {
    const hints = tariffInlays(rules, {
      source: SOURCE,
      context,
      kind: "tariff",
      from: 0,
      to: SOURCE.length,
    });
    expect(hints).toContainEqual({
      offset: at("miles =") + "miles".length,
      label: ": Distance",
      kind: "type",
    });
  });
});
