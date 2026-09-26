import { ACCOUNTING_INBOUND_POLICIES } from "@/lib/accounting-sync";
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
  });
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
