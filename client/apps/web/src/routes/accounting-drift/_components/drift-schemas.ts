import { z } from "zod";
import { translate } from "@trenova/shared/i18n/runtime";

const MAX_NOTE_LENGTH = 500;

export const dismissDriftSchema = z.object({
  note: z
    .string()
    .trim()
    .min(1, { error: () => translate("Say why both sides stay as they are") })
    .max(MAX_NOTE_LENGTH, { error: () => translate("Keep the note under 500 characters") }),
});

export type DismissDriftValues = z.infer<typeof dismissDriftSchema>;
