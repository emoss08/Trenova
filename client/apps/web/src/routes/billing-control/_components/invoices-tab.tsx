import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { SettingsSection, SettingsSections } from "@/components/settings/settings-section";
import { paymentTermChoices } from "@/lib/choices";
import type { BillingControl } from "@/types/billing-control";
import { useT } from "@trenova/shared/i18n/use-t";
import { useFormContext } from "react-hook-form";

/** What an invoice says when the customer's billing profile does not say otherwise. */
export function InvoicesTab() {
  const t = useT();
  const { control } = useFormContext<BillingControl>();

  return (
    <SettingsSections>
      <SettingsSection
        title={t("Terms")}
        description={t(
          "The terms an invoice carries when the customer's billing profile sets none of its own.",
        )}
      >
        <SelectField
          control={control}
          name="defaultPaymentTerm"
          label={t("Payment term")}
          description={t("How long a customer has to pay an invoice.")}
          options={paymentTermChoices}
          required
        />
        <TextareaField
          control={control}
          name="defaultInvoiceTerms"
          label={t("Invoice terms")}
          description={t("The payment, billing and remittance terms printed on each invoice.")}
          placeholder={t("Payment, billing, and remittance terms")}
        />
      </SettingsSection>

      <SettingsSection
        title={t("On the invoice")}
        description={t("What every invoice shows the customer.")}
      >
        <SwitchField
          control={control}
          name="showDueDateOnInvoice"
          label={t("Show the due date")}
          description={t("Prints the date payment is due.")}
        />
        <SwitchField
          control={control}
          name="showBalanceDueOnInvoice"
          label={t("Show the balance due")}
          description={t("Prints what is still owed on the invoice.")}
        />
        <TextareaField
          control={control}
          name="defaultInvoiceFooter"
          label={t("Footer")}
          description={t(
            "Text printed at the foot of an invoice whose customer has no footer of its own.",
          )}
          placeholder={t("Footer content displayed on invoices")}
        />
      </SettingsSection>
    </SettingsSections>
  );
}
