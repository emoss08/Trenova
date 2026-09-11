import { useT } from "@trenova/shared/i18n/use-t";
import { UsStateAutocompleteField } from "@/components/autocomplete-fields";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import { carrierPaymentMethodChoices } from "@/lib/choices";
import type { Carrier } from "@trenova/shared/types/carrier";
import { useFormContext } from "react-hook-form";

export function CarrierRemittanceForm() {
  const t = useT();

  const { control } = useFormContext<Carrier>();

  return (
    <div className="space-y-6">
      <FormSection
        title={t("Payment Terms")}
        description={t("How and when this carrier is paid for completed loads.")}
        className="border-b"
      >
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              control={control}
              rules={{ required: true }}
              name="paymentMethod"
              label={t("Payment Method")}
              placeholder={t("Payment Method")}
              description={t("Method used to remit payment to this carrier.")}
              options={carrierPaymentMethodChoices}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              rules={{ required: true }}
              name="paymentTermDays"
              label={t("Payment Term Days")}
              placeholder="30"
              sideText="days"
              description={t("Number of days after invoice receipt that payment is due. 0-365.")}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title={t("Remit-To Address")}
        description={t("Where payments to this carrier are mailed when it differs from the primary address.")}
      >
        <FormGroup cols={2}>
          <FormControl cols="full">
            <InputField
              control={control}
              name="remitToName"
              label={t("Remit-To Name")}
              placeholder={t("e.g., Swift Transportation Co. or factoring company")}
              description={t("Payee name printed on checks. Use the factoring company name when payments are factored.")}
              maxLength={255}
            />
          </FormControl>
          <FormControl cols="full">
            <InputField
              control={control}
              name="remitAddressLine1"
              label={t("Address Line 1")}
              placeholder={t("Street address")}
              description={t("Street address payments are mailed to.")}
              maxLength={150}
            />
          </FormControl>
          <FormControl cols="full">
            <InputField
              control={control}
              name="remitAddressLine2"
              label={t("Address Line 2")}
              placeholder={t("Suite, floor, building, etc.")}
              description={t("Additional remit-to address details such as suite number or PO box.")}
              maxLength={150}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="remitCity"
              label={t("City")}
              placeholder={t("City")}
              description={t("City for the remit-to address.")}
              maxLength={100}
            />
          </FormControl>
          <FormControl>
            <UsStateAutocompleteField
              control={control}
              name="remitStateId"
              label={t("State")}
              placeholder={t("State")}
              description={t("U.S. state for the remit-to address.")}
              clearable
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="remitPostalCode"
              label={t("Postal Code")}
              placeholder={t("e.g., 90210")}
              description={t("ZIP or ZIP+4 code for the remit-to address.")}
              maxLength={10}
            />
          </FormControl>
        </FormGroup>
      </FormSection>
    </div>
  );
}
