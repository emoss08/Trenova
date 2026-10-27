import assert from "node:assert/strict";
import { describe, it } from "node:test";
import {
  collapse,
  extractTariffDocs,
  fnParams,
  paramText,
  TariffExtractError,
} from "./extract-tariff.mjs";

const templates = (source) => extractTariffDocs(source, "std.tariff").map((d) => d.message);

describe("extractTariffDocs", () => {
  it("collapses each doc block into one template, with the line it starts on", () => {
    assert.deepEqual(
      extractTariffDocs(
        [
          'library "std.ltl"',
          "",
          "/// Less-than-truckload pricing: weight breaks and",
          "///   the density scale.",
          "",
          "/// One weight break.",
          "type Break = {",
          "  /// The lightest weight",
          "  /// the break prices.",
          "  from: Weight,",
          "}",
        ].join("\n"),
        "ltl.tariff",
      ),
      [
        { message: "Less-than-truckload pricing: weight breaks and the density scale.", line: 3 },
        { message: "One weight break.", line: 6 },
        { message: "The lightest weight the break prices.", line: 8 },
      ],
    );
  });

  it("splits blocks at a blank line and at a plain comment, and skips //// and code", () => {
    assert.deepEqual(
      templates(
        [
          'library "std.x"',
          "/// First.",
          "",
          "/// Second.",
          "// a plain comment",
          "/// Third.",
          "//// not a doc",
          'const A: Text = "/// not a doc either"',
        ].join("\n"),
      ),
      ["First.", "Second.", "Third."],
    );
  });

  it("gives each parameter the sentences of its function's doc that name it", () => {
    assert.deepEqual(
      templates(
        [
          'library "std.accessorials"',
          "/// Detention: time `dwell` runs past `free`, at `rate`. `$0` within the",
          "/// free time. Billed per `rate`.",
          "private fn detention(dwell: Duration, free: Duration = 2 h,",
          "    rate: Money/h, label: Text = \"a, b)\") -> Money {",
          "  return $0",
          "}",
        ].join("\n"),
      ),
      [
        "Detention: time `dwell` runs past `free`, at `rate`. `$0` within the free time. Billed per `rate`.",
        "Detention: time `dwell` runs past `free`, at `rate`.",
        "Detention: time `dwell` runs past `free`, at `rate`. Billed per `rate`.",
      ],
    );
  });

  it("reads only the standard library", () => {
    assert.deepEqual(templates('library "org.zones"\n/// The zones.\n'), []);
    assert.deepEqual(templates('tariff "Dry van"\n/// The rate.\ninput rate: Money\n'), []);
    assert.deepEqual(templates(""), []);
  });

  it("refuses a brace, which the hosts would read as a placeholder", () => {
    assert.throws(
      () => templates('library "std.x"\n\n/// Writes {0} here.\nconst A: Int = 1\n'),
      (error) => error instanceof TariffExtractError && error.message.startsWith("std.tariff:3: "),
    );
  });
});

describe("paramText", () => {
  it("matches the crate's sentence split", () => {
    const doc = "Time past `free`, rounded up to `increment`. Billed at `rate`.\nOr `free`.";
    assert.equal(paramText(doc, "rate"), "Billed at `rate`.");
    assert.equal(paramText(doc, "free"), "Time past `free`, rounded up to `increment`. Or `free`.");
    assert.equal(paramText(doc, "dwell"), null);
    assert.equal(paramText("With `allow: false` it bills.", "allow"), null);
  });
});

describe("fnParams", () => {
  it("names each parameter, past nested brackets and defaults", () => {
    assert.deepEqual(
      fnParams("fn f(a: List<Break>, b: { x: Int, y: Int }, c: Money = $5, d: Text = \"(,)\") -> Int"),
      ["a", "b", "c", "d"],
    );
    assert.deepEqual(fnParams("fn f() -> Int = 1"), []);
    assert.deepEqual(fnParams("fn f(a: Int"), []);
  });
});

describe("collapse", () => {
  it("joins lines with single spaces and trims", () => {
    assert.equal(collapse("  One\n  weight   break. "), "One weight break.");
  });
});
