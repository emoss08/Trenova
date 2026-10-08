import { z } from "zod";
import { translate } from "@trenova/shared/i18n/runtime";

/** The most sheets one print run asks for; the server refuses more than fifty. */
export const MAX_COVER_SHEETS = 20;

/**
 * Asking one of the person's computers to scan or catch a print. Empty
 * strings mean "the computer's default scanner", "the organization's default
 * profile" and "no document type"; they are sent as null.
 */
export const captureRequestFormSchema = z.object({
  deviceId: z.string().min(1, { error: () => translate("Choose a computer") }),
  sourceName: z.string(),
  profileId: z.string(),
  documentTypeId: z.string(),
});

export type CaptureRequestFormValues = z.infer<typeof captureRequestFormSchema>;

export const coverSheetFormSchema = z.object({
  copies: z
    .number({ error: () => translate("Enter how many sheets to print") })
    .int({ error: () => translate("Enter a whole number of sheets") })
    .min(1, { error: () => translate("Print at least one sheet") })
    .max(MAX_COVER_SHEETS, { error: () => translate("Print at most 20 sheets at a time") }),
  documentTypeId: z.string(),
});

export type CoverSheetFormValues = z.infer<typeof coverSheetFormSchema>;

/** Empty is "not chosen", which the API takes as null. */
export function emptyToNull(value: string): string | null {
  return value === "" ? null : value;
}
