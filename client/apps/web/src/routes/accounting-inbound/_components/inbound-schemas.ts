import { z } from "zod";
import { translate } from "@trenova/shared/i18n/runtime";

const MAX_NOTE_LENGTH = 500;

export const ignoreInboundSchema = z.object({
  note: z
    .string()
    .trim()
    .min(1, { error: () => translate("Say why this payment stays out of Trenova") })
    .max(MAX_NOTE_LENGTH, { error: () => translate("Keep the note under 500 characters") }),
});

export type IgnoreInboundValues = z.infer<typeof ignoreInboundSchema>;
