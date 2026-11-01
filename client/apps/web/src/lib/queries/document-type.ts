import { fetchGraphQLSelectOptions } from "@/lib/graphql/select-options";
import { createQueryKeys } from "@lukemorales/query-key-factory";

type QueryContext = { signal?: AbortSignal };

/** An organization's document types, as the screens that name them by id read them. */
export const documentType = createQueryKeys("document-type", {
  options: () => ({
    queryKey: ["options"],
    queryFn: ({ signal }: QueryContext) =>
      fetchGraphQLSelectOptions({ resource: "DOCUMENT_TYPE", initialLimit: 100 }, { signal }),
  }),
});
