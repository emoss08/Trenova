import { describe, expect, it } from "vitest";
import { slugify } from "@trenova/shared/lib/utils";
import {
  FORMULA_TEMPLATE_EXPORT_VERSION,
  buildBulkExport,
  buildTemplateExport,
  buildVersionExport,
  getBulkExportFilename,
  getExportFilename,
  getVersionExportFilename,
} from "../formula-template-export";
import {
  formulaTemplateSchema,
  formulaTemplateVersionSchema,
  type FormulaTemplate,
  type FormulaTemplateVersion,
} from "@trenova/shared/types/formula-template";

const SOURCE = `tariff "My Template"

input rate: Money/mi = $2.35/mi

charge "Linehaul" (quantity: shipment.distance, rate: rate)

test "bills the miles" {
  given shipment.distance = 100 mi
  expect total == $235
}
`;

function makeTemplate(overrides: Partial<FormulaTemplate> = {}): FormulaTemplate {
  return {
    ...formulaTemplateSchema.parse({
      id: "fmt_1",
      name: "My Template",
      description: "A test template",
      source: SOURCE,
      status: "Active",
      imports: [{ path: "org.accessorials", libraryId: "fl_1", versionNumber: 3 }],
      testCount: 1,
      sourceTemplateId: null,
      sourceVersionNumber: null,
    }),
    ...overrides,
  };
}

function makeVersion(overrides: Partial<FormulaTemplateVersion> = {}): FormulaTemplateVersion {
  return {
    ...formulaTemplateVersionSchema.parse({
      id: "v1",
      templateId: "fmt_1",
      organizationId: "org1",
      businessUnitId: "bu1",
      versionNumber: 1,
      name: "My Template",
      description: "Version 1",
      source: SOURCE,
      status: "Active",
      imports: [],
      changeMessage: "Initial version",
      tags: ["Stable"],
      createdById: "usr_1",
      createdAt: 1_700_000_000,
    }),
    ...overrides,
  };
}

describe("slugify", () => {
  it("converts spaces to hyphens and lowercases", () => {
    expect(slugify("My Template Name")).toBe("my-template-name");
  });

  it("trims and collapses special characters", () => {
    expect(slugify("--rate @ 2.35/mi!--")).toBe("rate-2-35-mi");
  });
});

describe("buildTemplateExport", () => {
  it("exports the source, which carries the tests, and the pinned imports", () => {
    const exported = buildTemplateExport(makeTemplate());

    expect(exported.exportVersion).toBe(FORMULA_TEMPLATE_EXPORT_VERSION);
    expect(exported.template).toEqual({
      name: "My Template",
      description: "A test template",
      source: SOURCE,
      status: "Active",
      imports: [{ path: "org.accessorials", libraryId: "fl_1", versionNumber: 3 }],
      sourceTemplateId: null,
      sourceVersionNumber: null,
    });
    expect(exported.versionHistory).toBeUndefined();
  });

  it("includes the version history when given, each version with its own source", () => {
    const exported = buildTemplateExport(makeTemplate(), {
      versions: [makeVersion(), makeVersion({ versionNumber: 2, source: `${SOURCE}// v2\n` })],
    });

    expect(
      exported.versionHistory?.map((version) => [version.versionNumber, version.source]),
    ).toEqual([
      [1, SOURCE],
      [2, `${SOURCE}// v2\n`],
    ]);
    expect(exported.versionHistory?.[0]).toMatchObject({ tags: ["Stable"], imports: [] });
  });

  it("omits the history for an empty list of versions", () => {
    expect(buildTemplateExport(makeTemplate(), { versions: [] }).versionHistory).toBeUndefined();
  });
});

describe("buildVersionExport", () => {
  it("exports one version as a template forked from that version", () => {
    const exported = buildVersionExport({ name: "My Template" }, makeVersion({ versionNumber: 4 }));
    expect(exported.template).toEqual({
      name: "My Template",
      description: "Version 1",
      source: SOURCE,
      status: "Active",
      imports: [],
      sourceTemplateId: undefined,
      sourceVersionNumber: 4,
    });
  });
});

describe("buildBulkExport", () => {
  it("exports every template's source", () => {
    const exported = buildBulkExport([makeTemplate(), makeTemplate({ name: "Second" })]);
    expect(exported.exportVersion).toBe(FORMULA_TEMPLATE_EXPORT_VERSION);
    expect(exported.templates.map((template) => [template.name, template.source])).toEqual([
      ["My Template", SOURCE],
      ["Second", SOURCE],
    ]);
  });
});

describe("file names", () => {
  it("names a template, its full history, a version and a bulk export", () => {
    expect(getExportFilename(makeTemplate(), false)).toBe("my-template.formula-template.json");
    expect(getExportFilename(makeTemplate(), true)).toBe("my-template.formula-template-full.json");
    expect(getVersionExportFilename({ name: "My Template" }, 3)).toBe(
      "my-template.v3.formula-template.json",
    );
    const today = new Date().toISOString().split("T")[0];
    expect(getBulkExportFilename()).toBe(`formula-templates-export-${today}.json`);
  });
});
