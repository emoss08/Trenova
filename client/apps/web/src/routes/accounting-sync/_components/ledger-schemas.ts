import { isLedgerObjectType } from "@/lib/accounting-sync";
import type {
  AccountingLedgerGranularity,
  AccountingSyncMode,
  AccountingSyncObjectType,
} from "@trenova/graphql/generated/graphql";
import { z } from "zod";
import { translate } from "@trenova/shared/i18n/runtime";

export const ACCOUNTING_SYNC_OBJECT_TYPES = [
  "Customer",
  "Invoice",
  "CreditMemo",
  "DebitMemo",
  "CustomerPayment",
  "CreditApplication",
  "CarrierVendor",
  "DriverVendor",
  "CarrierBill",
  "CarrierBillPayment",
  "DriverBill",
  "DriverBillPayment",
  "JournalEntry",
  "JournalSummary",
] as const satisfies readonly AccountingSyncObjectType[];

export const ACCOUNTING_BACKFILL_OBJECT_TYPES = [
  "Invoice",
  "DebitMemo",
  "CreditMemo",
  "CustomerPayment",
  "CreditApplication",
  "CarrierBill",
  "CarrierBillPayment",
  "DriverBill",
  "DriverBillPayment",
  "JournalEntry",
  "JournalSummary",
] as const satisfies readonly AccountingSyncObjectType[];

type BackfillObjectType = (typeof ACCOUNTING_BACKFILL_OBJECT_TYPES)[number];

const DRIVER_BACKFILL_OBJECT_TYPES: readonly AccountingSyncObjectType[] = [
  "DriverBill",
  "DriverBillPayment",
];

export function backfillObjectTypes(connection: {
  syncMode: AccountingSyncMode;
  ledgerGranularity?: AccountingLedgerGranularity | null;
  syncsDriverSettlements: boolean;
}): BackfillObjectType[] {
  if (connection.syncMode === "Ledger") {
    return [connection.ledgerGranularity === "DailySummary" ? "JournalSummary" : "JournalEntry"];
  }
  return ACCOUNTING_BACKFILL_OBJECT_TYPES.filter(
    (type) =>
      !isLedgerObjectType(type) &&
      (connection.syncsDriverSettlements || !DRIVER_BACKFILL_OBJECT_TYPES.includes(type)),
  );
}

const MAX_REASON_LENGTH = 500;

export const pauseSchema = z.object({
  reason: z
    .string()
    .trim()
    .max(MAX_REASON_LENGTH, {
      error: () => translate("Keep the reason under 500 characters"),
    }),
});

export type PauseValues = z.infer<typeof pauseSchema>;

export const skipSchema = z.object({
  reason: z
    .string()
    .trim()
    .min(1, { error: () => translate("Say why this document is not sent") })
    .max(MAX_REASON_LENGTH, { error: () => translate("Keep the reason under 500 characters") }),
});

export type SkipValues = z.infer<typeof skipSchema>;

export function backfillSchema(latestAllowed: number) {
  return z
    .object({
      rangeStart: z
        .number({ error: () => translate("Choose the first document date") })
        .int()
        .positive(),
      rangeEnd: z
        .number({ error: () => translate("Choose the last document date") })
        .int()
        .positive()
        .max(latestAllowed, { error: () => translate("The range cannot end in the future") }),
      objectTypes: z.array(z.enum(ACCOUNTING_BACKFILL_OBJECT_TYPES)),
    })
    .refine((value) => value.rangeStart <= value.rangeEnd, {
      error: () => translate("The range must end on or after the day it starts"),
      path: ["rangeEnd"],
    });
}

export type BackfillValues = z.infer<ReturnType<typeof backfillSchema>>;
