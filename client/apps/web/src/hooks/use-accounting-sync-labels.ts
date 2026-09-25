import type {
  AccountingSyncErrorCategory,
  AccountingSyncObjectType,
  AccountingSyncOperation,
  AccountingSyncRecordStatus,
  AccountingSyncSourceEvent,
} from "@trenova/graphql/generated/graphql";
import { useT } from "@trenova/shared/i18n/use-t";
import { useMemo } from "react";

export type AccountingSyncLabels = {
  status: Record<AccountingSyncRecordStatus, string>;
  objectType: Record<AccountingSyncObjectType, string>;
  operation: Record<AccountingSyncOperation, string>;
  errorCategory: Record<AccountingSyncErrorCategory, string>;
  sourceEvent: Record<AccountingSyncSourceEvent, string>;
};

export function useAccountingSyncLabels(): AccountingSyncLabels {
  const t = useT();

  return useMemo(
    () => ({
      status: {
        Queued: t("Queued"),
        AwaitingApproval: t("Waiting for release"),
        InFlight: t("Sending"),
        Retrying: t("Retrying"),
        Synced: t("Synced"),
        Blocked: t("Held"),
        DeadLettered: t("Failed"),
        Skipped: t("Skipped"),
        Superseded: t("Replaced"),
      },
      objectType: {
        Customer: t("Customer"),
        Invoice: t("Invoice"),
        CreditMemo: t("Credit memo"),
        DebitMemo: t("Debit memo"),
        CustomerPayment: t("Customer payment"),
        CreditApplication: t("Credit application"),
        CarrierVendor: t("Carrier"),
        DriverVendor: t("Owner-operator"),
        CarrierBill: t("Carrier settlement"),
        CarrierBillPayment: t("Carrier settlement payment"),
        DriverBill: t("Owner-operator settlement"),
        DriverBillPayment: t("Owner-operator settlement payment"),
      },
      operation: {
        Create: t("Create"),
        Update: t("Update"),
        Void: t("Void"),
      },
      errorCategory: {
        Transient: t("Temporary failure"),
        RateLimited: t("Rate limited"),
        Auth: t("Authorization"),
        Validation: t("Rejected"),
        Mapping: t("Missing mapping"),
        ClosedPeriod: t("Closed period"),
        Currency: t("Currency"),
        Duplicate: t("Duplicate"),
        NotFound: t("Not found"),
        Conflict: t("Conflict"),
        Configuration: t("Configuration"),
      },
      sourceEvent: {
        InvoicePosted: t("Invoice posted"),
        CreditMemoPosted: t("Credit memo posted"),
        DebitMemoPosted: t("Debit memo posted"),
        AdjustmentCreditMemo: t("Invoice adjustment"),
        CustomerPaymentPosted: t("Payment posted"),
        CustomerPaymentApplied: t("Payment applied"),
        CustomerPaymentReversed: t("Payment reversed"),
        CreditMemoApplied: t("Credit memo applied"),
        CreditMemoUnapplied: t("Credit memo unapplied"),
        CustomerUpdated: t("Customer updated"),
        CarrierSettlementPosted: t("Carrier settlement posted"),
        CarrierSettlementVoided: t("Carrier settlement voided"),
        CarrierSettlementPaid: t("Carrier settlement paid"),
        DriverSettlementPosted: t("Owner-operator settlement posted"),
        DriverSettlementVoided: t("Owner-operator settlement voided"),
        DriverSettlementPaid: t("Owner-operator settlement paid"),
        CarrierUpdated: t("Carrier updated"),
        DriverUpdated: t("Owner-operator updated"),
        DependencyOf: t("Needed by another document"),
        SafetyNet: t("Found by the hourly check"),
        Backfill: t("Backfill"),
      },
    }),
    [t],
  );
}
