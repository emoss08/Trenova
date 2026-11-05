import { formulaTemplateService } from "@/services/formula-template";
import type { FormulaPreviewRequest } from "@trenova/shared/types/formula-template";
import { createQueryKeys } from "@lukemorales/query-key-factory";
import type { QueryClient } from "@tanstack/react-query";

export const formulaTemplate = createQueryKeys("formulaTemplate", {
  get: (templateId: string) => ({
    queryKey: [templateId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      formulaTemplateService.get(templateId, { signal }),
  }),
  recent: (limit: number) => ({
    queryKey: [limit],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      formulaTemplateService.list({ limit }, { signal }),
  }),
  /** The inputs a template version declares; no version is the one that bills today. */
  inputs: (templateId: string, versionNumber: number | null) => ({
    queryKey: [templateId, versionNumber ?? "billing"],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      formulaTemplateService.getDeclaredInputs(templateId, versionNumber, { signal }),
  }),
  usage: (templateId: string) => ({
    queryKey: [templateId],
    queryFn: async () => formulaTemplateService.getUsage(templateId),
  }),
  /** The preview's rating of a source with sample inputs or a real shipment. */
  preview: (request: FormulaPreviewRequest) => ({
    queryKey: [request],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      formulaTemplateService.preview(request, { signal }),
  }),
  /** What the browser checks a tariff against (GET /formula-templates/context). */
  context: {
    queryKey: null,
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      formulaTemplateService.getContext({ signal }),
  },
  /**
   * The rate table cells the browser's tests have loaded (POST /formula-templates/context/cells),
   * kept per rate table stamp; written by `useTariffTableCells`, never fetched.
   */
  contextCells: (stamp: string) => ({ queryKey: [stamp] }),
  versionDiff: (templateId: string, fromVersion: number, toVersion: number) => ({
    queryKey: [templateId, fromVersion, toVersion],
    queryFn: async () => formulaTemplateService.compareVersions(templateId, fromVersion, toVersion),
  }),
  versions: (templateId: string, limit?: number, offset?: number) => ({
    queryKey: [templateId, limit, offset],
    queryFn: async () =>
      formulaTemplateService.listVersions(templateId, {
        limit,
        offset,
      }),
  }),
  lineage: (templateId: string) => ({
    queryKey: [templateId],
    queryFn: async () => {
      return formulaTemplateService.getLineage(templateId);
    },
  }),
  scheduledVersions: (templateId: string) => ({
    queryKey: [templateId],
    queryFn: async () => formulaTemplateService.listScheduledVersions(templateId),
  }),
  approvalImpact: (templateId: string) => ({
    queryKey: [templateId],
    queryFn: async () => formulaTemplateService.approvalImpact(templateId),
  }),
  reviewers: (templateId: string) => ({
    queryKey: [templateId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      formulaTemplateService.listReviewers(templateId, { signal }),
  }),
  readiness: (templateId: string) => ({
    queryKey: [templateId],
    queryFn: async () => formulaTemplateService.readiness(templateId),
  }),
  standards: {
    queryKey: null,
    queryFn: async () => formulaTemplateService.listStandards(),
  },
  reviews: (templateId: string) => ({
    queryKey: [templateId],
    queryFn: async () => formulaTemplateService.listReviews(templateId),
  }),
  reviewDiff: (templateId: string) => ({
    queryKey: [templateId],
    queryFn: async () => formulaTemplateService.reviewDiff(templateId),
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
