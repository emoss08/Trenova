import {
  BillStatementNowDocument,
  OpenStatementDocument,
  OpenStatementsDocument,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { safeParse } from "@trenova/shared/lib/parse";
import { z } from "zod";
import { commitInvoiceRunResultSchema } from "@trenova/shared/types/invoice-run";
import { openStatementSchema } from "@trenova/shared/types/statement";

/** Statements are how a biller reaches invoice runs: the open period, and billing it. */
export class InvoiceRunService {
  /** Every statement customer's open period, derived live from the billing queue. */
  public async listStatements(options: { signal?: AbortSignal } = {}) {
    const data = await requestGraphQL<{ openStatements: unknown }>({
      document: OpenStatementsDocument,
      operationName: "OpenStatements",
      signal: options.signal,
    });
    return safeParse(z.array(openStatementSchema), data.openStatements ?? [], "OpenStatementList");
  }

  public async getStatement(customerId: string, options: { signal?: AbortSignal } = {}) {
    const data = await requestGraphQL<{ openStatement: unknown }>({
      document: OpenStatementDocument,
      operationName: "OpenStatement",
      variables: { customerId },
      signal: options.signal,
    });
    return safeParse(openStatementSchema, data.openStatement, "OpenStatement");
  }

  /**
   * Bills an open period. The reason is recorded on the run and is required only
   * when the period has not closed yet; held billing-queue items stay approved
   * and uninvoiced for the next period.
   */
  public async billStatementNow(
    customerId: string,
    reason: string,
    heldBillingQueueItemIds: string[] = [],
  ) {
    const data = await requestGraphQL<{ billStatementNow: unknown }>({
      document: BillStatementNowDocument,
      operationName: "BillStatementNow",
      variables: {
        customerId,
        input: {
          reason,
          exclude: heldBillingQueueItemIds.map((billingQueueItemId) => ({
            billingQueueItemId,
            reason: "Held back by the biller before this statement billed",
          })),
        },
      },
    });
    return safeParse(commitInvoiceRunResultSchema, data.billStatementNow, "CommitInvoiceRunResult");
  }
}
