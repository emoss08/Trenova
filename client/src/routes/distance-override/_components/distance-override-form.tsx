import {
  CustomerAutocompleteField,
  LocationAutocompleteField,
} from "@/components/autocomplete-fields";
import { NumberField } from "@/components/fields/number-field";
import { Button } from "@/components/ui/button";
import { FormControl, FormGroup, FormSection } from "@/components/ui/form";
import type { DistanceOverride } from "@/types/distance-override";
import { PlusIcon, TrashIcon } from "lucide-react";
import { useFieldArray, useFormContext } from "react-hook-form";

export function DistanceOverrideForm() {
  const { control } = useFormContext<DistanceOverride>();
  const { fields, append, remove } = useFieldArray({
    control,
    name: "intermediateStops",
  });

  return (
    <FormGroup cols={2}>
      <FormControl>
        <LocationAutocompleteField
          control={control}
          rules={{ required: true }}
          name="originLocationId"
          label="Origin Location"
          placeholder="Select origin location"
          description="The origin location for this distance override"
        />
      </FormControl>
      <FormControl>
        <LocationAutocompleteField
          control={control}
          rules={{ required: true }}
          name="destinationLocationId"
          label="Destination Location"
          placeholder="Select destination location"
          description="The destination location for this distance override"
        />
      </FormControl>
      <FormControl>
        <NumberField
          control={control}
          rules={{ required: true }}
          name="distance"
          label="Distance"
          placeholder="Distance"
          description="The override distance between the two locations"
        />
      </FormControl>
      <FormControl>
        <CustomerAutocompleteField
          control={control}
          name="customerId"
          label="Customer"
          placeholder="Select customer (optional)"
          description="Optionally scope this override to a specific customer"
          clearable
        />
      </FormControl>
      <FormControl cols="full">
        <FormSection
          title="Intermediate Stops"
          description="Add optional stops between origin and destination in travel order"
          action={
            <Button
              type="button"
              variant="outline"
              size="xxs"
              onClick={() => append({ locationId: "" })}
            >
              <PlusIcon className="size-3" />
              Add Stop
            </Button>
          }
        >
          {fields.length === 0 ? (
            <p className="text-xs text-muted-foreground">No intermediate stops configured.</p>
          ) : (
            <div className="space-y-2">
              {fields.map((field, index) => (
                <div key={field.id} className="grid grid-cols-[1fr_auto] items-end gap-2">
                  <LocationAutocompleteField
                    control={control}
                    rules={{ required: true }}
                    name={`intermediateStops.${index}.locationId`}
                    label={`Stop ${index + 1}`}
                    placeholder="Select stop location"
                  />
                  <Button type="button" variant="ghost" size="icon" onClick={() => remove(index)}>
                    <TrashIcon className="size-4 text-destructive" />
                  </Button>
                </div>
              ))}
            </div>
          )}
        </FormSection>
      </FormControl>
    </FormGroup>
  );
}
