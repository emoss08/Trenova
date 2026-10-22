import type {
  FormulaImport,
  FormulaTemplate,
  FormulaTemplateVersion,
} from "@trenova/shared/types/formula-template";
import { downloadJsonFile, slugify } from "@trenova/shared/lib/utils";

/** The export file's format: a template is its Tariff source, tests included. */
export const FORMULA_TEMPLATE_EXPORT_VERSION = "2.0";

export type FormulaTemplateExportPayload = {
  name: string;
  description: string;
  /** The tariff, with its inputs, charges, total rules and tests. */
  source: string;
  status: FormulaTemplate["status"];
  /** The libraries the exported version pinned; an import re-pins against the importing org. */
  imports: FormulaImport[];
  sourceTemplateId: string | null | undefined;
  sourceVersionNumber: number | null | undefined;
};

export type FormulaTemplateExport = {
  exportVersion: typeof FORMULA_TEMPLATE_EXPORT_VERSION;
  exportedAt: string;
  template: FormulaTemplateExportPayload;
  versionHistory?: Array<{
    versionNumber: number;
    name: string;
    description?: string;
    source: string;
    status: FormulaTemplate["status"];
    imports: FormulaImport[];
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

export function downloadJson(data: unknown, filename: string): void {
  downloadJsonFile(filename, data);
}

function toExportPayload(template: FormulaTemplate): FormulaTemplateExportPayload {
  return {
    name: template.name,
    description: template.description,
    source: template.source,
    status: template.status,
    imports: template.imports,
    sourceTemplateId: template.sourceTemplateId,
    sourceVersionNumber: template.sourceVersionNumber,
  };
}

export type TemplateExportOptions = { versions?: FormulaTemplateVersion[] };

/** A template's export: its source, which carries its tests, and optionally its history. */
export function buildTemplateExport(
  template: FormulaTemplate,
  options?: TemplateExportOptions,
): FormulaTemplateExport {
  const versions = options?.versions;
  const exportData: FormulaTemplateExport = {
    exportVersion: FORMULA_TEMPLATE_EXPORT_VERSION,
    exportedAt: new Date().toISOString(),
    template: toExportPayload(template),
  };

  if (versions && versions.length > 0) {
    exportData.versionHistory = versions.map((v) => ({
      versionNumber: v.versionNumber,
      name: v.name,
      description: v.description,
      source: v.source,
      status: v.status,
      imports: v.imports,
      changeMessage: v.changeMessage,
      tags: v.tags,
      createdAt: v.createdAt,
    }));
  }

  return exportData;
}

/** One version as a template of its own, forked from that version. */
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
      source: version.source,
      status: version.status,
      imports: version.imports,
      sourceTemplateId: undefined,
      sourceVersionNumber: version.versionNumber,
    },
  };
}

export function buildBulkExport(templates: FormulaTemplate[]): BulkFormulaTemplateExport {
  return {
    exportVersion: FORMULA_TEMPLATE_EXPORT_VERSION,
    exportedAt: new Date().toISOString(),
    templates: templates.map(toExportPayload),
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
