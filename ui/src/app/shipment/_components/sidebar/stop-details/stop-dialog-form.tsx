import { AutoCompleteDateTimeField } from "@/components/fields/datetime-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { LocationAutocompleteField } from "@/components/ui/autocomplete-fields";
import { FormControl, FormGroup } from "@/components/ui/form";
import { NumberField } from "@/components/ui/number-input";
import { stopStatusChoices, stopTypeChoices } from "@/lib/choices";
import { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { formatLocation } from "@/lib/utils";
import { useEffect } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import { useLocationData } from "./queries";

export function StopDialogForm({
  moveIdx,
  stopIdx,
  stopFieldName = `moves.${moveIdx}.stops.${stopIdx}`,
}: {
  moveIdx: number;
  stopIdx: number;
  stopFieldName?: string;
}) {
  const { control, setValue, getValues } = useFormContext<ShipmentSchema>();

  // * StopIdx, is nullable because there are instances where the dialog is going to add a new stop, and there is no stop index yet.
  // * If there is no stop index, we will use the append function to add a new stop.
  // * If there is a stop index, we will use the update function to update the stop because the stop index is already set.

  const locationId = useWatch({
    control,
    name: `${stopFieldName}.locationId` as any,
  });

  const { data: locationData, isLoading: isLoadingLocation } =
    useLocationData(locationId);

  // Keep the address prefill functionality when a location is selected
  useEffect(() => {
    if (!isLoadingLocation && locationId && locationData) {
      const formattedLocation = formatLocation(locationData);
      setValue(`${stopFieldName}.addressLine` as any, formattedLocation, {
        shouldValidate: true,
      });

      // For the local form, we just need to set the location data on the stop
      if (stopFieldName === "stop") {
        const currentStop = getValues(stopFieldName as any);
        setValue(stopFieldName as any, {
          ...currentStop,
          location: locationData,
        });
      } else {
        // For the main form, update the move structure
        const currentValues = getValues();
        const currentMove = currentValues.moves?.[moveIdx];

        if (currentMove && currentMove.stops && currentMove.stops[stopIdx]) {
          const updatedStop = {
            ...currentMove.stops[stopIdx],
            location: locationData,
          };

          const updatedStops = [...currentMove.stops];
          updatedStops[stopIdx] = updatedStop;

          setValue(`moves.${moveIdx}`, {
            ...currentMove,
            stops: updatedStops,
          });
        }
      }
    }
  }, [
    isLoadingLocation,
    locationId,
    locationData,
    stopFieldName,
    setValue,
    moveIdx,
    stopIdx,
    getValues,
  ]);

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
              name={`${stopFieldName}.type` as any}
              label="Stop Type"
              placeholder="Select type"
              description="Defines the designated category or function of this stop."
              options={stopTypeChoices}
            />
          </FormControl>
          <FormControl>
            <SelectField
              control={control}
              isReadOnly
              name={`${stopFieldName}.status` as any}
              label="Current Status"
              placeholder="Select status"
              description="Indicates the current operational status of this stop."
              options={stopStatusChoices}
            />
          </FormControl>
          <FormControl>
            <NumberField
              name={`${stopFieldName}.pieces` as any}
              control={control}
              label="Pieces"
              placeholder="Enter quantity"
              description="Specifies the total number of items at this stop."
              sideText="pcs"
            />
          </FormControl>
          <FormControl>
            <NumberField
              name={`${stopFieldName}.weight` as any}
              control={control}
              label="Weight"
              placeholder="Enter weight"
              description="Specifies the total freight weight for this stop."
              sideText="lbs"
            />
          </FormControl>
          <FormControl cols="full">
            <LocationAutocompleteField
              name={`${stopFieldName}.locationId` as any}
              control={control}
              label="Location"
              rules={{ required: true }}
              placeholder="Select location"
              description="Select the designated location for this stop."
              extraSearchParams={{
                includeState: "true",
              }}
            />
          </FormControl>
          <FormControl cols="full">
            <InputField
              name={`${stopFieldName}.addressLine` as any}
              rules={{ required: true }}
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
          <div className="rounded-lg bg-muted p-4">
            <h4 className="text-sm font-medium text-foreground mb-3">
              Planned Times
            </h4>
            <FormGroup cols={2} className="gap-4">
              <FormControl>
                <AutoCompleteDateTimeField
                  name={`${stopFieldName}.plannedArrival` as any}
                  control={control}
                  rules={{ required: true }}
                  label="Planned Arrival"
                  placeholder="Select planned arrival"
                  description="Indicates the scheduled arrival time for this stop."
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateTimeField
                  name={`${stopFieldName}.plannedDeparture` as any}
                  control={control}
                  rules={{ required: true }}
                  label="Planned Departure"
                  placeholder="Select planned departure"
                  description="Indicates the scheduled departure time from this stop."
                />
              </FormControl>
            </FormGroup>
          </div>
          <div className="rounded-lg bg-muted p-4">
            <h4 className="text-sm font-medium text-foreground mb-3">
              Actual Times
            </h4>
            <FormGroup cols={2} className="gap-4">
              <FormControl>
                <AutoCompleteDateTimeField
                  name={`${stopFieldName}.actualArrival` as any}
                  control={control}
                  label="Actual Arrival"
                  placeholder="Select actual arrival"
                  description="Records the actual arrival time at this stop."
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateTimeField
                  name={`${stopFieldName}.actualDeparture` as any}
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
