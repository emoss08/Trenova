import { api } from "@trenova/shared/lib/api";
import { safeParse } from "@trenova/shared/lib/parse";
import { commitInvoiceRunResultSchema } from "@trenova/shared/types/invoice-run";
import {
  openStatementListSchema,
  openStatementSchema,
  type OpenStatement,
} from "@trenova/shared/types/statement";

/**
 * Statements are how a biller reaches invoice runs.
 *
 * The run endpoints under /billing/invoice-runs exist and are exercised
 * server-side by the scheduled sweep, but nothing in the client drives a run
 * directly: billing a statement previews, adjusts and commits one in a single
 * call. Wrappers for the rest would be code no screen calls.
 */
export class InvoiceRunService {
  /** Every statement customer's open period, derived live from the billing queue. */
  public async listStatements() {
    const response = await api.get("/billing/statements/");
    return safeParse(openStatementListSchema, response, "OpenStatementList");
  }

  public async getStatement(customerId: string) {
    const response = await api.get<OpenStatement>(`/billing/statements/${customerId}/`);
    return safeParse(openStatementSchema, response, "OpenStatement");
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
    const response = await api.post(`/billing/statements/${customerId}/bill/`, {
      reason,
      exclude: heldBillingQueueItemIds.map((billingQueueItemId) => ({
        billingQueueItemId,
        reason: "Held back by the biller before this statement billed",
      })),
    });
    return safeParse(commitInvoiceRunResultSchema, response, "CommitInvoiceRunResult");
  }
}
