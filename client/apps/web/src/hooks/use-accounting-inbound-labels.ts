import type {
  AccountingAppliedObjectType,
  AccountingInboundChangeKind,
  AccountingInboundChangeReason,
  AccountingInboundChangeStatus,
  AccountingInboundDocumentKind,
  AccountingInboundPaymentPolicy,
} from "@trenova/graphql/generated/graphql";
import { useT } from "@trenova/shared/i18n/use-t";
import { useMemo } from "react";

export type AccountingInboundLabels = {
  status: Record<AccountingInboundChangeStatus, string>;
  kind: Record<AccountingInboundChangeKind, string>;
  reason: Record<AccountingInboundChangeReason, string>;
  documentKind: Record<AccountingInboundDocumentKind, string>;
  appliedObject: Record<AccountingAppliedObjectType, string>;
  policy: Record<AccountingInboundPaymentPolicy, string>;
};

export function useAccountingInboundLabels(): AccountingInboundLabels {
  const t = useT();

  return useMemo(
    () => ({
      status: {
        Detected: t("Just read"),
        Proposed: t("Waiting for you"),
        Applied: t("Applied"),
        Ignored: t("Ignored"),
        Superseded: t("Voided in the books"),
      },
      kind: {
        CustomerPayment: t("Customer payment"),
        BillPayment: t("Bill payment"),
      },
      reason: {
        PolicyPropose: t("Waiting to be applied"),
        PeriodNotOpen: t("Period not open"),
        UnknownDocument: t("Pays documents Trenova did not send"),
        PartyMismatch: t("More than one party"),
        Overpayment: t("Pays more than is open"),
        PartialBillPayment: t("Partial settlement payment"),
        AlreadyPaid: t("Already paid in Trenova"),
        CurrencyMismatch: t("Different currency"),
        Voided: t("Voided in the books"),
        NotTrenovaDocument: t("Not a Trenova document"),
        SentFromTrenova: t("Sent from Trenova"),
        ApplyFailed: t("Could not be applied"),
      },
      documentKind: {
        Invoice: t("Invoice"),
        CreditMemo: t("Credit memo"),
        DebitMemo: t("Debit memo"),
        Bill: t("Vendor bill"),
        VendorCredit: t("Vendor credit"),
        Other: t("Other"),
      },
      appliedObject: {
        CustomerPayment: t("Customer payment"),
        CreditMemoApplication: t("Credit application"),
        CarrierSettlement: t("Carrier settlement"),
        DriverSettlement: t("Owner-operator settlement"),
      },
      policy: {
        Propose: t("Wait for someone to apply them"),
        Apply: t("Apply them automatically"),
        Off: t("Leave them out of Trenova"),
      },
    }),
    [t],
  );
}
