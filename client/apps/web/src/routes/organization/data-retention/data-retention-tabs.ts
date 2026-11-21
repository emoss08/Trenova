import type { SettingsTabSpec } from "@/components/settings/settings-tab";
import { defineLabels } from "@trenova/shared/i18n/labels";
import type { DataRetentionFormValues } from "./data-retention-form";

export type DataRetentionTab = "retention";

export type DataRetentionField = keyof DataRetentionFormValues & string;

export type DataRetentionTabSpec = SettingsTabSpec<DataRetentionField> & {
  value: DataRetentionTab;
};

const TAB_LABELS = defineLabels({
  retention: "Retention",
});

/** The form fields each tab holds, so a refused save marks and opens the tab at fault. */
const TAB_FIELDS: Record<DataRetentionTab, readonly DataRetentionField[]> = {
  retention: [
    "auditAccessRetentionDays",
    "auditOperationalRetentionDays",
    "auditFinancialRetentionDays",
    "auditSecurityRetentionDays",
    "ediInboundFileRetentionPeriod",
    "ediMessageRetentionPeriod",
    "driverQualificationRetentionPeriod",
    "aiAuditRetentionPeriod",
    "aiFeedbackRetentionPeriod",
    "aiCorrectionRetentionPeriod",
    "agentEvalCaseRetentionPeriod",
  ],
};

const TAB_ORDER: readonly DataRetentionTab[] = ["retention"];

/**
 * The page's tabs. It has one job, deciding how long each kind of record is kept before the
 * nightly purges remove it, so it has one tab and draws no tab row; every organization sees it.
 */
export function dataRetentionTabs(): DataRetentionTabSpec[] {
  return TAB_ORDER.map((tab) => ({ value: tab, label: TAB_LABELS[tab], fields: TAB_FIELDS[tab] }));
}

/** The tabs this organization sees, for the form and its loading state alike. */
export function useDataRetentionTabs(): DataRetentionTabSpec[] {
  return dataRetentionTabs();
}
