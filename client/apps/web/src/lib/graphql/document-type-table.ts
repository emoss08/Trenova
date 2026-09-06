import { DocumentTypeTableDocument } from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";

export const documentTypeTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: DocumentTypeTableDocument,
  operationName: "DocumentTypeTable",
  connectionKey: "documentTypes",
});
