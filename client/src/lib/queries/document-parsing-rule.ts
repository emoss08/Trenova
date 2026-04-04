import { apiService } from "@/services/api";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const documentParsingRule = createQueryKeys("documentParsingRule", {
  list: (documentKind?: string) => ({
    queryKey: ["list", documentKind],
    queryFn: async () => apiService.documentParsingRuleService.list(documentKind),
  }),
  detail: (id: string) => ({
    queryKey: ["detail", id],
    queryFn: async () => apiService.documentParsingRuleService.get(id),
  }),
  versions: (ruleSetId: string) => ({
    queryKey: ["versions", ruleSetId],
    queryFn: async () =>
      apiService.documentParsingRuleService.listVersions(ruleSetId),
  }),
  version: (versionId: string) => ({
    queryKey: ["version", versionId],
    queryFn: async () =>
      apiService.documentParsingRuleService.getVersion(versionId),
  }),
  fixtures: (ruleSetId: string) => ({
    queryKey: ["fixtures", ruleSetId],
    queryFn: async () =>
      apiService.documentParsingRuleService.listFixtures(ruleSetId),
  }),
  fixture: (fixtureId: string) => ({
    queryKey: ["fixture", fixtureId],
    queryFn: async () =>
      apiService.documentParsingRuleService.getFixture(fixtureId),
  }),
});
