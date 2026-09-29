import {
  ACCOUNTING_INBOUND_POLICIES,
  ACCOUNTING_LEDGER_GRANULARITIES,
  ACCOUNTING_SYNC_MODES,
} from "@/lib/accounting-sync";
import { z } from "zod";

export function accountingStartDateSchema(latestAllowed: number) {
  return z.object({
    startDate: z
      .number({ error: "Choose the first day documents are sent from" })
      .int()
      .positive({ error: "Choose the first day documents are sent from" })
      .max(latestAllowed, { error: "The start date cannot be in the future" }),
    autoSync: z.boolean(),
    driverSettlements: z.boolean(),
    backfill: z.boolean(),
    openingBalances: z.boolean(),
  });
}

export const accountingModeSchema = z.object({
  mode: z.enum(ACCOUNTING_SYNC_MODES, { error: "Choose what is sent" }),
  granularity: z.enum(ACCOUNTING_LEDGER_GRANULARITIES, {
    error: "Choose how journal entries are sent",
  }),
});

export type AccountingModeValues = z.infer<typeof accountingModeSchema>;

export function accountingModeInput(values: AccountingModeValues) {
  return {
    mode: values.mode,
    granularity: values.mode === "Ledger" ? values.granularity : null,
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
