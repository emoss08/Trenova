import type {
  BillingTransferFailureCode,
  BillingTransferRunItem,
} from "@/lib/graphql/billing-transfer";
import { buildCsv, spreadsheetSafeText, type ExportColumn } from "@/lib/data-table-export";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";

/**
 * The readiness check's steps, in the order the transfer stage reads them off a bill. Every
 * failure reason belongs to the one step that stops a shipment for it.
 */
export const READINESS_STEPS = [
  { key: "status", label: "Status" },
  { key: "documents", label: "Documents" },
  { key: "rate", label: "Rate" },
  { key: "billed", label: "Not yet billed" },
] as const;

export type ReadinessStep = (typeof READINESS_STEPS)[number]["key"];

export const BILLING_TRANSFER_FAILURE_REASONS: Record<
  BillingTransferFailureCode,
  { label: string; description: string; step: ReadinessStep }
> = {
  NotFound: {
    step: "status",
    label: "Shipment not found",
    description: "The shipment no longer exists or belongs to another organization.",
  },
  InvalidStatus: {
    step: "status",
    label: "Not ready to bill",
    description:
      "Only Completed or Ready to Invoice shipments can transfer. The shipment's status changed after it was listed.",
  },
  AlreadyTransferred: {
    step: "billed",
    label: "Already in billing",
    description: "The shipment is already in the billing queue.",
  },
  RequirementsUnmet: {
    step: "documents",
    label: "Missing billing requirements",
    description:
      "The customer's billing requirements are not met and the billing policy blocks transfer until they are.",
  },
  RateValidation: {
    step: "rate",
    label: "Rate validation failed",
    description: "The rate did not pass validation and the billing policy blocks transfer.",
  },
  ReturnToOperations: {
    step: "documents",
    label: "Returned to operations",
    description:
      "The billing policy sends shipments with review items back to operations to be corrected first.",
  },
  Unexpected: {
    step: "billed",
    label: "Transfer failed",
    description: "Something outside the billing policy stopped the transfer. Try it again.",
  },
};

type ReportRow = BillingTransferRunItem;

function reportColumns(t: TranslateFn): ExportColumn<ReportRow>[] {
  return [
    {
      id: "proNumber",
      header: t("PRO number"),
      getValue: (row) => spreadsheetSafeText(row.proNumber),
    },
    {
      id: "shipmentId",
      header: t("Shipment ID"),
      getValue: (row) => row.shipmentId,
    },
    {
      id: "outcome",
      header: t("Outcome"),
      getValue: (row) => {
        switch (row.status) {
          case "Transferred":
            return t("Transferred");
          case "NotTransferred":
            return t("Not transferred");
          default:
            return t("Not processed");
        }
      },
    },
    {
      id: "reason",
      header: t("Reason"),
      getValue: (row) =>
        row.failureCode ? t(BILLING_TRANSFER_FAILURE_REASONS[row.failureCode].label) : null,
    },
    {
      id: "details",
      header: t("Details"),
      getValue: (row) => spreadsheetSafeText(row.errorMessage),
    },
    {
      id: "missingDocuments",
      header: t("Missing documents"),
      getValue: (row) =>
        spreadsheetSafeText(
          row.missingRequirements.map((requirement) => requirement.documentTypeName).join("; "),
        ),
    },
    {
      id: "validationFailures",
      header: t("Validation failures"),
      getValue: (row) =>
        spreadsheetSafeText(row.validationFailures.map((failure) => failure.message).join("; ")),
    },
    {
      id: "billingQueueNumber",
      header: t("Billing queue number"),
      getValue: (row) => spreadsheetSafeText(row.billingQueueNumber),
    },
    {
      id: "markedReadyToInvoice",
      header: t("Marked ready to invoice"),
      getValue: (row) => (row.markedReadyToInvoice ? t("Yes") : t("No")),
    },
  ];
}

export function buildBulkBillingTransferReportCsv(
  items: readonly BillingTransferRunItem[],
  t: TranslateFn,
): string {
  return buildCsv([...items], reportColumns(t));
}
