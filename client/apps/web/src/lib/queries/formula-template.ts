import { apiService } from "@/services/api";
import type { FormulaTemplate } from "@trenova/shared/types/formula-template";
import { createQueryKeys } from "@lukemorales/query-key-factory";
import type { QueryClient } from "@tanstack/react-query";

export const formulaTemplate = createQueryKeys("formulaTemplate", {
  get: (templateId: FormulaTemplate["id"]) => ({
    queryKey: [templateId],
    queryFn: async () => apiService.formulaTemplateService.get(templateId),
  }),
  usage: (templateId: FormulaTemplate["id"]) => ({
    queryKey: [templateId],
    queryFn: async () => apiService.formulaTemplateService.getUsage(templateId),
  }),
  schema: (schemaId: string) => ({
    queryKey: [schemaId],
    queryFn: async () => apiService.formulaTemplateService.getSchema(schemaId),
  }),
  versionDiff: (templateId: FormulaTemplate["id"], fromVersion: number, toVersion: number) => ({
    queryKey: [templateId, fromVersion, toVersion],
    queryFn: async () =>
      apiService.formulaTemplateService.compareVersions(templateId, fromVersion, toVersion),
  }),
  testCases: (templateId: FormulaTemplate["id"]) => ({
    queryKey: [templateId],
    queryFn: async () => apiService.formulaTemplateService.listTestCases(templateId),
  }),
  versions: (templateId: FormulaTemplate["id"], limit?: number, offset?: number) => ({
    queryKey: [templateId, limit, offset],
    queryFn: async () =>
      apiService.formulaTemplateService.listVersions(templateId, {
        limit,
        offset,
      }),
  }),
  lineage: (templateId: FormulaTemplate["id"]) => ({
    queryKey: [templateId],
    queryFn: async () => {
      return apiService.formulaTemplateService.getLineage(templateId);
    },
  }),
  scheduledVersions: (templateId: FormulaTemplate["id"]) => ({
    queryKey: [templateId],
    queryFn: async () => apiService.formulaTemplateService.listScheduledVersions(templateId),
  }),
  approvalImpact: (templateId: FormulaTemplate["id"]) => ({
    queryKey: [templateId],
    queryFn: async () => apiService.formulaTemplateService.approvalImpact(templateId),
  }),
});

export async function invalidateFormulaTemplate(queryClient: QueryClient): Promise<void> {
  await Promise.all([
    queryClient.invalidateQueries({
      queryKey: ["formula-template-list"],
      refetchType: "all",
    }),
    queryClient.invalidateQueries({ queryKey: formulaTemplate._def }),
  ]);
}
