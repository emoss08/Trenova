import { apiService } from "@/services/api";
import type {
  FormulaTemplate,
  TestExpressionRequest,
} from "@trenova/shared/types/formula-template";
import { createQueryKeys } from "@lukemorales/query-key-factory";
import type { QueryClient } from "@tanstack/react-query";

export const formulaTemplate = createQueryKeys("formulaTemplate", {
  get: (templateId: FormulaTemplate["id"]) => ({
    queryKey: [templateId],
    queryFn: async () => apiService.formulaTemplateService.get(templateId),
  }),
  recent: (limit: number) => ({
    queryKey: [limit],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      apiService.formulaTemplateService.list({ limit }, { signal }),
  }),
  /** The inputs a template version declares; no version is the one that bills today. */
  inputs: (templateId: string, versionNumber: number | null) => ({
    queryKey: [templateId, versionNumber ?? "billing"],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      apiService.formulaTemplateService.getDeclaredInputs(templateId, versionNumber, { signal }),
  }),
  usage: (templateId: FormulaTemplate["id"]) => ({
    queryKey: [templateId],
    queryFn: async () => apiService.formulaTemplateService.getUsage(templateId),
  }),
  price: (request: TestExpressionRequest) => ({
    queryKey: [request],
    queryFn: async () => apiService.formulaTemplateService.test(request),
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
  reviewers: (templateId: FormulaTemplate["id"]) => ({
    queryKey: [templateId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      apiService.formulaTemplateService.listReviewers(templateId, { signal }),
  }),
  readiness: (templateId: FormulaTemplate["id"]) => ({
    queryKey: [templateId],
    queryFn: async () => apiService.formulaTemplateService.readiness(templateId),
  }),
  standards: {
    queryKey: null,
    queryFn: async () => apiService.formulaTemplateService.listStandards(),
  },
  reviews: (templateId: FormulaTemplate["id"]) => ({
    queryKey: [templateId],
    queryFn: async () => apiService.formulaTemplateService.listReviews(templateId),
  }),
  reviewDiff: (templateId: FormulaTemplate["id"]) => ({
    queryKey: [templateId],
    queryFn: async () => apiService.formulaTemplateService.reviewDiff(templateId),
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
