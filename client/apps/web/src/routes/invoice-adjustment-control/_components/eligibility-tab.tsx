import { SelectField } from "@/components/fields/select-field";
import {
  SettingsNested,
  SettingsSection,
  SettingsSections,
} from "@/components/settings/settings-section";
import {
  adjustmentAccountingDatePolicyChoices,
  adjustmentEligibilityPolicyChoices,
  closedPeriodAdjustmentPolicyChoices,
} from "@/lib/choices";
import {
  closedPeriodAdjustmentPolicySchema,
  type InvoiceAdjustmentControl,
} from "@/types/invoice-adjustment-control";
import { defineLabels } from "@trenova/shared/i18n/labels";
import { useT } from "@trenova/shared/i18n/use-t";
import { useFormContext, useWatch } from "react-hook-form";
import { approvesNothing, closedPeriodPolicyApplies } from "../invoice-adjustment-control-submit";

const REASONS = defineLabels({
  approvesNothing: "Adjustments never need approval. Change that under Approval first.",
});

/** Which invoices can be adjusted, and which accounting period an adjustment books in. */
export function EligibilityTab() {
  const t = useT();
  const { control } = useFormContext<InvoiceAdjustmentControl>();
  const datePolicy = useWatch({ control, name: "adjustmentAccountingDatePolicy" });
  const approvalPolicy = useWatch({ control, name: "standardAdjustmentApprovalPolicy" });

  const closedPeriodOptions = closedPeriodAdjustmentPolicyChoices.map((choice) =>
    choice.value === closedPeriodAdjustmentPolicySchema.enum.PostInNextOpenPeriodWithApproval &&
    approvesNothing(approvalPolicy)
      ? { ...choice, disabled: true, description: REASONS.approvesNothing }
      : choice,
  );

  return (
    <SettingsSections>
      <SettingsSection
        title={t("Invoices that can be adjusted")}
        description={t(
          "Whether an invoice that has been paid or disputed can still be credited, rebilled or written off. Adjusting one always needs approval.",
        )}
      >
        <SelectField
          control={control}
          name="partiallyPaidInvoiceAdjustmentPolicy"
          label={t("Partly paid invoices")}
          description={t("Invoices with part of their balance paid.")}
          options={adjustmentEligibilityPolicyChoices}
          required
        />
        <SelectField
          control={control}
          name="paidInvoiceAdjustmentPolicy"
          label={t("Paid invoices")}
          description={t("Invoices paid in full.")}
          options={adjustmentEligibilityPolicyChoices}
          required
        />
        <SelectField
          control={control}
          name="disputedInvoiceAdjustmentPolicy"
          label={t("Disputed invoices")}
          description={t("Invoices the customer has disputed.")}
          options={adjustmentEligibilityPolicyChoices}
          required
        />
      </SettingsSection>

      <SettingsSection
        title={t("Accounting date")}
        description={t("Which accounting period an adjustment's journal entry lands in.")}
      >
        <SelectField
          control={control}
          name="adjustmentAccountingDatePolicy"
          label={t("Book adjustments on")}
          description={t(
            "The invoice's own date while its period is open, or always the next open period.",
          )}
          options={adjustmentAccountingDatePolicyChoices}
          required
        />
        <SettingsNested show={closedPeriodPolicyApplies(datePolicy)}>
          <SelectField
            control={control}
            name="closedPeriodAdjustmentPolicy"
            label={t("When the period is closed")}
            description={t("What happens to an adjustment whose invoice is in a closed period.")}
            options={closedPeriodOptions}
            required
          />
        </SettingsNested>
      </SettingsSection>
    </SettingsSections>
  );
}
