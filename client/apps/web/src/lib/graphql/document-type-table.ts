import {
  DocumentTypeTableDocument,
} from "@trenova/graphql/generated/documents/document-type/table";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";

export const documentTypeTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: DocumentTypeTableDocument,
  operationName: "DocumentTypeTable",
  connectionKey: "documentTypes",
});
