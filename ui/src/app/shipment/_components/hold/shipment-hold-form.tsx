import { HoldReasonAutocompleteField } from "@/components/ui/autocomplete-fields";
import { FormControl, FormGroup } from "@/components/ui/form";
import { HoldShipmentRequestSchema } from "@/lib/schemas/shipment-hold-schema";
import { useFormContext } from "react-hook-form";

export function ShipmentHoldForm() {
  const { control } = useFormContext<HoldShipmentRequestSchema>();

  return (
    <FormGroup>
      <FormControl>
        <HoldReasonAutocompleteField
          control={control}
          name="holdReasonId"
          label="Hold Reason"
          rules={{ required: true }}
          description="The type of hold to apply to the shipment."
          placeholder="Select hold type"
        />
      </FormControl>
    </FormGroup>
  );
}
