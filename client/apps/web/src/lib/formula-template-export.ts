import type {
  FormulaTemplate,
  FormulaTemplateVersion,
  FormulaTestCase,
} from "@trenova/shared/types/formula-template";
import { downloadJsonFile } from "@trenova/shared/lib/utils";

export const FORMULA_TEMPLATE_EXPORT_VERSION = "1.2";

export type ExportedTestCase = {
  name: string;
  description: string;
  variables: Record<string, unknown>;
  expectedAmount: number;
  tolerance: number;
};

export type FormulaTemplateExportPayload = {
  name: string;
  description: string;
  type: FormulaTemplate["type"];
  expression: string;
  status: FormulaTemplate["status"];
  schemaId: string;
  variableDefinitions: FormulaTemplate["variableDefinitions"];
  breakdownDefinitions: FormulaTemplate["breakdownDefinitions"];
  minCharge: FormulaTemplate["minCharge"];
  maxCharge: FormulaTemplate["maxCharge"];
  metadata: FormulaTemplate["metadata"];
  sourceTemplateId: string | null | undefined;
  sourceVersionNumber: number | null | undefined;
  testCases?: ExportedTestCase[];
};

export type FormulaTemplateExport = {
  exportVersion: typeof FORMULA_TEMPLATE_EXPORT_VERSION;
  exportedAt: string;
  template: FormulaTemplateExportPayload;
  versionHistory?: Array<{
    versionNumber: number;
    name: string;
    description?: string;
    type: FormulaTemplate["type"];
    expression: string;
    status: FormulaTemplate["status"];
    schemaId: string;
    variableDefinitions: FormulaTemplate["variableDefinitions"];
    breakdownDefinitions: FormulaTemplate["breakdownDefinitions"];
    minCharge: FormulaTemplate["minCharge"];
    maxCharge: FormulaTemplate["maxCharge"];
    metadata: FormulaTemplate["metadata"];
    changeMessage?: string;
    tags: string[];
    createdAt: number;
  }>;
};

export type BulkFormulaTemplateExport = {
  exportVersion: typeof FORMULA_TEMPLATE_EXPORT_VERSION;
  exportedAt: string;
  templates: Array<FormulaTemplateExportPayload>;
};

export function slugify(name: string): string {
  return name
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-|-$/g, "");
}

export function downloadJson(data: unknown, filename: string): void {
  downloadJsonFile(filename, data);
}

export function toExportedTestCase(testCase: FormulaTestCase): ExportedTestCase {
  return {
    name: testCase.name,
    description: testCase.description,
    variables: testCase.variables,
    expectedAmount: testCase.expectedAmount,
    tolerance: testCase.tolerance,
  };
}

function toExportPayload(
  template: FormulaTemplate,
  testCases?: FormulaTestCase[],
): FormulaTemplateExportPayload {
  const payload: FormulaTemplateExportPayload = {
    name: template.name,
    description: template.description,
    type: template.type,
    expression: template.expression,
    status: template.status,
    schemaId: template.schemaId,
    variableDefinitions: template.variableDefinitions,
    breakdownDefinitions: template.breakdownDefinitions,
    minCharge: template.minCharge,
    maxCharge: template.maxCharge,
    metadata: template.metadata,
    sourceTemplateId: template.sourceTemplateId,
    sourceVersionNumber: template.sourceVersionNumber,
  };

  if (testCases && testCases.length > 0) {
    payload.testCases = testCases.map(toExportedTestCase);
  }

  return payload;
}

export type TemplateExportOptions = {
  versions?: FormulaTemplateVersion[];
  testCases?: FormulaTestCase[];
};

export function buildTemplateExport(
  template: FormulaTemplate,
  options?: TemplateExportOptions,
): FormulaTemplateExport {
  const { versions, testCases } = options ?? {};
  const exportData: FormulaTemplateExport = {
    exportVersion: FORMULA_TEMPLATE_EXPORT_VERSION,
    exportedAt: new Date().toISOString(),
    template: toExportPayload(template, testCases),
  };

  if (versions && versions.length > 0) {
    exportData.versionHistory = versions.map((v) => ({
      versionNumber: v.versionNumber,
      name: v.name,
      description: v.description,
      type: v.type,
      expression: v.expression,
      status: v.status,
      schemaId: v.schemaId,
      variableDefinitions: v.variableDefinitions,
      breakdownDefinitions: v.breakdownDefinitions,
      minCharge: v.minCharge,
      maxCharge: v.maxCharge,
      metadata: v.metadata,
      changeMessage: v.changeMessage,
      tags: v.tags,
      createdAt: v.createdAt,
    }));
  }

  return exportData;
}

// A version export deliberately carries no test scenarios: scenarios live on
// the template and pin its current behaviour, which an older version's
// expression has no obligation to satisfy.
export function buildVersionExport(
  template: Pick<FormulaTemplate, "name">,
  version: FormulaTemplateVersion,
): FormulaTemplateExport {
  return {
    exportVersion: FORMULA_TEMPLATE_EXPORT_VERSION,
    exportedAt: new Date().toISOString(),
    template: {
      name: template.name,
      description: version.description ?? "",
      type: version.type,
      expression: version.expression,
      status: version.status,
      schemaId: version.schemaId,
      variableDefinitions: version.variableDefinitions,
      breakdownDefinitions: version.breakdownDefinitions,
      minCharge: version.minCharge,
      maxCharge: version.maxCharge,
      metadata: version.metadata,
      sourceTemplateId: undefined,
      sourceVersionNumber: version.versionNumber,
    },
  };
}

export function buildBulkExport(
  templates: FormulaTemplate[],
  testCasesByTemplateId?: Record<string, FormulaTestCase[]>,
): BulkFormulaTemplateExport {
  return {
    exportVersion: FORMULA_TEMPLATE_EXPORT_VERSION,
    exportedAt: new Date().toISOString(),
    templates: templates.map((template) =>
      toExportPayload(template, template.id ? testCasesByTemplateId?.[template.id] : undefined),
    ),
  };
}

export function getExportFilename(template: FormulaTemplate, includeVersions: boolean): string {
  const slug = slugify(template.name);
  return includeVersions ? `${slug}.formula-template-full.json` : `${slug}.formula-template.json`;
}

export function getVersionExportFilename(
  template: Pick<FormulaTemplate, "name">,
  versionNumber: number,
): string {
  return `${slugify(template.name)}.v${versionNumber}.formula-template.json`;
}

export function getBulkExportFilename(): string {
  const date = new Date().toISOString().split("T")[0];
  return `formula-templates-export-${date}.json`;
}
