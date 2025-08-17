import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup } from "@/components/ui/form";
import { holdSeverityChoices, holdTypeChoices } from "@/lib/choices";
import { ShipmentHoldSchema } from "@/lib/schemas/shipment-hold-schema";
import { useFormContext } from "react-hook-form";

export function ShipmentHoldForm() {
  const { control } = useFormContext<ShipmentHoldSchema>();

  return (
    <FormGroup>
      <FormControl>
        <SelectField
          control={control}
          name="type"
          label="Hold Type"
          options={holdTypeChoices}
          rules={{ required: true }}
          description="The type of hold to apply to the shipment."
          placeholder="Select hold type"
        />
      </FormControl>
      <FormControl>
        <SelectField
          control={control}
          name="severity"
          label="Severity"
          options={holdSeverityChoices}
          rules={{ required: true }}
          description="The severity of the hold to apply to the shipment."
          placeholder="Select severity"
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          name="reasonCode"
          label="Reason Code"
          description="The reason code for the hold to apply to the shipment. e.g. ELD_OOS, APPT_PENDING"
          placeholder="Enter reason code"
        />
      </FormControl>
      <FormControl>
        <TextareaField
          control={control}
          name="notes"
          label="Notes"
          description="Additional notes about the hold to apply to the shipment."
          placeholder="Enter notes"
        />
      </FormControl>
    </FormGroup>
  );
}
