import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import { carrierTaxIdTypeChoices } from "@/lib/choices";
import type { Carrier } from "@trenova/shared/types/carrier";
import { useFormContext } from "react-hook-form";

export function CarrierTaxForm() {
  const { control } = useFormContext<Carrier>();

  return (
    <div className="space-y-6">
      <FormSection
        title="Tax Information"
        description="Tax identification and 1099 reporting details for this carrier."
      >
        <FormGroup cols={2}>
          <FormControl>
            <InputField
              control={control}
              name="taxId"
              label="Tax ID"
              placeholder="e.g., 12-3456789"
              description="EIN or SSN used for tax reporting. Selecting a tax ID type is required when this is set."
              maxLength={20}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              name="taxIdType"
              label="Tax ID Type"
              placeholder="Tax ID Type"
              description="Whether the tax ID is an employer identification number or a social security number."
              options={carrierTaxIdTypeChoices}
              isClearable
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="w9OnFile"
              label="W-9 On File"
              description="A completed W-9 form has been received from this carrier."
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="is1099Eligible"
              label="1099 Eligible"
              description="Payments to this carrier should be included in 1099 reporting."
            />
          </FormControl>
        </FormGroup>
      </FormSection>
    </div>
  );
}
