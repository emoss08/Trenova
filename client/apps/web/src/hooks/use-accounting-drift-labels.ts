import type {
  AccountingDriftDirection,
  AccountingDriftFixObject,
  AccountingDriftKind,
  AccountingDriftResolution,
  AccountingDriftStatus,
} from "@trenova/graphql/generated/graphql";
import { useT } from "@trenova/shared/i18n/use-t";
import { useMemo } from "react";

export type AccountingDriftLabels = {
  status: Record<AccountingDriftStatus, string>;
  kind: Record<AccountingDriftKind, string>;
  resolution: Record<AccountingDriftResolution, string>;
  direction: Record<AccountingDriftDirection, string>;
  fixObject: Record<AccountingDriftFixObject, string>;
};

export function useAccountingDriftLabels(): AccountingDriftLabels {
  const t = useT();

  return useMemo(
    () => ({
      status: {
        Open: t("Open"),
        Resolved: t("Resolved"),
        Dismissed: t("Dismissed"),
      },
      kind: {
        AmountMismatch: t("Different total"),
        StatusMismatch: t("Voided only in Trenova"),
        DeletedInProvider: t("Deleted in the books"),
        VoidedInProvider: t("Voided in the books"),
        CustomerBalanceMismatch: t("Different customer balance"),
      },
      resolution: {
        PushedTrenovaValue: t("Trenova's value sent"),
        AdjustedTrenova: t("Trenova adjusted"),
        NoLongerDiffers: t("No longer differs"),
        Dismissed: t("Dismissed"),
      },
      direction: {
        PushTrenovaValue: t("Push Trenova's value"),
        AdjustTrenova: t("Adjust Trenova"),
      },
      fixObject: {
        SyncRecord: t("Sync record"),
        CreditMemo: t("Credit memo"),
        DebitMemo: t("Debit memo"),
        InvoiceVoid: t("Invoice void"),
        PaymentReversal: t("Payment reversal"),
      },
    }),
    [t],
  );
}
