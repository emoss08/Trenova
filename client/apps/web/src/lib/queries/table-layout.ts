import { MyTableLayoutDocument } from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const tableLayout = createQueryKeys("tableLayout", {
  mine: (resource: string) => ({
    queryKey: [resource],
    queryFn: async ({ signal }) =>
      requestGraphQL({
        document: MyTableLayoutDocument,
        operationName: "MyTableLayout",
        variables: { resource },
        signal,
      }),
  }),
});
