import {
  DEFAULT_DETENTION_ALERT_MINUTES,
  DETENTION_ALERT_MAX_MINUTES,
  DETENTION_ALERT_MIN_MINUTES,
  driverDigestCadenceSchema,
  type DashControl,
} from "@/types/dash-control";

/**
 * The record as it is saved: a setting the page hides because another one makes it
 * meaningless is sent the way the server requires it then, so a save is never refused for a
 * field the person cannot see.
 */
export function prepareDashControlForSubmit(values: DashControl): DashControl {
  const next = { ...values };

  if (!next.requireLoadAcknowledgment) {
    next.allowLoadRefusals = false;
  }

  if (!next.showLoadPay) {
    next.showPayEstimates = false;
  }

  if (!next.allowExpenseSubmission) {
    next.requireExpenseReceipt = false;
  }

  if (!next.sendCredentialReminders) {
    next.driverDigestCadence = driverDigestCadenceSchema.enum.Immediate;
  }

  const minutes = next.detentionAlertThresholdMinutes;
  if (
    !next.enableDetentionAlerts &&
    (minutes === null ||
      minutes < DETENTION_ALERT_MIN_MINUTES ||
      minutes > DETENTION_ALERT_MAX_MINUTES)
  ) {
    next.detentionAlertThresholdMinutes = DEFAULT_DETENTION_ALERT_MINUTES;
  }

  return next;
}
