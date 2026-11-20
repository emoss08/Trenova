import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import {
  SettingsNested,
  SettingsSection,
  SettingsSections,
} from "@/components/settings/settings-section";
import {
  closedPeriodPostingPolicyChoices,
  lockedPeriodPostingPolicyChoices,
  periodCloseModeChoices,
  reconciliationModeChoices,
} from "@/lib/choices";
import {
  periodCloseModeSchema,
  reconciliationModeSchema,
  type AccountingControl,
} from "@/types/accounting-control";
import { useT } from "@trenova/shared/i18n/use-t";
import { useFormContext, useWatch } from "react-hook-form";

/** How accounting periods close, what may still post into them, and how reconciliation gates it. */
export function CloseTab() {
  const t = useT();
  const { control } = useFormContext<AccountingControl>();
  const closeMode = useWatch({ control, name: "periodCloseMode" });
  const reconciliationMode = useWatch({ control, name: "reconciliationMode" });
  const reconciles = reconciliationMode !== reconciliationModeSchema.enum.Disabled;

  return (
    <SettingsSections>
      <SettingsSection
        title={t("Period close")}
        description={t(
          "How accounting periods close, and what happens to postings dated in a locked or closed period.",
        )}
      >
        <SelectField
          control={control}
          name="periodCloseMode"
          label={t("Period close")}
          description={t("Whether someone closes each period or a scheduled job does.")}
          options={periodCloseModeChoices}
          required
        />
        <SettingsNested show={closeMode === periodCloseModeSchema.enum.ManualOnly}>
          <SwitchField
            control={control}
            name="requirePeriodCloseApproval"
            label={t("Require approval to close a period")}
            description={t("A period someone closes waits for approval before it is final.")}
          />
        </SettingsNested>
        <SelectField
          control={control}
          name="lockedPeriodPostingPolicy"
          label={t("Posting into a locked period")}
          description={t("What may still post into a period that is locked.")}
          options={lockedPeriodPostingPolicyChoices}
          required
        />
        <SelectField
          control={control}
          name="closedPeriodPostingPolicy"
          label={t("Posting into a closed period")}
          description={t(
            "Whether the period has to be reopened first or the posting moves to the next open period.",
          )}
          options={closedPeriodPostingPolicyChoices}
          required
        />
      </SettingsSection>

      <SettingsSection
        title={t("Reconciliation")}
        description={t(
          "What a discrepancy between the ledger and its sources does to posting and to closing a period.",
        )}
      >
        <SelectField
          control={control}
          name="reconciliationMode"
          label={t("Reconciliation")}
          description={t("Whether discrepancies are ignored, raised as warnings or stop posting.")}
          options={reconciliationModeChoices}
          required
        />
        <SettingsNested show={reconciles}>
          <NumberField
            control={control}
            name="reconciliationToleranceAmount"
            label={t("Tolerance")}
            description={t(
              "The largest discrepancy let through before reconciliation responds. Must be above zero.",
            )}
            placeholder="0.00"
            decimalScale={2}
            min={0}
            required={reconciles}
          />
          <SwitchField
            control={control}
            name="requireReconciliationToClose"
            label={t("Require reconciliation to close")}
            description={t("A period cannot close while a discrepancy is still open.")}
          />
          <SwitchField
            control={control}
            name="notifyOnReconciliationException"
            label={t("Notify on discrepancies")}
            description={t("Sends a notification each time a discrepancy is recorded.")}
          />
        </SettingsNested>
      </SettingsSection>
    </SettingsSections>
  );
}
