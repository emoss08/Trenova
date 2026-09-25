import { z } from "zod";

export function accountingStartDateSchema(latestAllowed: number) {
  return z.object({
    startDate: z
      .number({ error: "Choose the first day documents are sent from" })
      .int()
      .positive({ error: "Choose the first day documents are sent from" })
      .max(latestAllowed, { error: "The start date cannot be in the future" }),
    autoSync: z.boolean(),
    backfill: z.boolean(),
  });
}

export type AccountingStartDateValues = z.infer<ReturnType<typeof accountingStartDateSchema>>;

export function startDateInClosedBooks(
  startDate: number | null | undefined,
  booksClosedThrough: number | null | undefined,
): boolean {
  return startDate != null && booksClosedThrough != null && startDate <= booksClosedThrough;
}
