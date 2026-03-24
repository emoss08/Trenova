import { apiService } from "@/services/api";
import type { FormulaTemplate } from "@/types/formula-template";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const formulaTemplate = createQueryKeys("formulaTemplate", {
  versions: (
    templateId: FormulaTemplate["id"],
    limit?: number,
    offset?: number,
  ) => ({
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
});
