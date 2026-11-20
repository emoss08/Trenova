import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import {
  SettingsNested,
  SettingsSection,
  SettingsSections,
} from "@/components/settings/settings-section";
import {
  approvalPolicyChoices,
  requirementPolicyChoices,
  writeOffApprovalPolicyChoices,
} from "@/lib/choices";
import {
  approvalPolicySchema,
  writeOffApprovalPolicySchema,
  type InvoiceAdjustmentControl,
} from "@/types/invoice-adjustment-control";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { useT } from "@trenova/shared/i18n/use-t";
import { useFormContext, useWatch } from "react-hook-form";
import { nextPeriodNeedsApproval } from "../invoice-adjustment-control-submit";

const REASONS = defineLabels({
  nextPeriod:
    "Adjustments to a closed period post in the next open period with approval. Change that under Eligibility first.",
});

/** What an adjustment must carry, and who signs off on adjustments and write-offs. */
export function ApprovalTab() {
  const t = useT();
  const { control } = useFormContext<InvoiceAdjustmentControl>();
  const approvalPolicy = useWatch({ control, name: "standardAdjustmentApprovalPolicy" });
  const writeOffPolicy = useWatch({ control, name: "writeOffApprovalPolicy" });
  const datePolicy = useWatch({ control, name: "adjustmentAccountingDatePolicy" });
  const closedPeriodPolicy = useWatch({ control, name: "closedPeriodAdjustmentPolicy" });

  const approvalAbove = approvalPolicy === approvalPolicySchema.enum.AmountThreshold;
  const writeOffAbove =
    writeOffPolicy === writeOffApprovalPolicySchema.enum.RequireApprovalAboveThreshold;
  const mustApprove = nextPeriodNeedsApproval({ datePolicy, closedPeriodPolicy });
  const approvalOptions = approvalPolicyChoices.map((choice) =>
    mustApprove && choice.value === approvalPolicySchema.enum.None
      ? { ...choice, disabled: true, description: REASONS.nextPeriod }
      : choice,
  );

  return (
    <SettingsSections>
      <SettingsSection
        title={t("Adjustments")}
        description={t(
          "What a credit, rebill or credit memo must carry, and when it waits for approval before it posts.",
        )}
      >
        <SelectField
          control={control}
          name="adjustmentReasonRequirement"
          label={t("Reason")}
          description={t("Whether every adjustment must say why it was made.")}
          options={requirementPolicyChoices}
          required
        />
        <SelectField
          control={control}
          name="standardAdjustmentApprovalPolicy"
          label={t("Approval")}
          description={t("Which adjustments and credit memos wait for someone to approve them.")}
          options={approvalOptions}
          required
        />
        <SettingsNested show={approvalAbove}>
          <NumberField
            control={control}
            name="standardAdjustmentApprovalThreshold"
            valueType="string"
            label={t("Approval amount")}
            description={t("Adjustments of this amount or more wait for approval.")}
            placeholder="0.00"
            decimalScale={2}
            min={0}
            required={approvalAbove}
          />
        </SettingsNested>
      </SettingsSection>

      <SettingsSection
        title={t("Write-offs")}
        description={t(
          "Writing off a balance the customer will not pay, including a short payment.",
        )}
      >
        <SelectField
          control={control}
          name="writeOffApprovalPolicy"
          label={t("Write-offs")}
          description={t("Whether a balance can be written off, and when that needs approval.")}
          options={writeOffApprovalPolicyChoices}
          required
        />
        <SettingsNested show={writeOffAbove}>
          <NumberField
            control={control}
            name="writeOffApprovalThreshold"
            valueType="string"
            label={t("Approval amount")}
            description={t("Write-offs of this amount or more wait for approval.")}
            placeholder="0.00"
            decimalScale={2}
            min={0}
            required={writeOffAbove}
          />
        </SettingsNested>
      </SettingsSection>

      <SettingsSection
        title={t("Approvers")}
        description={t("Who may approve what someone else has asked for.")}
      >
        <SwitchField
          control={control}
          name="requireSeparateApprover"
          label={t("Require a separate approver")}
          description={t(
            "The person who submits an adjustment or credit memo cannot approve or post it themselves.",
          )}
        />
      </SettingsSection>
    </SettingsSections>
  );
}
