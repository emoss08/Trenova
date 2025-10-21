import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { LocationAutocompleteField } from "@/components/ui/autocomplete-fields";
import { FormControl, FormGroup, FormSection } from "@/components/ui/form";
import { NumberField } from "@/components/ui/number-input";
import { stopStatusChoices, stopTypeChoices } from "@/lib/choices";
import { queries } from "@/lib/queries";
import { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { formatLocation } from "@/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { useEffect } from "react";
import { useFormContext, useWatch } from "react-hook-form";

export function StopBasicInformationForm({
  moveIdx,
  stopIdx,
}: {
  moveIdx: number;
  stopIdx: number;
}) {
  const { control, setValue, getValues } = useFormContext<ShipmentSchema>();
  const locationId = useWatch({
    control,
    name: `moves.${moveIdx}.stops.${stopIdx}.locationId`,
    defaultValue: "",
  });

  const { data: locationData, isLoading: isLoadingLocation } = useQuery({
    ...queries.location.getById(locationId),
    enabled: !!locationId,
  });

  useEffect(() => {
    if (!isLoadingLocation && locationId && locationData) {
      const formattedLocation = formatLocation(locationData);
      setValue(
        `moves.${moveIdx}.stops.${stopIdx}.addressLine`,
        formattedLocation,
        {
          shouldValidate: true,
        },
      );

      const currentStop = getValues(`moves.${moveIdx}.stops.${stopIdx}`);
      setValue(`moves.${moveIdx}.stops.${stopIdx}`, {
        ...currentStop,
        location: locationData,
      });
    }
  }, [
    isLoadingLocation,
    locationId,
    locationData,
    moveIdx,
    stopIdx,
    setValue,
    getValues,
  ]);

  return (
    <FormSection
      title="Basic Information"
      description="Define the fundamental details and current status of this stop."
      className="mt-2 gap-1"
    >
      <FormGroup cols={2} className="gap-4">
        <FormControl>
          <SelectField
            control={control}
            name={`moves.${moveIdx}.stops.${stopIdx}.type`}
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
          <LocationAutocompleteField
            name={`moves.${moveIdx}.stops.${stopIdx}.locationId`}
            control={control}
            label="Location"
            rules={{ required: true }}
            placeholder="Select location"
            description="Select the designated location for this stop."
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
            maxLength={200}
          />
        </FormControl>
      </FormGroup>
    </FormSection>
  );
}
