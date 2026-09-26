import { useT } from "@trenova/shared/i18n/use-t";
import { useMemo } from "react";

export type JournalReviewStatus = "Pending" | "Approved";

export const JOURNAL_REVIEW_STATUSES: readonly JournalReviewStatus[] = ["Pending", "Approved"];

export type JournalReviewLabels = {
  status: Record<JournalReviewStatus, string>;
  source: (referenceType: string) => string;
};

export function useJournalReviewLabels(): JournalReviewLabels {
  const t = useT();

  return useMemo(() => {
    const sources: Record<string, string> = {
      InvoicePosted: t("Invoice posted"),
      CreditMemoPosted: t("Credit memo posted"),
      DebitMemoPosted: t("Debit memo posted"),
      InvoiceAdjustmentWriteOff: t("Invoice write-off"),
      CustomerPaymentPosted: t("Customer payment posted"),
      CustomerPaymentApplied: t("Customer payment applied"),
      CustomerPaymentReversed: t("Customer payment reversed"),
      CustomerShortPayRecognized: t("Customer short pay"),
      DriverSettlementPosted: t("Driver settlement posted"),
      DriverSettlementVoided: t("Driver settlement voided"),
      DriverSettlementPaid: t("Driver settlement paid"),
      CarrierSettlementPosted: t("Carrier settlement posted"),
      CarrierSettlementVoided: t("Carrier settlement voided"),
      CarrierSettlementPaid: t("Carrier settlement paid"),
      EscrowInterestAccrued: t("Escrow interest accrued"),
    };

    return {
      status: {
        Pending: t("Awaiting approval"),
        Approved: t("Ready to post"),
      },
      source: (referenceType: string) => sources[referenceType] ?? referenceType,
    };
  }, [t]);
}
