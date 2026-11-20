import { GLAccountAutocompleteField } from "@/components/autocomplete-fields";
import { SettingsSection, SettingsSections } from "@/components/settings/settings-section";
import {
  journalPostingModeSchema,
  journalSourceEventSchema,
  revenueRecognitionPolicySchema,
  type AccountingControl,
} from "@/types/accounting-control";
import { useT } from "@trenova/shared/i18n/use-t";
import { useFormContext, useWatch } from "react-hook-form";

/**
 * The general ledger accounts journals post to when nothing more specific names one. Each
 * is required exactly when the server needs it: the core four once posting is automatic,
 * and cash and unapplied cash once customer payments post.
 */
export function AccountsTab() {
  const t = useT();
  const { control } = useFormContext<AccountingControl>();
  const postingMode = useWatch({ control, name: "journalPostingMode" });
  const events = useWatch({ control, name: "autoPostSourceEvents" });
  const revenuePolicy = useWatch({ control, name: "revenueRecognitionPolicy" });

  const automatic = postingMode === journalPostingModeSchema.enum.Automatic;
  const paymentsPost =
    automatic &&
    ((events ?? []).includes(journalSourceEventSchema.enum.CustomerPaymentPosted) ||
      revenuePolicy === revenueRecognitionPolicySchema.enum.OnCashReceipt);

  return (
    <SettingsSections>
      <SettingsSection
        title={t("Receivables and revenue")}
        description={t("Where invoices, customer payments, write-offs and sales tax post.")}
      >
        <GLAccountAutocompleteField
          control={control}
          name="defaultRevenueAccountId"
          label={t("Revenue account")}
          placeholder={t("Select revenue account")}
          description={t("Credited for revenue when an invoice posts.")}
          required={automatic}
        />
        <GLAccountAutocompleteField
          control={control}
          name="defaultArAccountId"
          label={t("Accounts receivable account")}
          placeholder={t("Select AR account")}
          description={t("Debited for what customers owe on posted invoices.")}
          required={automatic}
        />
        <GLAccountAutocompleteField
          control={control}
          name="defaultCashAccountId"
          label={t("Cash account")}
          placeholder={t("Select cash account")}
          description={t("Debited when a customer payment posts.")}
          required={paymentsPost}
        />
        <GLAccountAutocompleteField
          control={control}
          name="defaultUnappliedCashAccountId"
          label={t("Unapplied cash account")}
          placeholder={t("Select unapplied cash account")}
          description={t(
            "Holds the part of a customer payment not yet applied to an invoice until it is.",
          )}
          required={paymentsPost}
        />
        <GLAccountAutocompleteField
          control={control}
          name="defaultWriteOffAccountId"
          label={t("Write-off account")}
          placeholder={t("Select write-off account")}
          description={t("Debited when an approved write-off is booked.")}
        />
        <GLAccountAutocompleteField
          control={control}
          name="defaultTaxLiabilityAccountId"
          label={t("Tax liability account")}
          placeholder={t("Select tax liability account")}
          description={t("Credited for the tax collected on posted invoices.")}
        />
      </SettingsSection>

      <SettingsSection
        title={t("Payables and expense")}
        description={t("Where vendor bills and the payments that settle them post.")}
      >
        <GLAccountAutocompleteField
          control={control}
          name="defaultExpenseAccountId"
          label={t("Expense account")}
          placeholder={t("Select expense account")}
          description={t("Debited for expense when a vendor bill posts.")}
          required={automatic}
        />
        <GLAccountAutocompleteField
          control={control}
          name="defaultApAccountId"
          label={t("Accounts payable account")}
          placeholder={t("Select AP account")}
          description={t("Credited for what is owed to vendors on posted bills.")}
          required={automatic}
        />
      </SettingsSection>

      <SettingsSection
        title={t("Equity")}
        description={t("Where the year's result is carried when the books close.")}
      >
        <GLAccountAutocompleteField
          control={control}
          name="defaultRetainedEarningsAccountId"
          label={t("Retained earnings account")}
          placeholder={t("Select retained earnings account")}
          description={t("Receives net income when a fiscal year is closed.")}
        />
      </SettingsSection>
    </SettingsSections>
  );
}
