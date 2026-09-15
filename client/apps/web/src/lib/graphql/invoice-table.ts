import {
  InvoiceRegisterDocument,
  type InvoiceRegisterQuery,
} from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";

export type InvoiceRegisterRow = NonNullable<
  InvoiceRegisterQuery["invoices"]["edges"]
>[number]["node"];

export const invoiceRegisterTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: InvoiceRegisterDocument,
  operationName: "InvoiceRegister",
  connectionKey: "invoices",
});
