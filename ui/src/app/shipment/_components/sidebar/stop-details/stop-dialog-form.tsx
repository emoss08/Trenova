import { AutocompleteField } from "@/components/fields/autocomplete";
import { AutoCompleteDateTimeField } from "@/components/fields/datetime-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { FormControl, FormGroup } from "@/components/ui/form";
import { stopStatusChoices, stopTypeChoices } from "@/lib/choices";
import { type LocationSchema } from "@/lib/schemas/location-schema";
import { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { formatLocation } from "@/lib/utils";
import { useEffect } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import { useLocationData } from "./queries";

export function StopDialogForm({
  moveIdx,
  stopIdx,
}: {
  moveIdx: number;
  stopIdx: number;
}) {
  const { control, setValue } = useFormContext<ShipmentSchema>();

  const locationId = useWatch({
    control,
    name: `moves.${moveIdx}.stops.${stopIdx}.locationId`,
  });

  const { data: locationData, isLoading: isLoadingLocation } =
    useLocationData(locationId);

  useEffect(() => {
    if (!isLoadingLocation && locationId && locationData) {
      const formattedLocation = formatLocation(locationData);
      setValue(
        `moves.${moveIdx}.stops.${stopIdx}.addressLine`,
        formattedLocation,
      );
    }
  }, [isLoadingLocation, locationId, locationData, setValue, moveIdx, stopIdx]);

  return (
    <div className="space-y-2">
      <div>
        <div className="flex items-center gap-2 mb-1">
          <h3 className="text-sm font-semibold text-foreground">
            Basic Information
          </h3>
        </div>
        <p className="text-2xs text-muted-foreground mb-3">
          Define the fundamental details and current status of this stop.
        </p>
        <FormGroup cols={2} className="gap-4">
          <FormControl>
            <SelectField
              control={control}
              rules={{ required: true }}
              name={`moves.${moveIdx}.stops.${stopIdx}.type`}
              label="Stop Type"
              placeholder="Select type"
              description="Defines the designated category or function of this stop (read-only)."
              isReadOnly
              options={stopTypeChoices}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              isReadOnly
              rules={{ required: true }}
              name={`moves.${moveIdx}.stops.${stopIdx}.status`}
              label="Current Status"
              placeholder="Select status"
              description="Indicates the current operational status of this stop."
              options={stopStatusChoices}
            />
          </FormControl>

          <FormControl>
            <InputField
              name={`moves.${moveIdx}.stops.${stopIdx}.pieces`}
              control={control}
              label="Pieces"
              placeholder="Enter quantity"
              type="number"
              description="Specifies the total number of items at this stop."
            />
          </FormControl>
          <FormControl>
            <InputField
              name={`moves.${moveIdx}.stops.${stopIdx}.weight`}
              control={control}
              label="Weight"
              placeholder="Enter weight"
              type="number"
              description="Specifies the total freight weight for this stop."
            />
          </FormControl>

          <FormControl cols="full">
            <AutocompleteField<LocationSchema, ShipmentSchema>
              name={`moves.${moveIdx}.stops.${stopIdx}.locationId`}
              control={control}
              link="/locations/"
              label="Location"
              rules={{ required: true }}
              placeholder="Select location"
              description="Select the designated location for this stop."
              getOptionValue={(option) => option.id || ""}
              getDisplayValue={(option) => option.name}
              renderOption={(option) => option.name}
            />
          </FormControl>
          <FormControl cols="full">
            <InputField
              name={`moves.${moveIdx}.stops.${stopIdx}.addressLine`}
              control={control}
              label="Address"
              placeholder="Full address details"
              description="Specifies the street address or main location detail for this stop."
            />
          </FormControl>
        </FormGroup>
      </div>

      <div className="pt-2">
        <div className="flex items-center gap-2 mb-1">
          <h3 className="text-sm font-semibold text-foreground">
            Schedule Information
          </h3>
        </div>
        <p className="text-2xs text-muted-foreground mb-3">
          Manage planned and actual arrival/departure times for this stop.
        </p>
        <div className="space-y-4">
          <div className="rounded-lg bg-accent/50 p-4">
            <h4 className="text-sm font-medium text-foreground mb-3">
              Planned Times
            </h4>
            <FormGroup cols={2} className="gap-4">
              <FormControl>
                <AutoCompleteDateTimeField
                  name={`moves.${moveIdx}.stops.${stopIdx}.plannedArrival`}
                  control={control}
                  rules={{ required: true }}
                  label="Planned Arrival"
                  placeholder="Select planned arrival"
                  description="Indicates the scheduled arrival time for this stop."
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateTimeField
                  name={`moves.${moveIdx}.stops.${stopIdx}.plannedDeparture`}
                  control={control}
                  rules={{ required: true }}
                  label="Planned Departure"
                  placeholder="Select planned departure"
                  description="Indicates the scheduled departure time from this stop."
                />
              </FormControl>
            </FormGroup>
          </div>

          <div className="rounded-lg bg-accent/50 p-4">
            <h4 className="text-sm font-medium text-foreground mb-3">
              Actual Times
            </h4>
            <FormGroup cols={2} className="gap-4">
              <FormControl>
                <AutoCompleteDateTimeField
                  name={`moves.${moveIdx}.stops.${stopIdx}.actualArrival`}
                  control={control}
                  label="Actual Arrival"
                  placeholder="Select actual arrival"
                  description="Records the actual arrival time at this stop."
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateTimeField
                  name={`moves.${moveIdx}.stops.${stopIdx}.actualDeparture`}
                  control={control}
                  label="Actual Departure"
                  placeholder="Select actual departure"
                  description="Records the actual departure time from this stop."
                />
              </FormControl>
            </FormGroup>
          </div>
        </div>
      </div>
    </div>
  );
}
