import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup } from "@/components/ui/form";
import { accessorialChargeMethodChoices, rateUnitChoices, statusChoices } from "@/lib/choices";
import type { AccessorialCharge } from "@/types/accessorial-charge";
import { useEffect } from "react";
import { useFormContext, useWatch } from "react-hook-form";

const amountDescriptions: Record<AccessorialCharge["method"], string> = {
  Flat: "The fixed dollar amount charged",
  PerUnit: "The rate per unit (e.g., per hour, per mile)",
  Percentage: "The percentage applied to the linehaul rate",
};

export function AccessorialChargeForm() {
  const { control, setValue } = useFormContext<AccessorialCharge>();
  const method = useWatch({ name: "method" });

  const methodIsPerUnit = method === "PerUnit";

  useEffect(() => {
    if (!methodIsPerUnit) {
      setValue("rateUnit", undefined);
    }
  }, [methodIsPerUnit, setValue]);

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
      <FormControl cols={methodIsPerUnit ? 1 : "full"}>
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
      {methodIsPerUnit && (
        <FormControl>
          <SelectField
            control={control}
            rules={{ required: methodIsPerUnit }}
            name="rateUnit"
            label="Rate Unit"
            placeholder="Rate Unit"
            description="Unit of measure for this charge (mile, hour, day, stop)"
            options={rateUnitChoices}
          />
        </FormControl>
      )}
      <FormControl cols="full">
        <NumberField
          control={control}
          rules={{ required: true }}
          name="amount"
          label="Amount"
          placeholder="Amount"
          decimalScale={2}
          sideText="USD"
          thousandSeparator
          description={
            amountDescriptions[method as AccessorialCharge["method"]] ?? amountDescriptions.Flat
          }
        />
      </FormControl>
    </FormGroup>
  );
}
