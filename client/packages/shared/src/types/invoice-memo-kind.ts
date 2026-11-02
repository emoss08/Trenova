import { z } from "zod";

/**
 * Which memo an invoice or a queue item bills. Its own module, because both the invoice and the
 * billing queue item schemas read it and the invoice schema already imports the queue item's.
 */
export const invoiceMemoKindSchema = z.enum(["Manual", "LateCharge"]);
export type InvoiceMemoKind = z.infer<typeof invoiceMemoKindSchema>;
