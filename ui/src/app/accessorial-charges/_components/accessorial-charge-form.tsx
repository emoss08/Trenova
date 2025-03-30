import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup } from "@/components/ui/form";
import { accessorialChargeMethodChoices, statusChoices } from "@/lib/choices";
import { type AccessorialChargeSchema } from "@/lib/schemas/accessorial-charge-schema";
import { useFormContext } from "react-hook-form";

export function AccessorialChargeForm() {
  const { control } = useFormContext<AccessorialChargeSchema>();

  return (
    <FormGroup cols={2}>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="status"
          label="Status"
          placeholder="Status"
          description="Current processing status of this accessorial charge (active, pending approval, etc.)"
          options={statusChoices}
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          rules={{ required: true }}
          name="code"
          label="Code"
          placeholder="Code"
          description="Standard industry or company-specific code identifying this accessorial service (e.g., LUM for lumper fee)"
        />
      </FormControl>
      <FormControl cols="full">
        <TextareaField
          control={control}
          rules={{ required: true }}
          name="description"
          label="Description"
          placeholder="Description"
          description="Detailed explanation of the accessorial service provided, including any special conditions or requirements for FMCSA compliance"
        />
      </FormControl>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="method"
          label="Method"
          placeholder="Method"
          description="Calculation method for this charge (flat rate, per mile, percentage of linehaul, etc.)"
          options={accessorialChargeMethodChoices}
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          rules={{ required: true }}
          name="unit"
          label="Units"
          type="number"
          placeholder="Unit"
          description="Quantity of units this charge applies to (number of pallets, hours of detention, etc.)"
          sideText="unit(s)"
        />
      </FormControl>
      <FormControl cols="full">
        <InputField
          control={control}
          rules={{ required: true }}
          name="amount"
          type="number"
          label="Amount"
          placeholder="Amount"
          description="Dollar value per unit for this accessorial service, used to calculate total charges for billing and settlement"
        />
      </FormControl>
    </FormGroup>
  );
}
