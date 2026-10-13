import { translate } from "@trenova/shared/i18n/runtime";
import { customerTrackingFieldSchema } from "@trenova/shared/types/shipment-tracking";
import { z } from "zod";

const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

/** Splits a free-form recipient list the way the server does. */
export function splitRecipients(raw: string): string[] {
  const seen = new Set<string>();
  for (const part of raw.split(/[,;\n\t]/)) {
    const address = part.trim().toLowerCase();
    if (address !== "") {
      seen.add(address);
    }
  }
  return [...seen];
}

/**
 * The customer's tracking settings as a person edits them, holding the limits
 * the server enforces so a mistake is caught before the save.
 */
export const customerTrackingSettingsSchema = z.object({
  linkEnabled: z.boolean(),
  visibleFields: z.array(customerTrackingFieldSchema),
  linkTtlDays: z
    .number({ error: () => translate("Enter how many days a link lasts") })
    .int()
    .min(1, { error: () => translate("A tracking link must last at least 1 day") })
    .max(180, { error: () => translate("A tracking link can last at most 180 days") }),
  notifyPickedUp: z.boolean(),
  notifyDelayed: z.boolean(),
  notifyDelivered: z.boolean(),
  notifyPodReady: z.boolean(),
  includeLink: z.boolean(),
  recipients: z
    .string()
    .max(2000, { error: () => translate("Recipients cannot be longer than 2000 characters") })
    .refine((value) => splitRecipients(value).length <= 20, {
      error: () => translate("List at most 20 recipients"),
    })
    .refine((value) => splitRecipients(value).every((address) => EMAIL_PATTERN.test(address)), {
      error: () => translate("Each recipient must be a valid email address"),
    }),
});

export type CustomerTrackingSettings = z.infer<typeof customerTrackingSettingsSchema>;
