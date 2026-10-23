import { tariffCodeSpans } from "@trenova/shared/lib/highlight/tariff";
import type { TariffToken } from "@trenova/shared/rules/tariff";
import { describe, expect, it } from "vitest";

describe("tariffCodeSpans", () => {
  it("draws each token class in its Gruvbox role and leaves a plain name in the ink", () => {
    // `let c = shipment.distance.in(mi)`, a text with an interpolation and an escape, a charge
    // code and a comment, each token classed as `tariff.tokens` classes it.
    const tokens: TariffToken[] = [
      { from: 0, to: 3, kind: "keyword" },
      { from: 4, to: 5, kind: "name" },
      { from: 6, to: 7, kind: "punct" },
      { from: 8, to: 16, kind: "name" },
      { from: 16, to: 17, kind: "punct" },
      { from: 17, to: 25, kind: "field" },
      { from: 25, to: 26, kind: "punct" },
      { from: 26, to: 28, kind: "function" },
      { from: 28, to: 29, kind: "punct" },
      { from: 29, to: 31, kind: "unit" },
      { from: 31, to: 32, kind: "punct" },
      { from: 33, to: 34, kind: "text" },
      { from: 34, to: 35, kind: "interpolation" },
      { from: 35, to: 36, kind: "number" },
      { from: 36, to: 37, kind: "interpolation" },
      { from: 37, to: 39, kind: "escape" },
      { from: 39, to: 40, kind: "text" },
      { from: 41, to: 44, kind: "code" },
      { from: 45, to: 52, kind: "comment" },
    ];

    expect(tariffCodeSpans(tokens).map((span) => span.role)).toEqual([
      "keyword",
      undefined,
      "operator",
      undefined,
      "operator",
      "property",
      "operator",
      "function",
      "operator",
      "type",
      "operator",
      "string",
      "meta",
      "number",
      "meta",
      "meta",
      "string",
      "meta",
      "comment",
    ]);
  });
});
