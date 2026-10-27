import { highlightSpans, lineText } from "@trenova/shared/lib/highlight/code";
import {
  looksLikeX12,
  X12_HIGHLIGHT_LIMIT,
  x12Separators,
  x12Spans,
} from "@trenova/shared/lib/highlight/x12";
import { describe, expect, it } from "vitest";

const ISA =
  "ISA*00*          *00*          *01*007912345      *ZZ*TRNVLOGISTICS  *260302*0815*^*00401*000000700*0*T*>~";

function roles(text: string): [string, string | undefined][] {
  return x12Spans(text).map((span) => [text.slice(span.from, span.to), span.role]);
}

describe("x12Separators", () => {
  it("reads the interchange's own separators from its ISA", () => {
    expect(x12Separators(`${ISA}GS*SM*ACME~`)).toEqual({
      element: "*",
      component: ">",
      repetition: "^",
      segment: "~",
    });
  });

  it("reads a pipe-separated interchange that ends each segment with a line", () => {
    const isa =
      "ISA|00|          |00|          |01|007912345      |ZZ|TRNVLOGISTICS  |260302|0815|U|00401|000000700|0|T|:\n";
    expect(x12Separators(`${isa}GS|SM|ACME\n`)).toEqual({
      element: "|",
      component: ":",
      repetition: "^",
      segment: "\n",
    });
  });

  it("reads a squeezed ISA, whose values are not padded to their width", () => {
    const squeezed = "ISA*00**00**ZZ*PARTNER*ZZ*TRENOVA*260107*1200*U*00401*104*0*P*:!\r\nGS*QM!";
    expect(x12Separators(squeezed)).toEqual({
      element: "*",
      component: ":",
      repetition: "^",
      segment: "!",
    });
  });

  it("uses the usual separators for text without an ISA", () => {
    expect(x12Separators("ST*204*0001~")).toEqual({
      element: "*",
      component: ">",
      repetition: "^",
      segment: "~",
    });
  });
});

describe("x12Spans", () => {
  it("colours envelope and transaction segment IDs apart, separators and numbers", () => {
    expect(roles("ST*204*0001~B2**TRNV**AM-50001~L3*20000*G~SE*4*0001~")).toEqual([
      ["ST", "meta"],
      ["*", "operator"],
      ["204", "number"],
      ["*", "operator"],
      ["0001", "number"],
      ["~", "operator"],
      ["B2", "keyword"],
      ["*", "operator"],
      ["*", "operator"],
      ["*", "operator"],
      ["*", "operator"],
      ["~", "operator"],
      ["L3", "keyword"],
      ["*", "operator"],
      ["20000", "number"],
      ["*", "operator"],
      ["~", "operator"],
      ["SE", "meta"],
      ["*", "operator"],
      ["4", "number"],
      ["*", "operator"],
      ["0001", "number"],
      ["~", "operator"],
    ]);
  });

  it("starts a segment after a line break that follows the terminator", () => {
    const text = "N1*SH*ACME PLANT 3~\nN3*400 FOUNDRY RD~";
    expect(roles(text).filter(([, role]) => role === "keyword")).toEqual([
      ["N1", "keyword"],
      ["N3", "keyword"],
    ]);
    expect(highlightSpans(text, x12Spans(text)).map(lineText)).toEqual([
      "N1*SH*ACME PLANT 3~",
      "N3*400 FOUNDRY RD~",
    ]);
  });

  it("leaves an interchange past the limit plain", () => {
    const segment = "LX*1~";
    const text = `${ISA}${segment.repeat(Math.ceil(X12_HIGHLIGHT_LIMIT / segment.length))}`;
    expect(x12Spans(text)).toEqual([]);
    expect(x12Spans(`${ISA}LX*1~`)).not.toEqual([]);
  });

  it("leaves text that is not X12 plain", () => {
    expect(looksLikeX12('{"order": "A-1"}')).toBe(false);
    expect(x12Spans("order,pickup\nA-1,03/10/2026")).toEqual([]);
  });
});
