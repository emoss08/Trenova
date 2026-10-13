import { FactoringCompanyTableDocument } from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";

export const factoringCompanyTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: FactoringCompanyTableDocument,
  operationName: "FactoringCompanyTable",
  connectionKey: "factoringCompanies",
});
