import { SettingsSection, SettingsSections } from "@/components/settings/settings-section";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { useT } from "@trenova/shared/i18n/use-t";
import {
  AGENT_EVAL_CASE_RETENTION_LIMIT_DAYS,
  AI_AUDIT_RETENTION_MIN_DAYS,
  AI_CORRECTION_RETENTION_MIN_DAYS,
  AI_FEEDBACK_RETENTION_MIN_DAYS,
  AUDIT_ACCESS_RETENTION_MIN_DAYS,
  AUDIT_FINANCIAL_RETENTION_MIN_DAYS,
  AUDIT_OPERATIONAL_RETENTION_MIN_DAYS,
  AUDIT_RETENTION_MAX_DAYS,
  AUDIT_SECURITY_RETENTION_MIN_DAYS,
  DRIVER_QUALIFICATION_RETENTION_MIN_DAYS,
  EDI_RETENTION_LIMIT_DAYS,
} from "../data-retention-form";
import { RetentionLimit, RetentionPeriodField } from "./retention-policy-row";

/**
 * Every kind of data the nightly purges remove, grouped by where it comes from: the audit
 * trail, raw EDI traffic, driver qualification files (flagged, never deleted) and the records AI
 * and agents leave. Each row says what is removed and
 * the date the next purge reaches back to. Someone without update access reads the same page
 * with nothing to change.
 */
export function RetentionTab({ now, readOnly }: { now: number; readOnly: boolean }) {
  const t = useT();
  const auditDeleted = (date: string) =>
    t("Entries from before {0} are permanently deleted in the next nightly purge.", date);

  return (
    <SettingsSections>
      {readOnly && (
        <Alert variant="info" size="sm" className="mb-5">
          <AlertDescription>
            {t("You can view these settings but not change them.")}
          </AlertDescription>
        </Alert>
      )}

      <SettingsSection
        title={t("Audit trail")}
        description={t(
          "Who did what across the organization. Entries older than their period are permanently deleted each night, unless a legal hold covers them, and cannot be recovered.",
        )}
      >
        <RetentionPeriodField
          name="auditAccessRetentionDays"
          label={t("Reads and downloads")}
          description={t("Who viewed, downloaded or exported a record. At least 90 days.")}
          consequence={auditDeleted}
          min={AUDIT_ACCESS_RETENTION_MIN_DAYS}
          max={AUDIT_RETENTION_MAX_DAYS}
          now={now}
          disabled={readOnly}
        />
        <RetentionPeriodField
          name="auditOperationalRetentionDays"
          label={t("Operational changes")}
          description={t(
            "Changes to shipments, equipment, people and settings. At least 730 days.",
          )}
          consequence={auditDeleted}
          min={AUDIT_OPERATIONAL_RETENTION_MIN_DAYS}
          max={AUDIT_RETENTION_MAX_DAYS}
          now={now}
          disabled={readOnly}
        />
        <RetentionPeriodField
          name="auditFinancialRetentionDays"
          label={t("Financial changes")}
          description={t(
            "Changes to invoices, rates, accounting and settlements. At least 2,555 days, seven years.",
          )}
          consequence={auditDeleted}
          min={AUDIT_FINANCIAL_RETENTION_MIN_DAYS}
          max={AUDIT_RETENTION_MAX_DAYS}
          now={now}
          disabled={readOnly}
        />
        <RetentionPeriodField
          name="auditSecurityRetentionDays"
          label={t("Security entries")}
          description={t(
            "Access changes, refused requests and other security events. At least 730 days.",
          )}
          consequence={auditDeleted}
          min={AUDIT_SECURITY_RETENTION_MIN_DAYS}
          max={AUDIT_RETENTION_MAX_DAYS}
          now={now}
          disabled={readOnly}
        />
      </SettingsSection>

      <SettingsSection
        title={t("EDI")}
        description={t(
          "The raw contents of EDI traffic. A purge blanks the contents for good but keeps each file and message record and its details.",
        )}
      >
        <RetentionLimit
          name="ediInboundFileRetentionPeriod"
          label={t("Blank old inbound files")}
          description={t(
            "Blanks the raw contents of inbound EDI files once they pass a set age. Off keeps every file whole.",
          )}
          periodLabel={t("Blank files after")}
          periodDescription={t("How old a received file is when its contents are blanked.")}
          consequence={(date) =>
            t(
              "Files received before {0} are blanked in the next nightly purge. This cannot be undone.",
              date,
            )
          }
          fallbackDays={EDI_RETENTION_LIMIT_DAYS}
          now={now}
          disabled={readOnly}
        />
        <RetentionLimit
          name="ediMessageRetentionPeriod"
          label={t("Blank old message payloads")}
          description={t(
            "Blanks the raw X12 and payload snapshots of sent and received messages once they pass a set age. A blanked message can no longer be replayed. Off keeps every payload.",
          )}
          periodLabel={t("Blank payloads after")}
          periodDescription={t("How old a message is when its payload is blanked.")}
          consequence={(date) =>
            t(
              "Payloads from before {0} are blanked in the next nightly purge, and those messages can no longer be replayed.",
              date,
            )
          }
          fallbackDays={EDI_RETENTION_LIMIT_DAYS}
          now={now}
          disabled={readOnly}
        />
      </SettingsSection>

      <SettingsSection
        title={t("Driver files")}
        description={t(
          "How long a driver's qualification file is held after the driver leaves. Old files are only flagged for review; nothing here ever deletes one.",
        )}
      >
        <RetentionPeriodField
          name="driverQualificationRetentionPeriod"
          label={t("Driver qualification files")}
          description={t(
            "Files of drivers who have left are flagged for review once they pass this age. They are never deleted. At least 1,095 days, three years after employment ends, the federal minimum under 49 CFR 391.51.",
          )}
          consequence={(date) =>
            t(
              "Files of drivers who left before {0} are flagged for review in the next nightly check.",
              date,
            )
          }
          min={DRIVER_QUALIFICATION_RETENTION_MIN_DAYS}
          now={now}
          disabled={readOnly}
        />
      </SettingsSection>

      <SettingsSection
        title={t("AI and agents")}
        description={t(
          "What AI and agents record as they work. Records older than their period are permanently deleted each night and cannot be recovered.",
        )}
      >
        <RetentionPeriodField
          name="aiAuditRetentionPeriod"
          label={t("AI audit trail")}
          description={t(
            "Signed events recording each AI decision. At least 365 days; seven years, 2,555 days, is what most audits of AI decisions ask for.",
          )}
          consequence={(date) =>
            t(
              "Events from before {0} are permanently deleted in the next nightly purge, up to the last point the trail was sealed, so what is left still verifies.",
              date,
            )
          }
          min={AI_AUDIT_RETENTION_MIN_DAYS}
          now={now}
          disabled={readOnly}
        />
        <RetentionPeriodField
          name="aiFeedbackRetentionPeriod"
          label={t("AI feedback")}
          description={t(
            "Ratings of AI answers, with the question and answer each person saw. At least 30 days.",
          )}
          consequence={(date) =>
            t("Feedback from before {0} is permanently deleted in the next nightly purge.", date)
          }
          min={AI_FEEDBACK_RETENTION_MIN_DAYS}
          now={now}
          disabled={readOnly}
        />
        <RetentionPeriodField
          name="aiCorrectionRetentionPeriod"
          label={t("AI corrections")}
          description={t(
            "What was read from a document beside what a person confirmed, used to measure reading accuracy. At least 30 days.",
          )}
          consequence={(date) =>
            t(
              "Corrections from before {0} are permanently deleted in the next nightly purge.",
              date,
            )
          }
          min={AI_CORRECTION_RETENTION_MIN_DAYS}
          now={now}
          disabled={readOnly}
        />
        <RetentionLimit
          name="agentEvalCaseRetentionPeriod"
          label={t("Delete old evaluation cases")}
          description={t(
            "Deletes captured agent cases and their replays, which hold a redacted copy of what the agent was given, once they pass a set age. Off keeps them until they expire or their conversation is deleted.",
          )}
          periodLabel={t("Delete cases after")}
          periodDescription={t("How old a captured case is when it and its replays are deleted.")}
          consequence={(date) =>
            t(
              "Cases captured before {0} and their replays are permanently deleted in the next nightly purge.",
              date,
            )
          }
          fallbackDays={AGENT_EVAL_CASE_RETENTION_LIMIT_DAYS}
          now={now}
          disabled={readOnly}
        />
      </SettingsSection>
    </SettingsSections>
  );
}
