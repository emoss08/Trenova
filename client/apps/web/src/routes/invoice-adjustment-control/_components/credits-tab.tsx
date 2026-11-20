import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import {
  SettingsNested,
  SettingsSection,
  SettingsSections,
} from "@/components/settings/settings-section";
import {
  customerCreditBalancePolicyChoices,
  overCreditPolicyChoices,
  replacementInvoiceReviewPolicyChoices,
  supersededInvoiceVisibilityPolicyChoices,
} from "@/lib/choices";
import {
  customerCreditBalancePolicySchema,
  replacementInvoiceReviewPolicySchema,
  type InvoiceAdjustmentControl,
} from "@/types/invoice-adjustment-control";
import { useT } from "@trenova/shared/i18n/use-t";
import { useFormContext, useWatch } from "react-hook-form";

/** Whether an adjustment may leave the customer holding credit, and how rebills are checked. */
export function CreditsTab() {
  const t = useT();
  const { control } = useFormContext<InvoiceAdjustmentControl>();
  const creditPolicy = useWatch({ control, name: "customerCreditBalancePolicy" });
  const reviewPolicy = useWatch({ control, name: "replacementInvoiceReviewPolicy" });
  const reviewsChanges =
    reviewPolicy === replacementInvoiceReviewPolicySchema.enum.RequireReviewWhenEconomicTermsChange;

  return (
    <SettingsSections>
      <SettingsSection
        title={t("Customer credit")}
        description={t(
          "Whether an adjustment may leave credit on the customer's account that is not applied to an invoice.",
        )}
      >
        <SelectField
          control={control}
          name="customerCreditBalancePolicy"
          label={t("Unapplied credit")}
          description={t("Whether an adjustment may leave the customer with credit to use later.")}
          options={customerCreditBalancePolicyChoices}
          required
        />
        <SettingsNested
          show={creditPolicy === customerCreditBalancePolicySchema.enum.AllowUnappliedCredit}
        >
          <SelectField
            control={control}
            name="overCreditPolicy"
            label={t("Credit beyond what is owed")}
            description={t(
              "An adjustment that credits more than is still owed because the invoice has been paid.",
            )}
            options={overCreditPolicyChoices}
            required
          />
        </SettingsNested>
      </SettingsSection>

      <SettingsSection
        title={t("Rebills")}
        description={t(
          "How a replacement invoice is checked when an invoice is credited and billed again.",
        )}
      >
        <SelectField
          control={control}
          name="replacementInvoiceReviewPolicy"
          label={t("Review replacement invoices")}
          description={t("When a replacement invoice waits for review.")}
          options={replacementInvoiceReviewPolicyChoices}
          required
        />
        <SettingsNested show={reviewsChanges}>
          <NumberField
            control={control}
            name="rerateVarianceTolerancePercent"
            valueType="string"
            label={t("Rerate tolerance")}
            description={t(
              "A rerated invoice that differs from the original by more than this is reviewed.",
            )}
            placeholder="0"
            sideText="%"
            decimalScale={2}
            min={0}
            max={999.99}
            required={reviewsChanges}
          />
        </SettingsNested>
        <SelectField
          control={control}
          name="supersededInvoiceVisibilityPolicy"
          label={t("Replaced invoices")}
          description={t(
            "Whether customers can still get an invoice a rebill replaced: by email, from an emailed link and on their statement. Your team always sees it.",
          )}
          options={supersededInvoiceVisibilityPolicyChoices}
          required
        />
      </SettingsSection>
    </SettingsSections>
  );
}
