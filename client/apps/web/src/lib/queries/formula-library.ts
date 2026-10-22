import { fetchFormulaLibrary } from "@/lib/graphql/formula-library";
import { formulaTemplate } from "@/lib/queries/formula-template";
import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";
import type { QueryClient } from "@tanstack/react-query";

type Signal = { signal?: AbortSignal };

export const formulaLibrary = createQueryKeys("formulaLibrary", {
  /** A library with its versions and review rounds (GraphQL `formulaLibrary`). */
  get: (libraryId: string) => ({
    queryKey: [libraryId],
    queryFn: async ({ signal }: Signal) => fetchFormulaLibrary(libraryId, { signal }),
  }),
  version: (libraryId: string, versionNumber: number) => ({
    queryKey: [libraryId, versionNumber],
    queryFn: async ({ signal }: Signal) =>
      apiService.formulaLibraryService.getVersion(libraryId, versionNumber, { signal }),
  }),
  versionDiff: (libraryId: string, fromVersion: number, toVersion: number) => ({
    queryKey: [libraryId, fromVersion, toVersion],
    queryFn: async ({ signal }: Signal) =>
      apiService.formulaLibraryService.compareVersions(libraryId, fromVersion, toVersion, {
        signal,
      }),
  }),
  readiness: (libraryId: string) => ({
    queryKey: [libraryId],
    queryFn: async ({ signal }: Signal) =>
      apiService.formulaLibraryService.readiness(libraryId, { signal }),
  }),
  reviewDiff: (libraryId: string) => ({
    queryKey: [libraryId],
    queryFn: async ({ signal }: Signal) =>
      apiService.formulaLibraryService.reviewDiff(libraryId, { signal }),
  }),
  reviewers: (libraryId: string) => ({
    queryKey: [libraryId],
    queryFn: async ({ signal }: Signal) =>
      apiService.formulaLibraryService.listReviewers(libraryId, { signal }),
  }),
  usage: (libraryId: string) => ({
    queryKey: [libraryId],
    queryFn: async ({ signal }: Signal) =>
      apiService.formulaLibraryService.usage(libraryId, { signal }),
  }),
});

/** Refreshes every library read, and the template context whose library sources may change. */
export async function invalidateFormulaLibrary(queryClient: QueryClient): Promise<void> {
  await Promise.all([
    queryClient.invalidateQueries({ queryKey: ["formula-library-list"], refetchType: "all" }),
    queryClient.invalidateQueries({ queryKey: formulaLibrary._def }),
    queryClient.invalidateQueries({ queryKey: formulaTemplate.context.queryKey }),
  ]);
}
