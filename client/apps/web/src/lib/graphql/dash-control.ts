import {
  DashControlDocument,
  UpdateDashControlDocument,
} from "@trenova/graphql/generated/documents/driver-portal/operations";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { DEFAULT_DETENTION_ALERT_MINUTES, type DashControl } from "@/types/dash-control";

export async function fetchDashControl(options?: { signal?: AbortSignal }): Promise<DashControl> {
  const data = await requestGraphQL({
    document: DashControlDocument,
    operationName: "DashControl",
    signal: options?.signal,
  });
  return data.dashControl;
}

/** Saves the record as the form holds it, after `prepareDashControlForSubmit`. */
export async function updateDashControl(record: DashControl): Promise<DashControl> {
  const data = await requestGraphQL({
    document: UpdateDashControlDocument,
    operationName: "UpdateDashControl",
    variables: {
      input: {
        version: record.version,
        requireLoadAcknowledgment: record.requireLoadAcknowledgment,
        allowLoadRefusals: record.allowLoadRefusals,
        allowStopActions: record.allowStopActions,
        allowLoadDocumentUpload: record.allowLoadDocumentUpload,
        allowLoadComments: record.allowLoadComments,
        showLoadPay: record.showLoadPay,
        showPayEstimates: record.showPayEstimates,
        allowExpenseSubmission: record.allowExpenseSubmission,
        requireExpenseReceipt: record.requireExpenseReceipt,
        allowSettlementDisputes: record.allowSettlementDisputes,
        allowProfileDocumentUpload: record.allowProfileDocumentUpload,
        allowContactInfoEdit: record.allowContactInfoEdit,
        allowPtoRequests: record.allowPtoRequests,
        sendCredentialReminders: record.sendCredentialReminders,
        requireContactChangeApproval: record.requireContactChangeApproval,
        driverDigestCadence: record.driverDigestCadence,
        driverDigestWeekday: record.driverDigestWeekday,
        enableDetentionAlerts: record.enableDetentionAlerts,
        // Never empty once prepared: required while alerts are on, filled in while off.
        detentionAlertThresholdMinutes:
          record.detentionAlertThresholdMinutes ?? DEFAULT_DETENTION_ALERT_MINUTES,
      },
    },
  });
  return data.updateDashControl;
}
