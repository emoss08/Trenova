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
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      apiService.formulaTemplateService.getUsage(templateId, { signal }),
  }),
  /** The preview's rating of a source with sample inputs or a real shipment. */
  preview: (request: FormulaPreviewRequest) => ({
    queryKey: [request],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      apiService.formulaTemplateService.preview(request, { signal }),
  }),
  /** What the browser checks a tariff against (the `formulaTemplateStudioContext` query). */
  context: {
    queryKey: null,
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      apiService.formulaTemplateService.getContext({ signal }),
  },
  /**
   * The rate table cells the browser's tests have loaded (the `formulaTemplateTableCells` query),
   * kept per rate table stamp; written by `useTariffTableCells`, never fetched.
   */
  contextCells: (stamp: string) => ({ queryKey: [stamp] }),
  versionDiff: (templateId: string, fromVersion: number, toVersion: number) => ({
    queryKey: [templateId, fromVersion, toVersion],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      apiService.formulaTemplateService.compareVersions(templateId, fromVersion, toVersion, {
        signal,
      }),
  }),
  versions: (templateId: string, limit?: number, offset?: number) => ({
    queryKey: [templateId, limit, offset],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      apiService.formulaTemplateService.listVersions(templateId, { limit, offset }, { signal }),
  }),
  lineage: (templateId: string) => ({
    queryKey: [templateId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      apiService.formulaTemplateService.getLineage(templateId, { signal }),
  }),
  scheduledVersions: (templateId: string) => ({
    queryKey: [templateId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      apiService.formulaTemplateService.listScheduledVersions(templateId, { signal }),
  }),
  approvalImpact: (templateId: string) => ({
    queryKey: [templateId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      apiService.formulaTemplateService.approvalImpact(templateId, undefined, { signal }),
  }),
  reviewers: (templateId: string) => ({
    queryKey: [templateId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      apiService.formulaTemplateService.listReviewers(templateId, { signal }),
  }),
  readiness: (templateId: string) => ({
    queryKey: [templateId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      apiService.formulaTemplateService.readiness(templateId, { signal }),
  }),
  standards: {
    queryKey: null,
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      apiService.formulaTemplateService.listStandards({ signal }),
  },
  reviews: (templateId: string) => ({
    queryKey: [templateId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      apiService.formulaTemplateService.listReviews(templateId, { signal }),
  }),
  reviewDiff: (templateId: string) => ({
    queryKey: [templateId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      apiService.formulaTemplateService.reviewDiff(templateId, { signal }),
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
