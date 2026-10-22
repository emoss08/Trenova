import { rulesSync } from "@trenova/shared/rules/runtime";
import { tariffCheck } from "@trenova/shared/rules/tariff";
import { EMPTY_TARIFF_CONTEXT } from "@trenova/shared/types/tariff";
import { describe, expect, it } from "vitest";
import { appendBlock, inputDocEdits, newTestBlock } from "../source-edits";

const applyEdits = (
  source: string,
  edits: { range: { from: number; to: number }; insert: string }[],
) =>
  [...edits]
    .sort((a, b) => b.range.from - a.range.from)
    .reduce(
      (text, edit) => text.slice(0, edit.range.from) + edit.insert + text.slice(edit.range.to),
      source,
    );

describe("appendBlock", () => {
  it("adds a block after one blank line, replacing only the trailing whitespace", () => {
    const source = 'tariff "t"\n\ncharge "x" = $1\n\n\n';
    const edit = appendBlock(source, 'test "a" {\n}\n\n');
    expect(edit).toEqual({
      from: source.trimEnd().length,
      to: source.length,
      insert: '\n\ntest "a" {\n}\n',
    });
  });

  it("starts an empty document with the block itself", () => {
    expect(appendBlock("  \n", "x")).toEqual({ from: 0, to: 3, insert: "x\n" });
  });
});

describe("newTestBlock", () => {
  it("names a test so it does not clash, and quotes what a test name cannot hold", () => {
    expect(newTestBlock("library", ["a new test", "a new test 2"], "a new test")).toBe(
      'test "a new test 3" {\n  expect 1 + 1 == 2\n}\n',
    );
    expect(newTestBlock("tariff", [], 'say "{x}"')).toBe(
      "test \"say '(x)'\" {\n  given shipment.distance = 100 mi\n  expect total > $0\n}\n",
    );
  });
});

describe("inputDocEdits", () => {
  const rules = rulesSync();
  const source = [
    'tariff "t"',
    "",
    "/// Paid per loaded mile.",
    "input rate: Money/mi = $2.35/mi",
    "input minimumCharge: Money = $450",
    "input extraDropFee: Money = $75",
    "",
    'charge "Linehaul" (quantity: shipment.distance, rate: rate)',
    "minimum minimumCharge",
    'charge "Drops" = extraDropFee',
    "",
  ].join("\n");
  const inputs = tariffCheck(rules, { source, context: EMPTY_TARIFF_CONTEXT }).inputs;

  it("writes each description as a doc comment the checker reads back as the input's doc", () => {
    const edits = inputDocEdits(source, inputs, {
      minimumCharge: "The least a shipment pays.",
      extraDropFee: "  Per drop past the first,\n  each.  ",
    });
    const next = applyEdits(source, edits);

    expect(
      tariffCheck(rules, { source: next, context: EMPTY_TARIFF_CONTEXT }).inputs.map((input) => [
        input.name,
        input.doc,
      ]),
    ).toEqual([
      ["rate", "Paid per loaded mile."],
      ["minimumCharge", "The least a shipment pays."],
      ["extraDropFee", "Per drop past the first, each."],
    ]);
  });

  it("leaves an input with no description, a blank one, or no range alone", () => {
    expect(inputDocEdits(source, inputs, { rate: "   " })).toEqual([]);
    expect(
      inputDocEdits(source, [{ name: "rate", range: null }], { rate: "The rate per mile." }),
    ).toEqual([]);
  });

  it("indents the comment as the declaration is", () => {
    const indented = 'tariff "t"\n  input rate: Money = $1\n';
    const from = indented.indexOf("rate");
    expect(
      inputDocEdits(indented, [{ name: "rate", range: { from, to: from + 4 } }], { rate: "R." }),
    ).toEqual([
      {
        range: { from: indented.indexOf("  input"), to: indented.indexOf("  input") },
        insert: "  /// R.\n",
      },
    ]);
  });
});
