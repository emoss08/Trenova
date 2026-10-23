import { fetchRateMatrixRecord, fetchRateMatrixVersionAxes } from "@/lib/graphql/rate-matrix";
import { formulaTemplate } from "@/lib/queries/formula-template";
import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";
import type { QueryClient } from "@tanstack/react-query";

type Signal = { signal?: AbortSignal };

export const rateMatrix = createQueryKeys("rateMatrix", {
  /** A matrix's draft state, published versions, review rounds and usages (GraphQL). */
  record: (rateMatrixId: string) => ({
    queryKey: [rateMatrixId],
    queryFn: async ({ signal }: Signal) => fetchRateMatrixRecord(rateMatrixId, { signal }),
  }),
  /** Every cell of the draft (no version) or of one published version. */
  cells: (rateMatrixId: string, versionNumber: number | null) => ({
    queryKey: [rateMatrixId, versionNumber ?? "draft"],
    queryFn: async ({ signal }: Signal) =>
      apiService.rateMatrixService.listCells(rateMatrixId, versionNumber, { signal }),
  }),
  /** One published version's axes. */
  versionAxes: (rateMatrixId: string, versionNumber: number) => ({
    queryKey: [rateMatrixId, versionNumber],
    queryFn: async ({ signal }: Signal) =>
      fetchRateMatrixVersionAxes(rateMatrixId, versionNumber, { signal }),
  }),
  /** Who can publish the organisation's matrices. */
  reviewers: (rateMatrixId: string) => ({
    queryKey: [rateMatrixId],
    queryFn: async ({ signal }: Signal) =>
      apiService.rateMatrixService.listReviewers(rateMatrixId, { signal }),
  }),
});

/**
 * Everything a matrix write can change: the list, the record and its cells, and the Studio's
 * context and every template's pinned tables, which read the published versions.
 */
export async function invalidateRateMatrix(queryClient: QueryClient): Promise<void> {
  await Promise.all([
    queryClient.invalidateQueries({ queryKey: ["rate-matrix-list"], refetchType: "all" }),
    queryClient.invalidateQueries({ queryKey: rateMatrix._def }),
    queryClient.invalidateQueries({ queryKey: formulaTemplate.context.queryKey }),
    // Named, not imported: the factory module would bring its query into every matrix screen.
    queryClient.invalidateQueries({ queryKey: ["formulaTemplateTables"] }),
  ]);
}
