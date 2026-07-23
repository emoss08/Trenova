import {
  DocumentTypeTableDocument,
  type DocumentTypeTableQueryVariables,
} from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DocumentType } from "@trenova/shared/types/document-type";

export const documentTypeTableGraphQLConfig = defineDataTableGraphQLConfig<
  DocumentType,
  DocumentTypeTableQueryVariables
>({
  document: DocumentTypeTableDocument,
  operationName: "DocumentTypeTable",
  connectionKey: "documentTypes",
});
