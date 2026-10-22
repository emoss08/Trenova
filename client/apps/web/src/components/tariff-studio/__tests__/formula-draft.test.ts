import { formulaProposalSchema, pageDraftSchema } from "@/types/page-draft";
import { rulesSync } from "@trenova/shared/rules/runtime";
import { describe, expect, it } from "vitest";
import {
  MAX_SOURCE_LENGTH,
  applyProposal,
  formattedOrAsWritten,
  formulaDraftFromEditor,
} from "../formula-draft";

const rules = rulesSync();

const SOURCE = 'tariff "Per mile"\n\ninput rate: Money/mi = $2.35/mi\n\ncharge "Linehaul" (quantity: shipment.distance, rate: rate)\n';

describe("formulaDraftFromEditor", () => {
  it("hands Nova a template's file, in the shape pagedraft.Formula reads", () => {
    const draft = formulaDraftFromEditor({
      kind: "tariff",
      recordId: "fmt_1",
      name: " Per mile ",
      description: "",
      source: SOURCE,
    });

    expect(pageDraftSchema.safeParse(draft).success).toBe(true);
    expect(draft).toEqual({
      surface: "formula",
      formula: { templateId: "fmt_1", kind: "tariff", name: "Per mile", source: SOURCE },
    });
  });

  it("names a library by its own ID and surface, and leaves an unsaved file unnamed", () => {
    expect(
      formulaDraftFromEditor({ kind: "library", recordId: "flb_1", source: "library \"x\"\n" }),
    ).toEqual({
      surface: "formula_library",
      formula: { libraryId: "flb_1", kind: "library", source: "library \"x\"\n" },
    });
    expect(formulaDraftFromEditor({ kind: "tariff", source: "" }).formula).toEqual({
      kind: "tariff",
      source: "",
    });
  });

  it("keeps the source within the module's limit", () => {
    const draft = formulaDraftFromEditor({ kind: "tariff", source: "x".repeat(MAX_SOURCE_LENGTH + 10) });
    expect(draft.formula?.source.length).toBe(MAX_SOURCE_LENGTH);
  });
});

describe("formattedOrAsWritten", () => {
  it("formats a source that parses and leaves one with syntax errors as written", () => {
    expect(formattedOrAsWritten(rules, 'tariff "t"\nlet a=1+2\n')).toBe('tariff "t"\nlet a = 1 + 2\n');
    expect(formattedOrAsWritten(rules, 'tariff "t"\nlet a = 1 +\n')).toBe('tariff "t"\nlet a = 1 +\n');
  });
});

describe("applyProposal", () => {
  const proposal = formulaProposalSchema.parse({
    name: "Per mile",
    description: "Linehaul by the loaded mile",
    source: 'tariff "Per mile"\nlet a=1\ncharge "Linehaul" = $1*a\n',
    explanation: "Bills a linehaul.",
    check: { ok: true, diagnostics: [] },
    tests: [],
  });

  it("takes Nova's source, formatted, and its name and description only where none is set", () => {
    expect(applyProposal(rules, { name: "", description: "", source: "" }, proposal)).toEqual({
      name: "Per mile",
      description: "Linehaul by the loaded mile",
      source: 'tariff "Per mile"\nlet a = 1\ncharge "Linehaul" = $1 * a\n',
    });
    expect(
      applyProposal(rules, { name: "Mine", description: "My words", source: SOURCE }, proposal),
    ).toMatchObject({ name: "Mine", description: "My words" });
  });
});
