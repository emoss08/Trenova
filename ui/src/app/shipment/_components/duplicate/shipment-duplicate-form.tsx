import { SwitchField } from "@/components/fields/switch-field";
import { FormControl, FormGroup } from "@/components/ui/form";
import { type ShipmentDuplicateSchema } from "@/lib/schemas/shipment-duplicate-schema";
import { useFormContext } from "react-hook-form";

export function ShipmentDuplicateForm() {
  const { control } = useFormContext<ShipmentDuplicateSchema>();

  return (
    <FormGroup cols={1}>
      <FormControl>
        <SwitchField
          name="overrideDates"
          control={control}
          label="Override Dates"
          outlined
          description="Override the planned arrival and departure dates for each stop in the new shipment."
        />
      </FormControl>
      <FormControl>
        <SwitchField
          name="includeCommodities"
          control={control}
          label="Include Commodities"
          outlined
          description="Include all commodities from the original shipment in the new shipment. (Pieces, Weight, etc.)"
        />
      </FormControl>
    </FormGroup>
  );
}
