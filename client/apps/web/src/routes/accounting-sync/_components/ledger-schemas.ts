import type { AccountingSyncObjectType } from "@trenova/graphql/generated/graphql";
import { z } from "zod";

export const ACCOUNTING_SYNC_OBJECT_TYPES = [
  "Customer",
  "Invoice",
  "CreditMemo",
  "DebitMemo",
  "CustomerPayment",
  "CreditApplication",
] as const satisfies readonly AccountingSyncObjectType[];

const MAX_REASON_LENGTH = 500;

export const pauseSchema = z.object({
  reason: z.string().trim().max(MAX_REASON_LENGTH, {
    error: "Keep the reason under 500 characters",
  }),
});

export type PauseValues = z.infer<typeof pauseSchema>;

export const skipSchema = z.object({
  reason: z
    .string()
    .trim()
    .min(1, { error: "Say why this document is not sent" })
    .max(MAX_REASON_LENGTH, { error: "Keep the reason under 500 characters" }),
});

export type SkipValues = z.infer<typeof skipSchema>;

export function backfillSchema(latestAllowed: number) {
  return z
    .object({
      rangeStart: z.number({ error: "Choose the first document date" }).int().positive(),
      rangeEnd: z
        .number({ error: "Choose the last document date" })
        .int()
        .positive()
        .max(latestAllowed, { error: "The range cannot end in the future" }),
      objectTypes: z.array(z.enum(ACCOUNTING_SYNC_OBJECT_TYPES)),
    })
    .refine((value) => value.rangeStart <= value.rangeEnd, {
      error: "The range must end on or after the day it starts",
      path: ["rangeEnd"],
    });
}

export type BackfillValues = z.infer<ReturnType<typeof backfillSchema>>;
