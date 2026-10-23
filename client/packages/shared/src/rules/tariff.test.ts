import {
  EMPTY_TARIFF_CONTEXT,
  type TariffContext,
  type TariffType,
} from "@trenova/shared/types/tariff";
import { formulaPreviewResponseSchema } from "@trenova/shared/types/formula-template";
import { describe, expect, it } from "vitest";
import { rulesSync } from "./runtime";
import {
  tariffCatalog,
  tariffCheck,
  tariffComplete,
  tariffActions,
  tariffDefinition,
  tariffFormat,
  tariffHover,
  tariffInlays,
  tariffPrepareRename,
  tariffReferences,
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
        constraint: null,
      },
      {
        name: "minimumCharge",
        doc: null,
        type: "Money",
        descriptor: { kind: "money" },
        required: false,
        default: { $q: "450", u: "$" },
        constraint: null,
      },
      {
        name: "extraDropFee",
        doc: null,
        type: "Money",
        descriptor: { kind: "money" },
        required: false,
        default: { $q: "75", u: "$" },
        constraint: null,
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

  it("sends a standard library field's doc as a message and a declared field's as written", () => {
    const source = `tariff "t"

use std.ltl.{ Break }

/// A delivery zone.
type Zone = {
  /// The zone's name.
  name: Text,
}

/// The LTL breaks.
input breaks: List<Break> = []
/// The zone.
input zone: Zone = { name: "a" }

charge "x" = $1
`;
    const [breaks, zone] = tariffCheck(rules, { source, context }).inputs;
    expect(breaks?.doc).toBe("The LTL breaks.");
    const fieldDocs = (descriptor: TariffType | null | undefined): unknown =>
      descriptor?.kind === "list"
        ? fieldDocs(descriptor.item)
        : descriptor?.kind === "record"
          ? descriptor.fields.map((field) => [field.name, field.doc])
          : null;
    expect(fieldDocs(breaks?.descriptor)).toEqual([
      ["from", { message: "The lightest weight the break prices." }],
      ["rate", { message: "What the break charges per hundredweight." }],
      ["minimum", { message: "The least a shipment priced at the break pays, if anything." }],
    ]);
    expect(fieldDocs(zone?.descriptor)).toEqual([["name", "The zone's name."]]);
  });

  it("folds runs of comments and of imports as well as blocks", () => {
    const source = 'tariff "t"\n\n// one\n// two\nuse std.ltl\nuse std.fuel\n\ncharge "x" = $1\n';
    const folding = tariffCheck(rules, { source, context }).outline?.folding ?? [];
    expect(
      folding.map((range) => [range.kind, source.slice(range.range.from, range.range.to)]),
    ).toEqual([
      ["comment", "// one\n// two"],
      ["imports", "use std.ltl\nuse std.fuel"],
    ]);
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
          {
            range: { from: at("total == $1"), to: at("total == $1") + 5 },
            value: { $q: "450", u: "$" },
            display: "$450.00",
          },
        ],
      },
    ]);
    expect(failed?.bill?.total).toEqual({ $q: "450", u: "$" });
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
        constraint: null,
        value: null,
      },
    });
  });

  it("hovers a schema field with its doc as a catalog message", () => {
    expect(
      tariffHover(rules, cursor(at("shipment.distance") + "shipment.di".length)),
    ).toMatchObject({
      contents: { signature: "distance: Distance", doc: { message: expect.any(String) } },
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

// The reference's examples of the new language (formula-language.md, "Input constraints",
// "Choosing between methods", "Properties" and "Coverage"), in one file: a bounded rate and a
// list-bounded service level, a `best` of three methods (one refusing on the sample load), a
// weight break that inverts at 1,000 lb, and a test that reaches only the light branch.
const LTL = `tariff "LTL"

/// Paid per hundredweight.
input rate: Money/lb = $0.31/lb where $0.05/lb..=$2/lb
input service: Text = "Standard" where ["Standard", "Guaranteed"]

let weight = shipment.weight ?? 0 lb
if weight >= 1_000 lb {
  charge "Freight" = weight * $0.25/lb
} else {
  charge "Freight" = weight * rate
}
best "Linehaul" {
  option "Per mile" {
    charge "Linehaul" (quantity: shipment.distance, rate: $2/mi)
  }
  option "Flat" {
    charge "Flat rate" = $900
  }
  option "Volume" {
    require weight >= 6_000 lb else "Volume rates start at 6,000 lb"
    charge "Volume" = weight * $0.10/lb
  }
}
note "Service: {service}"

property "heavier never bills less" {
  vary shipment.weight in 1 lb..40_000 lb
  expect total nondecreasing in shipment.weight
}

test "a light load" {
  given shipment.weight = 500 lb
  given shipment.distance = 100 mi
  expect charge("Freight") == $155
}
`;

const ltlAt = (text: string) => LTL.indexOf(text);

describe("input constraints", () => {
  it("reports each input's folded where clause", () => {
    const check = tariffCheck(rules, { source: LTL, kind: "tariff", context });
    expect(check.inputs.map((input) => [input.name, input.constraint])).toEqual([
      [
        "rate",
        {
          range: {
            min: { $q: "0.05", u: "$/lb" },
            max: { $q: "2", u: "$/lb" },
            maxInclusive: true,
          },
        },
      ],
      ["service", { oneOf: ["Standard", "Guaranteed"] }],
    ]);
  });

  it("refuses a contract value outside the bounds with an input-value finding at its path", () => {
    const check = tariffCheck(rules, {
      source: LTL,
      kind: "tariff",
      context,
      values: { rule: { rate: { $q: "3", u: "$/lb" }, service: "Rush" } },
    });
    expect(
      check.diagnostics
        .filter((finding) => finding.code === "input-value")
        .map((finding) => [finding.path, finding.message]),
    ).toEqual([
      [
        "rule.rate",
        {
          message: "{0} must be from {1} to {2}; {3} is outside it",
          args: ["`rate`", "$0.05/lb", "$2.00/lb", "$3.00/lb"],
        },
      ],
      [
        "rule.service",
        {
          message: "{0} must be one of {1}; {2} isn't",
          args: ["`service`", "[Standard, Guaranteed]", "Rush"],
        },
      ],
    ]);
  });

  it("hovers a constrained input with its bound in words, apart from its doc", () => {
    const use = ltlAt("weight * rate") + "weight * ".length;
    expect(
      tariffHover(rules, { source: LTL, context, kind: "tariff", offset: use + 1 })?.contents,
    ).toMatchObject({
      doc: "Paid per hundredweight.",
      constraint: { message: "from {0} to {1}", args: ["$0.05/lb", "$2.00/lb"] },
    });
  });
});

describe("properties and coverage", () => {
  const run = tariffTest(rules, { source: LTL, context });

  it("runs a property beside the tests and shrinks its failure to the out-of-order pair", () => {
    expect(run.tests.map((test) => [test.kind, test.name, test.status])).toEqual([
      ["property", "heavier never bills less", "failed"],
      ["test", "a light load", "passed"],
    ]);
    const property = run.tests[0];
    expect(property?.cases).toBe(24);
    expect(property?.skipped).toBe(0);
    expect(property?.counterexample?.values).toEqual([
      { target: "shipment.weight", display: "999 lb", value: { $q: "999", u: "lb" } },
    ]);
    expect(property?.counterexample?.monotone).toEqual({
      target: "shipment.weight",
      expression: "total",
      direction: "nondecreasing",
      before: {
        at: { display: "999 lb", value: { $q: "999", u: "lb" } },
        result: expect.objectContaining({ value: expect.objectContaining({ u: "$" }) }),
      },
      after: {
        at: { display: "1000 lb", value: { $q: "1000", u: "lb" } },
        result: expect.objectContaining({ value: expect.objectContaining({ u: "$" }) }),
      },
    });
    expect(property?.counterexample?.test).toMatch(
      /^test "heavier never bills less: 999 lb" \{\n {2}given shipment\.weight = 999 lb\n/,
    );
  });

  it("gives a test one case and no counterexample", () => {
    expect(run.tests[1]).toMatchObject({ cases: 1, skipped: 0, counterexample: null });
  });

  it("counts the regions every case reached, in source order", () => {
    const coverage = run.coverage;
    expect(coverage?.total).toBe(coverage?.regions.length);
    expect(coverage?.reached).toBe(coverage?.regions.filter((region) => region.hits > 0).length);
    expect(coverage?.regions.map((region) => region.kind)).toEqual([
      "if",
      "charge",
      "else",
      "charge",
      "option",
      "charge",
      "option",
      "charge",
      "option",
      "refusal",
      "charge",
    ]);
    const starts = coverage?.regions.map((region) => region.range.from) ?? [];
    expect([...starts].sort((a, b) => a - b)).toEqual(starts);
  });

  it("sends no coverage when only some tests ran", () => {
    expect(
      tariffTest(rules, { source: LTL, context, names: ["a light load"] }).coverage,
    ).toBeNull();
  });
});

describe("choices", () => {
  it("reads every option of a best with its subtotal or refusal, as the preview carries it", () => {
    const answer = rules.call("tariff.evaluate", {
      context: {},
      record: null,
      trace: true,
      programs: [{ id: "ltl", source: LTL, inputs: {} }],
    }) as { results: Record<string, unknown>[] };
    const preview = formulaPreviewResponseSchema.parse(answer.results[0]);
    expect(preview.choices).toEqual([
      {
        label: "Linehaul",
        kind: "best",
        range: expect.objectContaining({ from: ltlAt('best "Linehaul"') }),
        chosen: "Per mile",
        options: [
          expect.objectContaining({ label: "Per mile", status: "chosen", refusal: null }),
          expect.objectContaining({
            label: "Flat",
            status: "lost",
            subtotal: { $q: "900", u: "$" },
            refusal: null,
          }),
          expect.objectContaining({
            label: "Volume",
            status: "refused",
            subtotal: null,
            refusal: expect.objectContaining({ message: "Volume rates start at 6,000 lb" }),
          }),
        ],
      },
    ]);
    expect(preview.trace).toContainEqual({
      kind: "choice",
      label: "Linehaul",
      chosen: "Per mile",
      range: expect.objectContaining({ from: ltlAt('best "Linehaul"') }),
    });
  });
});

describe("tariffReferences", () => {
  const cursor = (offset: number) => ({ source: SOURCE, context, kind: "tariff" as const, offset });

  it("lists a binding's declaration and every read, in source order", () => {
    const uses = tariffReferences(rules, cursor(at("drops > 1") + 1));
    expect(uses).toEqual([
      { range: { from: at("drops ="), to: at("drops =") + 5 }, kind: "declaration", module: null },
      { range: { from: at("drops > 1"), to: at("drops > 1") + 5 }, kind: "read", module: null },
      {
        range: { from: at("drops - 1"), to: at("drops - 1") + 5 },
        kind: "read",
        module: null,
      },
    ]);
  });

  it("leaves the declaration out when asked", () => {
    const uses = tariffReferences(rules, {
      ...cursor(at("drops > 1") + 1),
      includeDeclaration: false,
    });
    expect(uses.map((use) => use.kind)).toEqual(["read", "read"]);
  });

  it("marks an assignment as a write", () => {
    const source =
      'tariff "t"\nvar extra = $0\nif shipment.distance > 10 mi {\n  extra = $5\n}\ncharge "x" = extra\n';
    const uses = tariffReferences(rules, {
      source,
      context,
      kind: "tariff",
      offset: source.indexOf("extra = $5") + 1,
    });
    expect(uses.map((use) => use.kind)).toEqual(["declaration", "write", "read"]);
  });

  it("puts an imported name's declaration in its library's source", () => {
    const library = 'library "org.fees"\nfn fee() -> Money = $5\n';
    const source = 'tariff "t"\nuse org.fees.{ fee }\ncharge "a" = fee()\n';
    const uses = tariffReferences(rules, {
      source,
      context: { ...context, libraries: [{ path: "org.fees", version: "v1", source: library }] },
      kind: "tariff",
      offset: source.indexOf("fee()") + 1,
    });
    expect(uses[0]).toEqual({
      range: { from: library.indexOf("fee("), to: library.indexOf("fee(") + 3 },
      kind: "declaration",
      module: "org.fees",
    });
    expect(uses.slice(1).every((use) => use.module === null)).toBe(true);
  });

  it("answers nothing off a name", () => {
    expect(tariffReferences(rules, cursor(at("\n\nlet miles") + 1))).toEqual([]);
  });
});

describe("tariffActions", () => {
  const actionsAt = (source: string, from: number, to = from) =>
    tariffActions(rules, { source, context, kind: "tariff", from, to });

  it("promotes a charge's amount to an input named after the charge, selecting the new name", () => {
    const source = 'tariff "t"\ninput rate: Money/mi = $2/mi\ncharge "Flat rate" = $900\n';
    const [action, ...rest] = actionsAt(source, source.indexOf("$900") + 1);
    expect(rest).toEqual([]);
    expect(action?.id).toBe("promote-input");
    expect(action?.title).toEqual({ message: "Promote {0} to an input", args: ["`$900`"] });
    let next = source;
    for (const edit of [...(action?.edits ?? [])].sort((a, b) => b.range.from - a.range.from)) {
      next = next.slice(0, edit.range.from) + edit.insert + next.slice(edit.range.to);
    }
    expect(next).toBe(
      'tariff "t"\ninput rate: Money/mi = $2/mi\ninput flatRateAmount: Money = $900\ncharge "Flat rate" = flatRateAmount\n',
    );
    expect(next.slice(action?.select?.from, action?.select?.to)).toBe("flatRateAmount");
  });

  it("offers to promote every equal literal at once", () => {
    const source = 'tariff "t"\ncharge "a" = $5\ncharge "b" = $5\n';
    expect(actionsAt(source, source.indexOf("$5") + 1).map((action) => action.id)).toEqual([
      "promote-input",
      "promote-input-all",
    ]);
  });

  it("offers nothing off a literal", () => {
    expect(actionsAt(SOURCE, 2)).toEqual([]);
  });
});
