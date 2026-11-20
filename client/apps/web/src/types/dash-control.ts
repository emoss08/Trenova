import { translate } from "@trenova/shared/i18n/runtime";
import { z } from "zod";

export const driverDigestCadenceSchema = z.enum(["Immediate", "Daily", "Weekly"]);
export type DriverDigestCadence = z.infer<typeof driverDigestCadenceSchema>;

/** The server's bounds on the detention alert threshold, in minutes: 15 minutes to a day. */
export const DETENTION_ALERT_MIN_MINUTES = 15;
export const DETENTION_ALERT_MAX_MINUTES = 1440;

/** The threshold a new organization starts with: two hours, the usual free time at a stop. */
export const DEFAULT_DETENTION_ALERT_MINUTES = 120;

/**
 * An organization's Dash settings, as the `dashControl` query sends them and the form edits
 * them. A setting that only means something while another is on is not refused here: the
 * page hides it then, and the save sends it the way the server requires
 * (`dash-control-submit.ts`).
 */
export const dashControlSchema = z
  .object({
    id: z.string(),
    version: z.number().int().min(0),

    requireLoadAcknowledgment: z.boolean(),
    allowLoadRefusals: z.boolean(),
    allowStopActions: z.boolean(),
    allowLoadDocumentUpload: z.boolean(),
    allowLoadComments: z.boolean(),
    enableDetentionAlerts: z.boolean(),
    detentionAlertThresholdMinutes: z.number().int().nullable(),

    showLoadPay: z.boolean(),
    showPayEstimates: z.boolean(),
    allowExpenseSubmission: z.boolean(),
    requireExpenseReceipt: z.boolean(),
    allowSettlementDisputes: z.boolean(),

    allowProfileDocumentUpload: z.boolean(),
    allowContactInfoEdit: z.boolean(),
    requireContactChangeApproval: z.boolean(),
    allowPtoRequests: z.boolean(),

    sendCredentialReminders: z.boolean(),
    driverDigestCadence: driverDigestCadenceSchema,
    driverDigestWeekday: z
      .number({ error: () => translate("Choose the day the weekly round-up goes out") })
      .int()
      .min(0)
      .max(6),
  })
  .superRefine((values, ctx) => {
    if (!values.enableDetentionAlerts) return;
    const minutes = values.detentionAlertThresholdMinutes;
    if (minutes === null) {
      ctx.addIssue({
        code: "custom",
        path: ["detentionAlertThresholdMinutes"],
        message: translate("Enter how long a stop runs before dispatch is alerted"),
      });
      return;
    }
    if (minutes < DETENTION_ALERT_MIN_MINUTES || minutes > DETENTION_ALERT_MAX_MINUTES) {
      ctx.addIssue({
        code: "custom",
        path: ["detentionAlertThresholdMinutes"],
        message: translate("The threshold must be between 15 minutes and 24 hours"),
      });
    }
  });

export type DashControl = z.infer<typeof dashControlSchema>;
