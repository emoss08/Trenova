import { translateMessage } from "@trenova/shared/i18n/message";
import { requireCatalog, setLocale } from "@trenova/shared/i18n/runtime";
import { formulaSchemaResponseSchema } from "@trenova/shared/types/formula-template";
import { afterEach, beforeAll, describe, expect, it } from "vitest";
import corpus from "../../../../../services/crates/rules/testdata/conformance.json";
import described from "../../../../../services/tms/internal/modules/rating/app/formula/testdata/shipment.describe.json";
import {
  formulaCatalog,
  formulaCheck,
  formulaFormat,
  formulaIdentifier,
  formulaSignature,
  type FormulaCheckSchema,
} from "./formula";
import { rulesSync } from "./runtime";

const rules = rulesSync();

// The schema and the template input of the formula.check example in
// docs/engineering/rules-engine.md, "The static formula operations".
const documentedSchema: FormulaCheckSchema = {
  variables: [
    {
      name: "weight",
      type: "number",
      description: "Total weight",
      category: "shipment",
      nullable: true,
      computed: false,
    },
    { name: "totalDistance", type: "number", nullable: false },
    { name: "stops", type: "array" },
    { name: "stops[].type", type: "string" },
  ],
  inputs: [{ name: "baseRate", type: "Number" }],
  operators: [],
};

beforeAll(() => {
  requireCatalog("core", "web");
});

afterEach(async () => {
  await setLocale("en");
});

describe("formulaCatalog", () => {
  it("lists the reference functions, the built-ins, the disabled names and the reserved words", () => {
    const catalog = formulaCatalog(rules);

    expect(catalog.functions.find((fn) => fn.name === "round")).toEqual({
      name: "round",
      signature: "round(x, decimals?)",
      description: {
        message: "Rounds a number to the given decimal places (default 0, between -12 and 12).",
      },
      example: { message: "round(3.14159, 2) // 3.14" },
      category: "rounding",
      operator: false,
      minArgs: 1,
      maxArgs: 2,
      returns: "Number",
      numericArgs: true,
      predicate: false,
    });
    expect(catalog.builtins.find((fn) => fn.name === "filter")).toMatchObject({
      signature: "filter(list, predicate)",
      category: "predicate",
      predicate: true,
    });
    expect(catalog.functions.some((fn) => fn.name === "filter")).toBe(false);
    expect(catalog.functions.find((fn) => fn.name === "startsWith")?.operator).toBe(true);
    expect(catalog.disabled.find((fn) => fn.name === "now")).toEqual({
      name: "now",
      reason: "nondeterministic",
      replacement: "ratingTime",
      message: {
        message:
          "{0} isn't available in formulas: a formula must price a shipment the same way every time it runs. Use {1}, the date the contract terms are read at",
        args: ["`now()`", "`ratingTime`"],
      },
    });
    expect(catalog.reserved).toEqual([...catalog.reserved].sort());
    expect(catalog.reserved).toEqual(expect.arrayContaining(["let", "nil", "and", "round"]));
  });

  it("is read once per loaded runtime", () => {
    expect(formulaCatalog(rules)).toBe(formulaCatalog(rules));
  });
});

describe("formulaCheck", () => {
  it("answers the documented example exactly", () => {
    const check = formulaCheck(rules, {
      source:
        'let linehaul = baseRate * totalDistance;\nlinehaul + weight * lookup("fuel", 1) + extra',
      expect: "amount",
      schema: documentedSchema,
      tables: [{ name: "fuel", axes: 1, banded: false }],
    });

    expect(check).toEqual({
      empty: false,
      pending: false,
      diagnostics: [
        {
          from: 52,
          to: 58,
          severity: "warning",
          code: "nullable",
          message:
            "{0} is empty on some shipments, and the formula would fail to rate them. Give it a fallback: {1}",
          args: ["`weight`", "coalesce(weight, 0)"],
          name: "weight",
          fixes: [
            {
              kind: "replace",
              label: { message: "Use {0}", args: ["coalesce(weight, 0)"] },
              changes: [{ from: 52, to: 58, insert: "coalesce(weight, 0)" }],
            },
          ],
        },
        {
          from: 81,
          to: 86,
          severity: "error",
          code: "unknown-variable",
          message:
            "{0} isn't a shipment field or one of this template's inputs. Declare it as an input, or pick a field from the reference.",
          args: ["`extra`"],
          name: "extra",
          fixes: [
            {
              kind: "declare",
              label: { message: "Declare {0} as a {1} input", args: ["extra", "Number"] },
              name: "extra",
              type: "Number",
            },
          ],
        },
      ],
      resultType: "Number",
      reads: ["baseRate", "totalDistance", "weight"],
      schemaReads: ["totalDistance", "weight"],
      inputReads: ["baseRate"],
      functionCalls: ["lookup"],
      bindings: [
        { name: "linehaul", type: "Number", reads: 1, from: 0, to: 39, nameFrom: 4, nameTo: 12 },
      ],
      statements: [
        {
          index: 0,
          kind: "let",
          name: "linehaul",
          code: "let linehaul = baseRate * totalDistance",
          from: 0,
          to: 39,
        },
        {
          index: 1,
          kind: "result",
          code: 'linehaul + weight * lookup("fuel", 1) + extra',
          from: 41,
          to: 86,
        },
      ],
      lookupTables: [{ name: "fuel", function: "lookup", from: 68, to: 74 }],
    });
  });

  it("counts positions in UTF-16 code units, as the editor does", () => {
    const source = 'len("🚚🚚") + extra';
    const [unknown] = formulaCheck(rules, { source, schema: documentedSchema }).diagnostics;

    expect(unknown).toMatchObject({ code: "unknown-variable", name: "extra" });
    expect(source.slice(unknown.from, unknown.to)).toBe("extra");
  });

  it("holds back unknown names while the schema is pending, and still checks the language", () => {
    const check = formulaCheck(rules, { source: "extra + round(1, 2, 3)", schema: null });

    expect(check.pending).toBe(true);
    expect(check.diagnostics.map((diagnostic) => diagnostic.code)).toEqual(["arity"]);
  });

  it("checks a formula at the 1,000-node limit well inside the linter's 300 ms delay", () => {
    // Each line is about twelve nodes; 82 of them and their sum come to just under the
    // parser's 1,000-node limit, which two more lines cross.
    const formula = (lines: number) => {
      const lets = Array.from(
        { length: lines },
        (_, i) => `let v${i} = round(weight * ${i} + totalDistance / ${i + 1}, 2)`,
      );
      const sum = Array.from({ length: lines }, (_, i) => `v${i}`).join(" + ");
      return `${lets.join(";\n")};\n${sum}`;
    };
    const codes = (source: string) =>
      formulaCheck(rules, { source, schema: documentedSchema }).diagnostics.map(
        (diagnostic) => diagnostic.code,
      );
    const largest = formula(82);
    expect(codes(largest)).not.toContain("limit");
    expect(codes(formula(84))).toContain("limit");

    const runs = Array.from({ length: 15 }, (_, run) => {
      const started = performance.now();
      formulaCheck(rules, { source: `${largest}${" ".repeat(run)}`, schema: documentedSchema });
      return performance.now() - started;
    }).sort((a, b) => a - b);
    expect(runs[7]).toBeLessThan(50);
  });

  it("says a formula of spaces and comments is empty", () => {
    const check = formulaCheck(rules, { source: "  // nothing yet\n", schema: documentedSchema });

    expect(check.empty).toBe(true);
    expect(check.diagnostics).toEqual([
      {
        from: 0,
        to: 17,
        severity: "error",
        code: "empty",
        message: "The formula is empty",
        fixes: [],
      },
    ]);
  });

  it("asks a predicate for a yes or no", () => {
    expect(
      formulaCheck(rules, { source: "totalDistance > 500", expect: "predicate", schema: null })
        .diagnostics,
    ).toEqual([]);
    expect(
      formulaCheck(rules, { source: "totalDistance > 500", expect: "amount", schema: null })
        .diagnostics[0]?.code,
    ).toBe("result-type");
  });

  it("gives a message whose nested phrase translates with it", async () => {
    const [diagnostic] = formulaCheck(rules, {
      source: 'duration("1h")',
      schema: documentedSchema,
    }).diagnostics;
    expect(diagnostic).toMatchObject({
      code: "result-type",
      message: "The formula must come to an amount, but this comes to {0}",
      args: [{ message: "a duration" }],
    });

    expect(translateMessage(diagnostic)).toBe(
      "The formula must come to an amount, but this comes to a duration",
    );
    await setLocale("es");
    expect(translateMessage(diagnostic)).toBe(
      "La fórmula debe dar un importe, pero esto da una duración",
    );
  });
});

describe("formulaFormat", () => {
  it("formats a formula, and formatting its own output changes nothing", () => {
    expect(formulaFormat(rules, { source: "a+b" })).toEqual({
      ok: true,
      text: "a + b",
      changed: true,
    });
    expect(formulaFormat(rules, { source: "a + b" })).toEqual({
      ok: true,
      text: "a + b",
      changed: false,
    });
  });

  it("leaves alone a formula it would lose text from", () => {
    expect(formulaFormat(rules, { source: "  " })).toEqual({ ok: false, reason: "empty" });
    expect(formulaFormat(rules, { source: "a +" })).toEqual({ ok: false, reason: "syntax" });
    expect(formulaFormat(rules, { source: "a + b // why" })).toEqual({
      ok: false,
      reason: "comments",
    });
  });

  it("breaks a line past the width it is given", () => {
    const result = formulaFormat(rules, { source: "alpha + beta + gamma", width: 12 });
    expect(result).toEqual({ ok: true, text: "alpha\n  + beta\n  + gamma", changed: true });
  });

  it("keeps the space that stops a number taking the dot of a member after it", () => {
    const result = formulaFormat(rules, { source: "g..27\n.g4\n..54\n..5.", width: 71 });
    expect(result).toEqual({ ok: true, text: "g..27 .g4..54..5.", changed: true });
    expect(formulaFormat(rules, { source: "g..27 .g4..54..5.", width: 71 })).toEqual({
      ok: true,
      text: "g..27 .g4..54..5.",
      changed: false,
    });
  });
});

describe("formulaSignature", () => {
  it("names the call around the cursor, the argument it is in and its parentheses", () => {
    expect(formulaSignature(rules, { source: "round(totalWeight, 2)", position: 19 })).toEqual({
      name: "round",
      active: 1,
      from: 5,
      to: 21,
    });
  });

  it("is null outside any call", () => {
    expect(formulaSignature(rules, { source: "totalWeight * 2", position: 3 })).toBeNull();
  });
});

describe("formulaIdentifier", () => {
  it("reads a dotted path as one name", () => {
    expect(
      formulaIdentifier(rules, { source: "customer.name + 1", position: 3, schema: null }),
    ).toEqual({ name: "customer.name", kind: "variable", from: 0, to: 13 });
  });

  it("tells an input from a field with the schema, and an item's field without one", () => {
    expect(
      formulaIdentifier(rules, { source: "baseRate * 2", position: 2, schema: documentedSchema }),
    ).toEqual({ name: "baseRate", kind: "input", from: 0, to: 8 });
    expect(
      formulaIdentifier(rules, {
        source: 'count(stops, .type == "Pickup")',
        position: 14,
        schema: null,
      }),
    ).toEqual({ name: "type", kind: "field", from: 14, to: 18 });
  });

  it("is null on anything but a name, and inside a string", () => {
    expect(formulaIdentifier(rules, { source: "a * b", position: 2 })).toBeNull();
    expect(formulaIdentifier(rules, { source: '"weight"', position: 3 })).toBeNull();
  });
});

describe("the conformance corpus, end to end through the module", () => {
  const schema: FormulaCheckSchema = {
    variables: formulaSchemaResponseSchema.parse(described).variables,
    inputs: corpus.inputs.map((input) => ({ name: input.name, type: input.type })),
  };
  type Case = { id: string; expression: string; accepted: boolean; code?: string; from?: number };
  const cases = new Map((corpus.cases as Case[]).map((tc) => [tc.id, tc]));
  const ids = [
    "per-mile",
    "chained-comparison",
    "text-plus-number",
    "result-is-a-test",
    "unknown-name-on-a-later-line",
    "unknown-name-after-a-non-ascii-name",
  ];

  it.each(ids)("%s", (id) => {
    const tc = cases.get(id);
    if (!tc) throw new Error(`the corpus has no case ${id}`);
    const errors = formulaCheck(rules, { source: tc.expression, schema })
      .diagnostics.filter((diagnostic) => diagnostic.severity === "error")
      .map((diagnostic) => ({ code: diagnostic.code, from: diagnostic.from }));

    if (tc.accepted) {
      expect(errors).toEqual([]);
    } else {
      expect(errors).toContainEqual({ code: tc.code, from: tc.from });
    }
  });
});
