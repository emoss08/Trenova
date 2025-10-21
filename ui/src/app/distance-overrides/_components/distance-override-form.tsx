import {
  CustomerAutocompleteField,
  LocationAutocompleteField,
} from "@/components/ui/autocomplete-fields";
import { FormControl, FormGroup } from "@/components/ui/form";
import { NumberField } from "@/components/ui/number-input";
import type { DistanceOverrideSchema } from "@/lib/schemas/distance-override-schema";
import { useFormContext } from "react-hook-form";

export function DistanceOverrideForm() {
  const { control } = useFormContext<DistanceOverrideSchema>();

  return (
    <FormGroup cols={2}>
      <FormControl>
        <LocationAutocompleteField<DistanceOverrideSchema>
          name="originLocationId"
          control={control}
          label="Origin Location"
          placeholder="Select Origin Location"
          description="Starting location for the distance override"
          rules={{ required: true }}
        />
      </FormControl>
      <FormControl>
        <LocationAutocompleteField<DistanceOverrideSchema>
          name="destinationLocationId"
          control={control}
          label="Destination Location"
          placeholder="Select Destination Location"
          description="Final delivery location for the distance override"
          rules={{ required: true }}
        />
      </FormControl>
      <FormControl>
        <CustomerAutocompleteField<DistanceOverrideSchema>
          name="customerId"
          control={control}
          label="Customer"
          placeholder="Select Customer"
          description="Customer account associated with this distance override"
        />
      </FormControl>
      <FormControl>
        <NumberField
          control={control}
          rules={{ required: true }}
          name="distance"
          label="Distance"
          placeholder="Distance"
          description="Distance for the distance override"
        />
      </FormControl>
    </FormGroup>
  );
}
