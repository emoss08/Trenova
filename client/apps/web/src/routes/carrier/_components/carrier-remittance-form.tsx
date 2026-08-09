import { UsStateAutocompleteField } from "@/components/autocomplete-fields";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import { carrierPaymentMethodChoices } from "@/lib/choices";
import type { Carrier } from "@trenova/shared/types/carrier";
import { useFormContext } from "react-hook-form";

export function CarrierRemittanceForm() {
  const { control } = useFormContext<Carrier>();

  return (
    <div className="space-y-6">
      <FormSection
        title="Payment Terms"
        description="How and when this carrier is paid for completed loads."
        className="border-b"
      >
        <FormGroup cols={2}>
          <FormControl>
            <SelectField
              control={control}
              rules={{ required: true }}
              name="paymentMethod"
              label="Payment Method"
              placeholder="Payment Method"
              description="Method used to remit payment to this carrier."
              options={carrierPaymentMethodChoices}
            />
          </FormControl>
          <FormControl>
            <NumberField
              control={control}
              rules={{ required: true }}
              name="paymentTermDays"
              label="Payment Term Days"
              placeholder="30"
              sideText="days"
              description="Number of days after invoice receipt that payment is due. 0-365."
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title="Remit-To Address"
        description="Where payments to this carrier are mailed when it differs from the primary address."
      >
        <FormGroup cols={2}>
          <FormControl cols="full">
            <InputField
              control={control}
              name="remitToName"
              label="Remit-To Name"
              placeholder="e.g., Swift Transportation Co. or factoring company"
              description="Payee name printed on checks. Use the factoring company name when payments are factored."
              maxLength={255}
            />
          </FormControl>
          <FormControl cols="full">
            <InputField
              control={control}
              name="remitAddressLine1"
              label="Address Line 1"
              placeholder="Street address"
              description="Street address payments are mailed to."
              maxLength={150}
            />
          </FormControl>
          <FormControl cols="full">
            <InputField
              control={control}
              name="remitAddressLine2"
              label="Address Line 2"
              placeholder="Suite, floor, building, etc."
              description="Additional remit-to address details such as suite number or PO box."
              maxLength={150}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="remitCity"
              label="City"
              placeholder="City"
              description="City for the remit-to address."
              maxLength={100}
            />
          </FormControl>
          <FormControl>
            <UsStateAutocompleteField
              control={control}
              name="remitStateId"
              label="State"
              placeholder="State"
              description="U.S. state for the remit-to address."
              clearable
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              name="remitPostalCode"
              label="Postal Code"
              placeholder="e.g., 90210"
              description="ZIP or ZIP+4 code for the remit-to address."
              maxLength={10}
            />
          </FormControl>
        </FormGroup>
      </FormSection>
    </div>
  );
}
