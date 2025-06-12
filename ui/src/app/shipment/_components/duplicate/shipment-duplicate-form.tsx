import { SwitchField } from "@/components/fields/switch-field";
import { FormControl, FormGroup, FormSection } from "@/components/ui/form";
import { NumberField } from "@/components/ui/number-input";
import { type ShipmentDuplicateSchema } from "@/lib/schemas/shipment-duplicate-schema";
import { useFormContext } from "react-hook-form";

export function ShipmentDuplicateForm() {
  const { control } = useFormContext<ShipmentDuplicateSchema>();

  return (
    <FormGroup cols={1}>
      <FormControl>
        <NumberField
          name="count"
          control={control}
          max={20}
          min={1}
          label="Number of Copies"
          rules={{ required: true, min: 1, max: 20 }}
          description="The number of shipments to duplicate."
        />
      </FormControl>
      <FormSection
        title="Duplication Options"
        description="Configure settings for the duplicated shipments."
        className="border-t pt-4"
      >
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
        <FormControl>
          <SwitchField
            name="includeAdditionalCharges"
            control={control}
            label="Include Additional Charges"
            outlined
            description="Include all additional charges from the original shipment in the new shipment."
          />
        </FormControl>
      </FormSection>
    </FormGroup>
  );
}
