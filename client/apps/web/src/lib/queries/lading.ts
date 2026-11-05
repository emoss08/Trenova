import { ladingService } from "@/services/edi/lading";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const lading = createQueryKeys("lading", {
  context: () => ({
    queryKey: ["context"],
    queryFn: async ({ signal }: { signal?: AbortSignal }) => ladingService.getContext({ signal }),
  }),
  partnerProject: (partnerId: string) => ({
    queryKey: ["partner-project", partnerId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      ladingService.getPartnerProject(partnerId, { signal }),
  }),
  versions: (projectId: string) => ({
    queryKey: ["versions", projectId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      ladingService.listVersions(projectId, { signal }),
  }),
  version: (versionId: string) => ({
    queryKey: ["version", versionId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      ladingService.getVersion(versionId, { signal }),
  }),
  replays: (versionId: string) => ({
    queryKey: ["replays", versionId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      ladingService.listReplays(versionId, { signal }),
  }),
  shadowResults: (versionId: string, onlyDiffering: boolean) => ({
    queryKey: ["shadow-results", versionId, onlyDiffering],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      ladingService.shadowResults(versionId, onlyDiffering, { signal }),
  }),
  flowInstance: (instanceId: string) => ({
    queryKey: ["flow-instance", instanceId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      ladingService.getFlowInstance(instanceId, { signal }),
  }),
});
