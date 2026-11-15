import {
  FactoringCompanyTableDocument,
} from "@trenova/graphql/generated/documents/factoring-company/table";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";

export const factoringCompanyTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: FactoringCompanyTableDocument,
  operationName: "FactoringCompanyTable",
  connectionKey: "factoringCompanies",
});
