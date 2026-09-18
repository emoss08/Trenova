import type {
  BillingTransferFailureCode,
  BillingTransferRunItem,
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

type ReportRow = BillingTransferRunItem;

function reportColumns(t: TranslateFn): ExportColumn<ReportRow>[] {
  return [
    {
      id: "proNumber",
      header: t("PRO Number"),
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
      header: t("Missing Documents"),
      getValue: (row) =>
        spreadsheetSafeText(
          row.missingRequirements.map((requirement) => requirement.documentTypeName).join("; "),
        ),
    },
    {
      id: "validationFailures",
      header: t("Validation Failures"),
      getValue: (row) =>
        spreadsheetSafeText(row.validationFailures.map((failure) => failure.message).join("; ")),
    },
    {
      id: "billingQueueNumber",
      header: t("Billing Queue Number"),
      getValue: (row) => spreadsheetSafeText(row.billingQueueNumber),
    },
    {
      id: "markedReadyToInvoice",
      header: t("Marked Ready to Invoice"),
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
