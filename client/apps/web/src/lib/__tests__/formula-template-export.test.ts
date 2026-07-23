import { describe, expect, it } from "vitest";
import {
  slugify,
  buildTemplateExport,
  buildBulkExport,
  getExportFilename,
  getBulkExportFilename,
} from "../formula-template-export";
import type { FormulaTemplate, FormulaTemplateVersion } from "@/types/formula-template";

function makeTemplate(overrides: Partial<FormulaTemplate> = {}): FormulaTemplate {
  return {
    name: "My Template",
    description: "A test template",
    type: "FreightCharge",
    expression: "weight * 0.5",
    status: "Active",
    schemaId: "shipment",
    variableDefinitions: [],
    metadata: null,
    sourceTemplateId: null,
    sourceVersionNumber: null,
    ...overrides,
  } as FormulaTemplate;
}

function makeVersion(overrides: Partial<FormulaTemplateVersion> = {}): FormulaTemplateVersion {
  return {
    id: "v1",
    templateId: "t1",
    organizationId: "org1",
    businessUnitId: "bu1",
    versionNumber: 1,
    name: "My Template",
    description: "Version 1",
    type: "FreightCharge",
    expression: "weight * 0.5",
    status: "Active",
    schemaId: "shipment",
    variableDefinitions: [],
    metadata: null,
    changeMessage: "Initial version",
    changeSummary: null,
    tags: ["Stable"],
    createdById: "user1",
    createdAt: 1700000000,
    createdBy: null,
    ...overrides,
  } as FormulaTemplateVersion;
}

describe("slugify", () => {
  it("converts spaces to hyphens and lowercases", () => {
    expect(slugify("My Template")).toBe("my-template");
  });

  it("replaces special chars with hyphens", () => {
    expect(slugify("Test (v2) & More!")).toBe("test-v2-more");
  });

  it("trims leading and trailing hyphens", () => {
    expect(slugify("--hello--")).toBe("hello");
  });

  it("leaves already slugified string unchanged", () => {
    expect(slugify("my-template")).toBe("my-template");
  });

  it("collapses multiple special chars into single hyphen", () => {
    expect(slugify("a   b___c")).toBe("a-b-c");
  });
});

describe("buildTemplateExport", () => {
  it("builds export without versions", () => {
    const template = makeTemplate();
    const result = buildTemplateExport(template);
    expect(result.exportVersion).toBe("1.0");
    expect(result.exportedAt).toBeDefined();
    expect(result.template.name).toBe("My Template");
    expect(result.versionHistory).toBeUndefined();
  });

  it("includes version history when provided", () => {
    const template = makeTemplate();
    const versions = [makeVersion({ versionNumber: 1 }), makeVersion({ versionNumber: 2 })];
    const result = buildTemplateExport(template, versions);
    expect(result.versionHistory).toHaveLength(2);
    expect(result.versionHistory![0].versionNumber).toBe(1);
  });

  it("omits versionHistory for empty versions array", () => {
    const template = makeTemplate();
    const result = buildTemplateExport(template, []);
    expect(result.versionHistory).toBeUndefined();
  });
});

describe("buildBulkExport", () => {
  it("maps all templates", () => {
    const templates = [makeTemplate({ name: "A" }), makeTemplate({ name: "B" })];
    const result = buildBulkExport(templates);
    expect(result.templates).toHaveLength(2);
    expect(result.templates[0].name).toBe("A");
    expect(result.templates[1].name).toBe("B");
  });

  it("has exportVersion and exportedAt", () => {
    const result = buildBulkExport([makeTemplate()]);
    expect(result.exportVersion).toBe("1.0");
    expect(result.exportedAt).toBeDefined();
  });
});

describe("getExportFilename", () => {
  it("generates filename without versions", () => {
    const template = makeTemplate({ name: "My Template" });
    expect(getExportFilename(template, false)).toBe("my-template.formula-template.json");
  });

  it("generates filename with versions", () => {
    const template = makeTemplate({ name: "My Template" });
    expect(getExportFilename(template, true)).toBe("my-template.formula-template-full.json");
  });

  it("slugifies the name", () => {
    const template = makeTemplate({ name: "Test & Debug" });
    expect(getExportFilename(template, false)).toBe("test-debug.formula-template.json");
  });
});

describe("getBulkExportFilename", () => {
  it("matches expected pattern", () => {
    const filename = getBulkExportFilename();
    expect(filename).toMatch(/^formula-templates-export-\d{4}-\d{2}-\d{2}\.json$/);
  });

  it("contains today's date", () => {
    const today = new Date().toISOString().split("T")[0];
    expect(getBulkExportFilename()).toContain(today);
  });
});
