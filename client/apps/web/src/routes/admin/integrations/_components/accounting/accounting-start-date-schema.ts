import {
  ACCOUNTING_INBOUND_POLICIES,
  ACCOUNTING_LEDGER_GRANULARITIES,
  ACCOUNTING_SYNC_MODES,
} from "@/lib/accounting-sync";
import { z } from "zod";
import { translate } from "@trenova/shared/i18n/runtime";

export function accountingStartDateSchema(latestAllowed: number) {
  return z.object({
    startDate: z
      .number({ error: () => translate("Choose the first day documents are sent from") })
      .int()
      .positive({ error: () => translate("Choose the first day documents are sent from") })
      .max(latestAllowed, { error: () => translate("The start date cannot be in the future") }),
    autoSync: z.boolean(),
    driverSettlements: z.boolean(),
    backfill: z.boolean(),
    openingBalances: z.boolean(),
  });
}

export function accountingModeSchema(ledgerAvailable: boolean) {
  return z.object({
    mode: z
      .enum(ACCOUNTING_SYNC_MODES, { error: () => translate("Choose what is sent") })
      .refine((value) => ledgerAvailable || value !== "Ledger", {
        error: () => translate("This accounting system cannot receive journal entries"),
      }),
    granularity: z.enum(ACCOUNTING_LEDGER_GRANULARITIES, {
      error: () => translate("Choose how journal entries are sent"),
    }),
  });
}

export type AccountingModeValues = z.infer<ReturnType<typeof accountingModeSchema>>;

export function accountingModeInput(values: AccountingModeValues, ledgerAvailable: boolean) {
  const mode = ledgerAvailable ? values.mode : "Document";
  return {
    mode,
    granularity: mode === "Ledger" ? values.granularity : null,
  };
}

export const accountingSyncSettingsSchema = z.object({
  autoSync: z.boolean(),
  driverSettlements: z.boolean(),
  inboundPayments: z.enum(ACCOUNTING_INBOUND_POLICIES),
});

export type AccountingSyncSettingsValues = z.infer<typeof accountingSyncSettingsSchema>;

export type AccountingStartDateValues = z.infer<ReturnType<typeof accountingStartDateSchema>>;

export function startDateInClosedBooks(
  startDate: number | null | undefined,
  booksClosedThrough: number | null | undefined,
): boolean {
  return startDate != null && booksClosedThrough != null && startDate <= booksClosedThrough;
}
