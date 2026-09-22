import { InvoiceRegisterDocument } from "@trenova/graphql/generated/graphql";
import { defineDataTableGraphQLConfig } from "@trenova/shared/lib/graphql/data-table";
import type { DataTableRow } from "@trenova/shared/types/graphql-connection";

/*
 * DataTableRow rather than reaching into the query type by hand.
 *
 * The connection node is a masked fragment reference: read off it directly and
 * every column, every cell renderer and every test fixture sees a type with no
 * properties. DataTableRow is the same node with the masks resolved, which is
 * what defineDataTableGraphQLConfig below infers anyway — so the two now agree.
 */
export type InvoiceRegisterRow = DataTableRow<typeof InvoiceRegisterDocument, "invoices">;

export const invoiceRegisterTableGraphQLConfig = defineDataTableGraphQLConfig({
  document: InvoiceRegisterDocument,
  operationName: "InvoiceRegister",
  connectionKey: "invoices",
});
