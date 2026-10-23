import { sourceProposalSchema, pageDraftSchema } from "@/types/page-draft";
import { rulesSync } from "@trenova/shared/rules/runtime";
import { describe, expect, it } from "vitest";
import {
  MAX_SOURCE_BYTES,
  applyProposal,
  formattedOrAsWritten,
  formulaDraftFromEditor,
} from "../formula-draft";

const rules = rulesSync();

const SOURCE =
  'tariff "Per mile"\n\ninput rate: Money/mi = $2.35/mi\n\ncharge "Linehaul" (quantity: shipment.distance, rate: rate)\n';

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
      formula: { templateId: "fmt_1", name: "Per mile", source: SOURCE },
    });
  });

  it("hands Nova a library in its own payload, and leaves an unsaved file unnamed", () => {
    const draft = formulaDraftFromEditor({
      kind: "library",
      recordId: "fl_1",
      source: 'library "org.x"\n',
    });
    expect(pageDraftSchema.safeParse(draft).success).toBe(true);
    expect(draft).toEqual({
      surface: "formula_library",
      formulaLibrary: { libraryId: "fl_1", source: 'library "org.x"\n' },
    });
    expect(formulaDraftFromEditor({ kind: "tariff", source: "" })).toEqual({
      surface: "formula",
      formula: { source: "" },
    });
  });

  it("keeps the name within each surface's limit: 100 for a template, 64 for a library", () => {
    const name = "n".repeat(120);
    expect(formulaDraftFromEditor({ kind: "tariff", name, source: "" }).formula?.name).toHaveLength(
      100,
    );
    expect(
      formulaDraftFromEditor({ kind: "library", name, source: "" }).formulaLibrary?.name,
    ).toHaveLength(64);
  });

  it("keeps the source within the module's limit in UTF-8 bytes, whole characters only", () => {
    const draft = formulaDraftFromEditor({
      kind: "tariff",
      source: "é".repeat(MAX_SOURCE_BYTES),
    });
    expect(draft.formula?.source).toBe("é".repeat(MAX_SOURCE_BYTES / 2));
  });
});

describe("formattedOrAsWritten", () => {
  it("formats a source that parses and leaves one with syntax errors as written", () => {
    expect(formattedOrAsWritten(rules, 'tariff "t"\nlet a=1+2\n')).toBe(
      'tariff "t"\nlet a = 1 + 2\n',
    );
    expect(formattedOrAsWritten(rules, 'tariff "t"\nlet a = 1 +\n')).toBe(
      'tariff "t"\nlet a = 1 +\n',
    );
  });
});

describe("applyProposal", () => {
  const proposal = sourceProposalSchema.parse({
    name: "Per mile",
    description: "Linehaul by the loaded mile",
    source: 'tariff "Per mile"\nlet a=1\ncharge "Linehaul" = $1*a\n',
    explanation: "Bills a linehaul.",
    inputs: [],
    imports: [],
    charges: ["Linehaul"],
    warnings: [],
    tests: { total: 1, passed: 1 },
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
