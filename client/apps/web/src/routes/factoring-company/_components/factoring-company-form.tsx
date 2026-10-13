import { useT } from "@trenova/shared/i18n/use-t";
import { UsStateAutocompleteField } from "@/components/autocomplete-fields";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import { statusChoices } from "@/lib/choices";
import { MAX_FACTOR_TEXT, type FactoringCompany } from "@/types/factoring-company";
import { useFormContext } from "react-hook-form";

export function FactoringCompanyForm() {
  const t = useT();
  const { control } = useFormContext<FactoringCompany>();

  return (
    <div className="flex flex-col gap-6">
      <FormSection
        title={t("General information")}
        description={t("How the factor is named on invoices and in customer billing profiles")}
      >
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              control={control}
              rules={{ required: true }}
              name="status"
              label={t("Status")}
              placeholder={t("Status")}
              description={t(
                "Only active factors can be assigned to customers. A factor customers are still assigned to cannot be made inactive.",
              )}
              options={statusChoices}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              rules={{ required: true }}
              name="code"
              label={t("Code")}
              placeholder={t("e.g., RTSFIN")}
              description={t("Short identifier, unique across your organization.")}
              maxLength={10}
            />
          </FormControl>
          <FormControl cols="full">
            <InputField
              control={control}
              rules={{ required: true }}
              name="name"
              label={t("Name")}
              placeholder={t("e.g., RTS Financial Service, Inc.")}
              description={t(
                "The factor's legal name. It is printed as the payee on every invoice assigned to it.",
              )}
              maxLength={255}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title={t("Remit-to address")}
        description={t("Where customers send payment for invoices assigned to this factor")}
      >
        <FormGroup cols={2}>
          <FormControl cols="full">
            <InputField
              control={control}
              rules={{ required: true }}
              name="addressLine1"
              label={t("Address line 1")}
              placeholder={t("e.g., PO Box 840267")}
              maxLength={150}
            />
          </FormControl>
          <FormControl cols="full">
            <InputField
              control={control}
              name="addressLine2"
              label={t("Address line 2")}
              placeholder={t("Suite, floor, lockbox, etc.")}
              maxLength={150}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              rules={{ required: true }}
              name="city"
              label={t("City")}
              placeholder={t("City")}
              maxLength={100}
            />
          </FormControl>
          <FormControl>
            <UsStateAutocompleteField
              control={control}
              name="stateId"
              label={t("State")}
              placeholder={t("State")}
            />
          </FormControl>
          <FormControl cols="full">
            <InputField
              control={control}
              rules={{ required: true }}
              name="postalCode"
              label={t("Postal code")}
              placeholder={t("e.g., 75284")}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title={t("Contact")}
        description={t("Who customers and your team reach at the factor")}
      >
        <FormGroup cols={2}>
          <FormControl>
            <InputField
              control={control}
              name="email"
              type="email"
              label={t("Email")}
              placeholder={t("e.g., payments@factor.com")}
              maxLength={255}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="phoneNumber"
              label={t("Phone")}
              placeholder={t("e.g., (800) 555-0142")}
              maxLength={30}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title={t("Invoice wording")}
        description={t("What every invoice assigned to this factor prints")}
      >
        <FormGroup cols={1}>
          <FormControl>
            <TextareaField
              control={control}
              name="remittanceInstructions"
              label={t("Remittance instructions")}
              placeholder={t("e.g., ACH: routing 111000614, account 8842091")}
              description={t(
                "Bank, lockbox or ACH details. They print under the factor's address wherever an invoice says where to pay.",
              )}
              maxLength={MAX_FACTOR_TEXT}
              rows={3}
            />
          </FormControl>
          <FormControl>
            <TextareaField
              control={control}
              name="noticeOfAssignment"
              label={t("Notice of assignment")}
              placeholder={t(
                "Leave blank to print the standard notice: this invoice has been assigned to, and is payable only to, the factor.",
              )}
              description={t(
                "Use the exact wording your factoring agreement requires, if it gives one.",
              )}
              maxLength={MAX_FACTOR_TEXT}
              rows={4}
            />
          </FormControl>
          <FormControl>
            <TextareaField
              control={control}
              name="notes"
              label={t("Notes")}
              placeholder={t("e.g., Recourse agreement, 90-day chargeback")}
              description={t("For your team only. Never printed.")}
              maxLength={MAX_FACTOR_TEXT}
              rows={3}
            />
          </FormControl>
        </FormGroup>
      </FormSection>
    </div>
  );
}
