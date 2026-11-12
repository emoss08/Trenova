import { translate } from "@trenova/shared/i18n/runtime";
import { z } from "zod";

/** The most teammates one share can name, as the server enforces it. */
export const MAX_RECORD_SHARE_RECIPIENTS = 25;
/** The longest note a share can carry, as the server enforces it. */
export const MAX_RECORD_SHARE_NOTE_LENGTH = 1000;

/** What the share dialog's form holds: who to tell, and what to tell them. */
export const shareRecordFormSchema = z.object({
  userIds: z
    .array(z.string().min(1))
    .min(1, { error: () => translate("Choose at least one teammate to share with") })
    .max(MAX_RECORD_SHARE_RECIPIENTS, {
      error: () => translate("A record can be shared with at most 25 teammates at a time"),
    }),
  note: z.string().max(MAX_RECORD_SHARE_NOTE_LENGTH, {
    error: () => translate("Note must be 1000 characters or fewer"),
  }),
});
export type ShareRecordFormValues = z.infer<typeof shareRecordFormSchema>;
