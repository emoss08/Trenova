import { apiService } from "@/services/api";
import type { FormulaPreviewRequest } from "@trenova/shared/types/formula-template";
import { createQueryKeys } from "@lukemorales/query-key-factory";
import type { QueryClient } from "@tanstack/react-query";

export const formulaTemplate = createQueryKeys("formulaTemplate", {
  get: (templateId: string) => ({
    queryKey: [templateId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      apiService.formulaTemplateService.get(templateId, { signal }),
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
  usage: (templateId: string) => ({
    queryKey: [templateId],
    queryFn: async () => apiService.formulaTemplateService.getUsage(templateId),
  }),
  /** The preview's rating of a source with sample inputs or a real shipment. */
  preview: (request: FormulaPreviewRequest) => ({
    queryKey: [request],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      apiService.formulaTemplateService.preview(request, { signal }),
  }),
  /** What the browser checks a tariff against (GET /formula-templates/context). */
  context: {
    queryKey: null,
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      apiService.formulaTemplateService.getContext({ signal }),
  },
  versionDiff: (templateId: string, fromVersion: number, toVersion: number) => ({
    queryKey: [templateId, fromVersion, toVersion],
    queryFn: async () =>
      apiService.formulaTemplateService.compareVersions(templateId, fromVersion, toVersion),
  }),
  versions: (templateId: string, limit?: number, offset?: number) => ({
    queryKey: [templateId, limit, offset],
    queryFn: async () =>
      apiService.formulaTemplateService.listVersions(templateId, {
        limit,
        offset,
      }),
  }),
  lineage: (templateId: string) => ({
    queryKey: [templateId],
    queryFn: async () => {
      return apiService.formulaTemplateService.getLineage(templateId);
    },
  }),
  scheduledVersions: (templateId: string) => ({
    queryKey: [templateId],
    queryFn: async () => apiService.formulaTemplateService.listScheduledVersions(templateId),
  }),
  approvalImpact: (templateId: string) => ({
    queryKey: [templateId],
    queryFn: async () => apiService.formulaTemplateService.approvalImpact(templateId),
  }),
  reviewers: (templateId: string) => ({
    queryKey: [templateId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      apiService.formulaTemplateService.listReviewers(templateId, { signal }),
  }),
  readiness: (templateId: string) => ({
    queryKey: [templateId],
    queryFn: async () => apiService.formulaTemplateService.readiness(templateId),
  }),
  standards: {
    queryKey: null,
    queryFn: async () => apiService.formulaTemplateService.listStandards(),
  },
  reviews: (templateId: string) => ({
    queryKey: [templateId],
    queryFn: async () => apiService.formulaTemplateService.listReviews(templateId),
  }),
  reviewDiff: (templateId: string) => ({
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
