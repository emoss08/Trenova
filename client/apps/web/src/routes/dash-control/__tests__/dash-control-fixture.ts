import type { DashControl } from "@/types/dash-control";

/**
 * A saved Dash control as the `dashControl` query sends it (the Go entity through the
 * GraphQL type): every setting that opens rows beneath it switched off, a round-up day of
 * Monday and the two-hour threshold a new organization starts with.
 */
export const savedDashControl: DashControl = {
  id: "dashc_01",
  version: 4,
  requireLoadAcknowledgment: false,
  allowLoadRefusals: false,
  allowStopActions: true,
  allowLoadDocumentUpload: true,
  allowLoadComments: true,
  enableDetentionAlerts: false,
  detentionAlertThresholdMinutes: 120,
  showLoadPay: false,
  showPayEstimates: false,
  allowExpenseSubmission: false,
  requireExpenseReceipt: false,
  allowSettlementDisputes: true,
  allowProfileDocumentUpload: true,
  allowContactInfoEdit: false,
  requireContactChangeApproval: false,
  allowPtoRequests: true,
  sendCredentialReminders: false,
  driverDigestCadence: "Immediate",
  driverDigestWeekday: 1,
};
