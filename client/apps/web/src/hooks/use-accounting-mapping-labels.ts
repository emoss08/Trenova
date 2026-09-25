import type {
  AccountingMappingSource,
  AccountingMappingState,
  AccountingMappingTargetType,
  AccountingReferenceKind,
} from "@trenova/graphql/generated/graphql";
import { useT } from "@trenova/shared/i18n/use-t";
import { useMemo } from "react";

export type AccountingMappingLabels = {
  targetType: (targetType: AccountingMappingTargetType) => string;
  state: (state: AccountingMappingState) => string;
  source: (source: AccountingMappingSource) => string;
  recordKind: (kind: AccountingReferenceKind) => string;
};

export function useAccountingMappingLabels(): AccountingMappingLabels {
  const t = useT();

  return useMemo(() => {
    const targetTypes: Record<AccountingMappingTargetType, string> = {
      AccountRole: t("Account roles"),
      LineType: t("Line types"),
      AccessorialCharge: t("Accessorial charges"),
      ItemRole: t("Item roles"),
      Customer: t("Customers"),
      Carrier: t("Carriers"),
      Driver: t("Owner-operators"),
      GLAccount: t("GL accounts"),
      PaymentTerm: t("Payment terms"),
      PaymentMethod: t("Payment methods"),
    };
    const states: Record<AccountingMappingState, string> = {
      Unmatched: t("Unmatched"),
      Proposed: t("Proposed"),
      Confirmed: t("Confirmed"),
    };
    const sources: Record<AccountingMappingSource, string> = {
      Suggested: t("Matched by Trenova"),
      Model: t("Suggested by the model"),
      Manual: t("Chosen by a person"),
      CreatedInProvider: t("Created from Trenova"),
      Agent: t("Chosen by an agent"),
    };
    const recordKinds: Record<AccountingReferenceKind, string> = {
      Account: t("Account"),
      Item: t("Item"),
      Customer: t("Customer"),
      Vendor: t("Vendor"),
      Term: t("Term"),
      PaymentMethod: t("Payment method"),
    };

    return {
      targetType: (targetType) => targetTypes[targetType],
      state: (state) => states[state],
      source: (source) => sources[source],
      recordKind: (kind) => recordKinds[kind],
    };
  }, [t]);
}
