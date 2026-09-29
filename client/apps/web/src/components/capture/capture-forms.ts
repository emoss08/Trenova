import {
  CAPTURE_DEVICE_NAME_MAX_BYTES,
  CAPTURE_REVOKE_REASON_MAX_BYTES,
  fitsCaptureLimit,
  isCompletePairingCode,
  normalizePairingCode,
} from "@/lib/capture";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { z } from "zod";

/** The code a person types or pastes; it leaves the form as the letters the server compares. */
export function pairingCodeFormSchema(t: TranslateFn) {
  return z.object({
    code: z
      .string()
      .refine(isCompletePairingCode, t("Enter all eight letters of the code, like ABCD-EFGH"))
      .transform(normalizePairingCode),
  });
}

export type PairingCodeFormInput = z.input<ReturnType<typeof pairingCodeFormSchema>>;
export type PairingCodeFormOutput = z.output<ReturnType<typeof pairingCodeFormSchema>>;

/**
 * The name a computer is approved under. Blank keeps the machine's own name,
 * which is what the server does with it too.
 */
export function approvePairingFormSchema(t: TranslateFn) {
  return z.object({
    deviceName: z
      .string()
      .trim()
      .refine(
        (value) => fitsCaptureLimit(value, CAPTURE_DEVICE_NAME_MAX_BYTES),
        t("That name is too long. Keep it to {0} characters.", CAPTURE_DEVICE_NAME_MAX_BYTES),
      ),
  });
}

export type ApprovePairingFormValues = z.input<ReturnType<typeof approvePairingFormSchema>>;

export function revokeDeviceFormSchema(t: TranslateFn) {
  return z.object({
    reason: z
      .string()
      .trim()
      .refine(
        (value) => fitsCaptureLimit(value, CAPTURE_REVOKE_REASON_MAX_BYTES),
        t("That reason is too long. Keep it to {0} characters.", CAPTURE_REVOKE_REASON_MAX_BYTES),
      ),
  });
}

export type RevokeDeviceFormValues = z.input<ReturnType<typeof revokeDeviceFormSchema>>;
