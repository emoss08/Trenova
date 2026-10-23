import { fetchFormulaTemplateTables } from "@/lib/graphql/formula-template-tables";
import { createQueryKeys } from "@lukemorales/query-key-factory";

/**
 * The rate tables a formula template reads, each with its pinned and newest published version.
 * Its own scope, so a page that reads templates or matrices brings only the query it runs; a
 * matrix write or a published version invalidates it.
 */
export const formulaTemplateTables = createQueryKeys("formulaTemplateTables", {
  get: (templateId: string) => ({
    queryKey: [templateId],
    queryFn: async ({ signal }: { signal?: AbortSignal }) =>
      fetchFormulaTemplateTables(templateId, { signal }),
  }),
});
