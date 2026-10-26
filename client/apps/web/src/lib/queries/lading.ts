import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const lading = createQueryKeys("lading", {
  context: () => ({
    queryKey: ["context"],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      apiService.ladingService.getContext({ signal }),
  }),
  partnerProject: (partnerId: string) => ({
    queryKey: ["partner-project", partnerId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      apiService.ladingService.getPartnerProject(partnerId, { signal }),
  }),
  versions: (projectId: string) => ({
    queryKey: ["versions", projectId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      apiService.ladingService.listVersions(projectId, { signal }),
  }),
  version: (versionId: string) => ({
    queryKey: ["version", versionId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      apiService.ladingService.getVersion(versionId, { signal }),
  }),
  replays: (versionId: string) => ({
    queryKey: ["replays", versionId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      apiService.ladingService.listReplays(versionId, { signal }),
  }),
  shadowResults: (versionId: string, onlyDiffering: boolean) => ({
    queryKey: ["shadow-results", versionId, onlyDiffering],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      apiService.ladingService.shadowResults(versionId, onlyDiffering, { signal }),
  }),
  flowInstance: (instanceId: string) => ({
    queryKey: ["flow-instance", instanceId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      apiService.ladingService.getFlowInstance(instanceId, { signal }),
  }),
});
