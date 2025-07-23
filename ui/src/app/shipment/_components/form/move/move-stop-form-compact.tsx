/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { AutoCompleteDateTimeField } from "@/components/fields/datetime-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { LocationAutocompleteField } from "@/components/ui/autocomplete-fields";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { FormControl, FormGroup } from "@/components/ui/form";
import { Icon } from "@/components/ui/icons";
import { NumberField } from "@/components/ui/number-input";
import { stopStatusChoices, stopTypeChoices } from "@/lib/choices";
import { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { StopType } from "@/lib/schemas/stop-schema";
import { cn, formatLocation } from "@/lib/utils";
import { faInfoCircle, faLocationDot } from "@fortawesome/pro-solid-svg-icons";
import { memo, useEffect, useMemo } from "react";
import { useFormContext } from "react-hook-form";
import { useLocationData } from "../../sidebar/stop-details/queries";

type CompactStopFormProps = {
  moveIdx: number;
  stopIdx: number;
  onCancel: () => void;
  onSave: () => void;
  isInlineForm?: boolean;
  isFirstOrLastStop?: boolean;
};

const CompactStopFormComponent = ({
  moveIdx,
  stopIdx,
  onCancel,
  onSave,
  isInlineForm = false,
  isFirstOrLastStop = false,
}: CompactStopFormProps) => {
  const { control, watch, setValue, getValues } =
    useFormContext<ShipmentSchema>();
  const stopType = watch(`moves.${moveIdx}.stops.${stopIdx}.type`);
  const stopsLength = watch(`moves.${moveIdx}.stops`)?.length || 0;
  const locationId = watch(`moves.${moveIdx}.stops.${stopIdx}.locationId`);

  // Determine if this is first or last stop
  const isFirstStop = stopIdx === 0;
  const isLastStop = stopIdx === stopsLength - 1;

  // Memoize the stop title to prevent recalculation on each render
  const stopTitle = useMemo(() => {
    if (isFirstStop) return "Origin Stop (Pickup)";
    if (isLastStop) return "Destination Stop (Delivery)";
    return `Intermediate Stop ${stopIdx + 1}`;
  }, [isFirstStop, isLastStop, stopIdx]);

  const { data: locationData, isLoading: isLoadingLocation } =
    useLocationData(locationId);

  // Set address when location changes
  useEffect(() => {
    if (!isLoadingLocation && locationId && locationData) {
      const formattedLocation = formatLocation(locationData);
      setValue(
        `moves.${moveIdx}.stops.${stopIdx}.addressLine`,
        formattedLocation,
      );

      // Get current move values
      const currentValues = getValues();
      const currentMove = currentValues.moves?.[moveIdx];

      if (currentMove && currentMove.stops && currentMove.stops[stopIdx]) {
        // Update the stop with location data
        const updatedStop = {
          ...currentMove.stops[stopIdx],
          location: locationData,
        };

        // Update all the stops
        const updatedStops = [...currentMove.stops];
        updatedStops[stopIdx] = updatedStop;

        // Update the entire move
        setValue(`moves.${moveIdx}`, {
          ...currentMove,
          stops: updatedStops,
        });
      }
    }
  }, [
    isLoadingLocation,
    locationId,
    locationData,
    setValue,
    moveIdx,
    stopIdx,
    getValues,
  ]);

  return (
    <Card className={cn("border", isInlineForm ? "p-4" : "p-5")}>
      <div className="flex justify-between items-center mb-4">
        <div className="flex items-center gap-2">
          <div
            className={cn(
              "size-8 rounded-full flex items-center justify-center",
              stopType === StopType.enum.Pickup
                ? "bg-blue-100 text-blue-600"
                : "bg-green-100 text-green-600",
            )}
          >
            <Icon icon={faLocationDot} className="size-4" />
          </div>
          <h3 className="text-sm font-semibold text-foreground">{stopTitle}</h3>
        </div>
        <div className="flex gap-2">
          <Button size="sm" variant="outline" onClick={onCancel}>
            Cancel
          </Button>
          <Button size="sm" onClick={onSave}>
            Save
          </Button>
        </div>
      </div>

      <div className="bg-blue-50 border border-blue-100 rounded-md p-3 mb-4 flex items-start gap-2">
        <Icon icon={faInfoCircle} className="size-4 text-blue-500 mt-0.5" />
        <div className="text-sm text-blue-700">
          <p>
            {isFirstStop
              ? "This is the origin pickup stop for this move. The stop type cannot be changed."
              : "This is the destination delivery stop for this move. The stop type cannot be changed."}
          </p>
        </div>
      </div>

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
                description="Defines the designated category or function of this stop."
                isReadOnly={isFirstOrLastStop}
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
              <NumberField
                name={`moves.${moveIdx}.stops.${stopIdx}.pieces`}
                control={control}
                label="Pieces"
                placeholder="Enter quantity"
                description="Specifies the total number of items at this stop."
                sideText="pcs"
              />
            </FormControl>
            <FormControl>
              <NumberField
                name={`moves.${moveIdx}.stops.${stopIdx}.weight`}
                control={control}
                label="Weight"
                placeholder="Enter weight"
                description="Specifies the total freight weight for this stop."
                sideText="lbs"
              />
            </FormControl>
            <FormControl cols="full">
              <LocationAutocompleteField<ShipmentSchema>
                name={`moves.${moveIdx}.stops.${stopIdx}.locationId`}
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
                name={`moves.${moveIdx}.stops.${stopIdx}.addressLine`}
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
            <div className="rounded-lg bg-muted p-4">
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
    </Card>
  );
};

export const CompactStopForm = memo(CompactStopFormComponent);
