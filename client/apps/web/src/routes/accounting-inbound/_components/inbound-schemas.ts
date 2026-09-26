import { z } from "zod";

const MAX_NOTE_LENGTH = 500;

export const ignoreInboundSchema = z.object({
  note: z
    .string()
    .trim()
    .min(1, { error: "Say why this payment stays out of Trenova" })
    .max(MAX_NOTE_LENGTH, { error: "Keep the note under 500 characters" }),
});

export type IgnoreInboundValues = z.infer<typeof ignoreInboundSchema>;
