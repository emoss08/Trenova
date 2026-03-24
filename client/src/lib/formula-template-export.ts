import type {
  FormulaTemplate,
  FormulaTemplateVersion,
} from "@/types/formula-template";

export type FormulaTemplateExport = {
  exportVersion: "1.0";
  exportedAt: string;
  template: {
    name: string;
    description: string;
    type: FormulaTemplate["type"];
    expression: string;
    status: FormulaTemplate["status"];
    schemaId: string;
    variableDefinitions: FormulaTemplate["variableDefinitions"];
    metadata: FormulaTemplate["metadata"];
    sourceTemplateId: string | null | undefined;
    sourceVersionNumber: number | null | undefined;
  };
  versionHistory?: Array<{
    versionNumber: number;
    name: string;
    description?: string;
    type: FormulaTemplate["type"];
    expression: string;
    status: FormulaTemplate["status"];
    schemaId: string;
    variableDefinitions: FormulaTemplate["variableDefinitions"];
    metadata: FormulaTemplate["metadata"];
    changeMessage?: string;
    tags: string[];
    createdAt: number;
  }>;
};

export type BulkFormulaTemplateExport = {
  exportVersion: "1.0";
  exportedAt: string;
  templates: Array<FormulaTemplateExport["template"]>;
};

export function slugify(name: string): string {
  return name
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-|-$/g, "");
}

export function downloadJson(data: unknown, filename: string): void {
  const json = JSON.stringify(data, null, 2);
  const blob = new Blob([json], { type: "application/json" });
  const url = URL.createObjectURL(blob);

  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
}

export function buildTemplateExport(
  template: FormulaTemplate,
  versions?: FormulaTemplateVersion[],
): FormulaTemplateExport {
  const exportData: FormulaTemplateExport = {
    exportVersion: "1.0",
    exportedAt: new Date().toISOString(),
    template: {
      name: template.name,
      description: template.description,
      type: template.type,
      expression: template.expression,
      status: template.status,
      schemaId: template.schemaId,
      variableDefinitions: template.variableDefinitions,
      metadata: template.metadata,
      sourceTemplateId: template.sourceTemplateId,
      sourceVersionNumber: template.sourceVersionNumber,
    },
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
      metadata: v.metadata,
      changeMessage: v.changeMessage,
      tags: v.tags,
      createdAt: v.createdAt,
    }));
  }

  return exportData;
}

export function buildBulkExport(
  templates: FormulaTemplate[],
): BulkFormulaTemplateExport {
  return {
    exportVersion: "1.0",
    exportedAt: new Date().toISOString(),
    templates: templates.map((template) => ({
      name: template.name,
      description: template.description,
      type: template.type,
      expression: template.expression,
      status: template.status,
      schemaId: template.schemaId,
      variableDefinitions: template.variableDefinitions,
      metadata: template.metadata,
      sourceTemplateId: template.sourceTemplateId,
      sourceVersionNumber: template.sourceVersionNumber,
    })),
  };
}

export function getExportFilename(
  template: FormulaTemplate,
  includeVersions: boolean,
): string {
  const slug = slugify(template.name);
  return includeVersions
    ? `${slug}.formula-template-full.json`
    : `${slug}.formula-template.json`;
}

export function getBulkExportFilename(): string {
  const date = new Date().toISOString().split("T")[0];
  return `formula-templates-export-${date}.json`;
}
