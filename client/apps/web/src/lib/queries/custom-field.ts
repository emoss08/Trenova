import {
  CustomFieldFormFieldsDocument,
  CustomFieldResourceTypesDocument,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { createQueryKeys } from "@lukemorales/query-key-factory";

type QueryContext = { signal?: AbortSignal };

export const customField = createQueryKeys("custom-field", {
  /** The active custom fields a record's form shows, in display order. */
  formFields: (resourceType: string) => ({
    queryKey: ["form-fields", resourceType],
    queryFn: async ({ signal }: QueryContext) =>
      (
        await requestGraphQL({
          document: CustomFieldFormFieldsDocument,
          operationName: "CustomFieldFormFields",
          variables: { resourceType },
          signal,
        })
      ).customFieldFormFields,
  }),
  /** The resource types custom fields can be defined on. */
  resourceTypes: () => ({
    queryKey: ["resource-types"],
    queryFn: async ({ signal }: QueryContext) =>
      (
        await requestGraphQL({
          document: CustomFieldResourceTypesDocument,
          operationName: "CustomFieldResourceTypes",
          signal,
        })
      ).customFieldResourceTypes,
  }),
});
