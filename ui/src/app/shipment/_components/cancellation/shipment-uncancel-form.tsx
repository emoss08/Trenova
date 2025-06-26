import { SwitchField } from "@/components/fields/switch-field";
import { FormControl, FormGroup } from "@/components/ui/form";
import { ShipmentUncancelSchema } from "@/lib/schemas/shipment-cancellation-schema";
import { useFormContext } from "react-hook-form";

export function ShipmentUncancelForm() {
  const { control } = useFormContext<ShipmentUncancelSchema>();

  return (
    <FormGroup cols={1}>
      <FormControl cols="full">
        <SwitchField
          control={control}
          name="updateAppointments"
          label="Update Appointments"
          outlined
          description="Override the appointments of the shipment."
        />
      </FormControl>
    </FormGroup>
  );
}
