import type {
  BillingTransferFailureCode,
  BulkBillingTransferResult,
} from "@/lib/graphql/billing-transfer";
import { buildCsv, spreadsheetSafeText, type ExportColumn } from "@/lib/data-table-export";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";

export const BILLING_TRANSFER_FAILURE_REASONS: Record<
  BillingTransferFailureCode,
  { label: string; description: string }
> = {
  NotFound: {
    label: "Shipment not found",
    description: "The shipment no longer exists or belongs to another organization.",
  },
  InvalidStatus: {
    label: "Not ready to bill",
    description:
      "Only Completed or Ready to Invoice shipments can transfer. The shipment's status changed after it was listed.",
  },
  AlreadyTransferred: {
    label: "Already in billing",
    description: "The shipment is already in the billing queue.",
  },
  RequirementsUnmet: {
    label: "Missing billing requirements",
    description:
      "The customer's billing requirements are not met and the billing policy blocks transfer until they are.",
  },
  RateValidation: {
    label: "Rate validation failed",
    description: "The rate did not pass validation and the billing policy blocks transfer.",
  },
  ReturnToOperations: {
    label: "Returned to operations",
    description:
      "The billing policy sends shipments with review items back to operations to be corrected first.",
  },
  Unexpected: {
    label: "Transfer failed",
    description: "Something outside the billing policy stopped the transfer. Try it again.",
  },
};

const NON_RETRYABLE_FAILURES: ReadonlySet<BillingTransferFailureCode> = new Set([
  "NotFound",
  "AlreadyTransferred",
]);

export type BulkBillingTransferOutcome = {
  results: BulkBillingTransferResult[];
  notProcessedIds: string[];
};

export type BulkBillingTransferSummary = {
  transferred: number;
  markedReadyToInvoice: number;
  notTransferred: number;
  notProcessed: number;
  total: number;
};

export function summarizeBulkBillingTransfer({
  results,
  notProcessedIds,
}: BulkBillingTransferOutcome): BulkBillingTransferSummary {
  let transferred = 0;
  let markedReadyToInvoice = 0;
  for (const result of results) {
    if (!result.success) continue;
    transferred++;
    if (result.markedReadyToInvoice) markedReadyToInvoice++;
  }

  return {
    transferred,
    markedReadyToInvoice,
    notTransferred: results.length - transferred,
    notProcessed: notProcessedIds.length,
    total: results.length + notProcessedIds.length,
  };
}

export function retryableShipmentIds({
  results,
  notProcessedIds,
}: BulkBillingTransferOutcome): string[] {
  const ids: string[] = [];
  for (const result of results) {
    if (result.success) continue;
    if (result.failureCode && NON_RETRYABLE_FAILURES.has(result.failureCode)) continue;
    ids.push(result.shipmentId);
  }
  ids.push(...notProcessedIds);
  return ids;
}

/**
 * A retry answers again for some of the shipments, so its outcomes replace the
 * earlier ones in place and everything it did not touch keeps its first answer.
 * Shipments that were unprocessed before and have an answer now join the end.
 */
export function mergeBulkBillingTransferRetry(
  previous: BulkBillingTransferOutcome,
  retry: BulkBillingTransferOutcome,
): BulkBillingTransferOutcome {
  const retried = new Map(retry.results.map((result) => [result.shipmentId, result]));
  const stillUnprocessed = new Set(retry.notProcessedIds);
  const placed = new Set<string>();

  const results: BulkBillingTransferResult[] = [];
  for (const result of previous.results) {
    if (stillUnprocessed.has(result.shipmentId)) continue;
    const replacement = retried.get(result.shipmentId);
    results.push(replacement ?? result);
    if (replacement) placed.add(result.shipmentId);
  }
  for (const result of retry.results) {
    if (!placed.has(result.shipmentId)) results.push(result);
  }

  const notProcessedIds = previous.notProcessedIds.filter(
    (shipmentId) => !retried.has(shipmentId) && !stillUnprocessed.has(shipmentId),
  );
  notProcessedIds.push(...retry.notProcessedIds);

  return { results, notProcessedIds };
}

type ReportRow =
  | { kind: "result"; result: BulkBillingTransferResult }
  | { kind: "notProcessed"; shipmentId: string };

function reportColumns(t: TranslateFn): ExportColumn<ReportRow>[] {
  const resultOf = (row: ReportRow) => (row.kind === "result" ? row.result : null);

  return [
    {
      id: "proNumber",
      header: t("PRO Number"),
      getValue: (row) => spreadsheetSafeText(resultOf(row)?.proNumber),
    },
    {
      id: "shipmentId",
      header: t("Shipment ID"),
      getValue: (row) => (row.kind === "result" ? row.result.shipmentId : row.shipmentId),
    },
    {
      id: "outcome",
      header: t("Outcome"),
      getValue: (row) => {
        const result = resultOf(row);
        if (!result) return t("Not processed");
        return result.success ? t("Transferred") : t("Not transferred");
      },
    },
    {
      id: "reason",
      header: t("Reason"),
      getValue: (row) => {
        const code = resultOf(row)?.failureCode;
        return code ? t(BILLING_TRANSFER_FAILURE_REASONS[code].label) : null;
      },
    },
    {
      id: "details",
      header: t("Details"),
      getValue: (row) => spreadsheetSafeText(resultOf(row)?.error),
    },
    {
      id: "missingDocuments",
      header: t("Missing Documents"),
      getValue: (row) =>
        spreadsheetSafeText(
          resultOf(row)
            ?.missingRequirements.map((requirement) => requirement.documentTypeName)
            .join("; "),
        ),
    },
    {
      id: "validationFailures",
      header: t("Validation Failures"),
      getValue: (row) =>
        spreadsheetSafeText(
          resultOf(row)
            ?.validationFailures.map((failure) => failure.message)
            .join("; "),
        ),
    },
    {
      id: "billingQueueNumber",
      header: t("Billing Queue Number"),
      getValue: (row) => spreadsheetSafeText(resultOf(row)?.billingQueueItem?.number),
    },
    {
      id: "markedReadyToInvoice",
      header: t("Marked Ready to Invoice"),
      getValue: (row) => (resultOf(row)?.markedReadyToInvoice ? t("Yes") : t("No")),
    },
  ];
}

export function buildBulkBillingTransferReportCsv(
  { results, notProcessedIds }: BulkBillingTransferOutcome,
  t: TranslateFn,
): string {
  const rows: ReportRow[] = [
    ...results.map((result): ReportRow => ({ kind: "result", result })),
    ...notProcessedIds.map((shipmentId): ReportRow => ({ kind: "notProcessed", shipmentId })),
  ];
  return buildCsv(rows, reportColumns(t));
}
