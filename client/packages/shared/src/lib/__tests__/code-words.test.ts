import { describe, expect, it } from "vitest";
import { codeWords } from "../code-words";

describe("codeWords", () => {
  it("splits a PascalCase code into title-case words", () => {
    expect(codeWords("AgentTurn")).toBe("Agent Turn");
    expect(codeWords("DocumentIntelligenceExtract")).toBe("Document Intelligence Extract");
  });

  it("keeps an initialism whole, alone or inside a code", () => {
    expect(codeWords("POD")).toBe("POD");
    expect(codeWords("MissingBOL")).toBe("Missing BOL");
    expect(codeWords("EDIInbound")).toBe("EDI Inbound");
  });

  it("reads snake, kebab and constant-case codes as words", () => {
    expect(codeWords("amount_mismatch")).toBe("Amount Mismatch");
    expect(codeWords("needs-attention")).toBe("Needs Attention");
    expect(codeWords("IN_TRANSIT")).toBe("In Transit");
  });

  it("capitalizes a lower-case single word", () => {
    expect(codeWords("draft")).toBe("Draft");
  });

  it("returns an empty code unchanged", () => {
    expect(codeWords("")).toBe("");
  });
});
