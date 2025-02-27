import { AutocompleteField } from "@/components/fields/autocomplete";
import { AutoCompleteDateTimeField } from "@/components/fields/datetime-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { FormControl, FormGroup } from "@/components/ui/form";
import { Icon } from "@/components/ui/icons";
import { stopStatusChoices, stopTypeChoices } from "@/lib/choices";
import { LocationSchema } from "@/lib/schemas/location-schema";
import { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { cn, formatLocation } from "@/lib/utils";
import { StopType } from "@/types/stop";
import { faInfoCircle, faLocationDot } from "@fortawesome/pro-solid-svg-icons";
import { useEffect } from "react";
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

export function CompactStopForm({
  moveIdx,
  stopIdx,
  onCancel,
  onSave,
  isInlineForm = false,
  isFirstOrLastStop = false,
}: CompactStopFormProps) {
  const { control, watch, setValue } = useFormContext<ShipmentSchema>();
  const stopType = watch(`moves.${moveIdx}.stops.${stopIdx}.type`);
  const stopsLength = watch(`moves.${moveIdx}.stops`)?.length || 0;
  const locationId = watch(`moves.${moveIdx}.stops.${stopIdx}.locationId`);

  // Determine if this is first or last stop
  const isFirstStop = stopIdx === 0;
  const isLastStop = stopIdx === stopsLength - 1;

  // Determine appropriate title
  const getStopTitle = () => {
    if (isFirstStop) return "Origin Stop (Pickup)";
    if (isLastStop) return "Destination Stop (Delivery)";
    return `Intermediate Stop ${stopIdx + 1}`;
  };

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
    <Card className={`border ${isInlineForm ? "p-4" : "p-5"}`}>
      <div className="flex justify-between items-center mb-4">
        <div className="flex items-center gap-2">
          <div
            className={cn(
              "size-8 rounded-full flex items-center justify-center",
              stopType === StopType.Pickup
                ? "bg-blue-100 text-blue-600"
                : "bg-green-100 text-green-600",
            )}
          >
            <Icon icon={faLocationDot} className="size-4" />
          </div>
          <h3 className="text-sm font-semibold text-foreground">
            {getStopTitle()}
          </h3>
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

      {isFirstOrLastStop && (
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
      )}

      <div className="space-y-2">
        {/* Basic Information Row */}
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
              <InputField
                name={`moves.${moveIdx}.stops.${stopIdx}.pieces`}
                control={control}
                label="Pieces"
                placeholder="Enter quantity"
                type="number"
                description="Specifies the total number of items at this stop."
                sideText="pcs"
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
                sideText="lbs"
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
                renderOption={(option) => (
                  <div className="flex flex-col gap-0.5 items-start size-full">
                    <span className="text-sm font-normal">{option.name}</span>
                    <span className="text-2xs text-muted-foreground truncate w-full">
                      {formatLocation(option)}
                    </span>
                  </div>
                )}
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
    </Card>
  );
}
