import {
  BulkEditFieldsDocument,
  BulkEditJobFieldsFragmentDoc,
  MyBulkEditDocument,
  PreviewBulkEditDocument,
  type BulkEditInput,
} from "@trenova/graphql/generated/graphql";
import { getFragmentData } from "@trenova/graphql/fragment-data";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { createQueryKeys } from "@lukemorales/query-key-factory";

export const bulkEdit = createQueryKeys("bulkEdit", {
  fields: (resource: string) => ({
    queryKey: [resource],
    queryFn: async ({ signal }) => {
      const data = await requestGraphQL({
        document: BulkEditFieldsDocument,
        operationName: "BulkEditFields",
        variables: { resource },
        signal,
      });
      return data.bulkEditFields;
    },
  }),
  preview: (input: BulkEditInput) => ({
    queryKey: [input],
    queryFn: async ({ signal }) => {
      const data = await requestGraphQL({
        document: PreviewBulkEditDocument,
        operationName: "PreviewBulkEdit",
        variables: { input },
        signal,
      });
      return data.previewBulkEdit;
    },
  }),
  job: (id: string) => ({
    queryKey: [id],
    queryFn: async ({ signal }) => {
      const data = await requestGraphQL({
        document: MyBulkEditDocument,
        operationName: "MyBulkEdit",
        variables: { id },
        signal,
      });
      return data.myBulkEdit ? getFragmentData(BulkEditJobFieldsFragmentDoc, data.myBulkEdit) : null;
    },
  }),
});
