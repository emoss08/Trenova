import { AutocompleteField } from "@/components/fields/autocomplete";
import { AutoCompleteDateField } from "@/components/fields/date-field";
import { InputField } from "@/components/fields/input-field";
import { ColorOptionValue } from "@/components/fields/select-components";
import { SelectField } from "@/components/fields/select-field";
import { FormControl, FormGroup } from "@/components/ui/form";
import { equipmentStatusChoices } from "@/lib/choices";
import { queries } from "@/lib/queries";
import { EquipmentManufacturerSchema } from "@/lib/schemas/equipment-manufacturer-schema";
import { EquipmentTypeSchema } from "@/lib/schemas/equipment-type-schema";
import { FleetCodeSchema } from "@/lib/schemas/fleet-code-schema";
import { type TrailerSchema } from "@/lib/schemas/trailer-schema";
import { useQuery } from "@tanstack/react-query";
import { Control, useFormContext } from "react-hook-form";

function GeneralInformationSection({
  control,
}: {
  control: Control<TrailerSchema>;
}) {
  return (
    <FormGroup cols={2}>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="status"
          label="Status"
          placeholder="Status"
          description="Indicates the current operational status of the trailer."
          options={equipmentStatusChoices}
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          rules={{ required: true }}
          name="code"
          label="Code"
          placeholder="Code"
          description="A unique code identifying the trailer."
        />
      </FormControl>
      <FormControl>
        <AutocompleteField<EquipmentTypeSchema, TrailerSchema>
          name="equipmentTypeId"
          control={control}
          link="/equipment-types/"
          label="Equipment Type"
          rules={{ required: true }}
          placeholder="Equipment Type"
          description="The type of equipment the trailer is categorized under."
          getOptionValue={(option) => option.id || ""}
          getDisplayValue={(option) => (
            <ColorOptionValue color={option.color} value={option.code} />
          )}
          renderOption={(option) => (
            <ColorOptionValue color={option.color} value={option.code} />
          )}
        />
      </FormControl>
      <FormControl>
        <AutocompleteField<EquipmentManufacturerSchema, TrailerSchema>
          name="equipmentManufacturerId"
          control={control}
          link="/equipment-manufacturers/"
          label="Equip. Manufacturer"
          placeholder="Equip. Manufacturer"
          description="The manufacturer of the trailer's equipment."
          getOptionValue={(option) => option.id || ""}
          getDisplayValue={(option) => option.name}
          renderOption={(option) => option.name}
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          name="model"
          label="Model"
          placeholder="Model"
          description="The specific model of the trailer."
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          name="make"
          label="Make"
          placeholder="Make"
          description="The manufacturer of the trailer."
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          type="number"
          name="year"
          label="Year"
          placeholder="Year"
          description="The production year of the trailer."
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          name="maxLoadWeight"
          type="number"
          label="Max Load Weight"
          placeholder="Max Load Weight"
          description="The maximum load weight the trailer can carry."
        />
      </FormControl>
      <FormControl cols="full">
        <AutocompleteField<FleetCodeSchema, TrailerSchema>
          name="fleetCodeId"
          control={control}
          link="/fleet-codes/"
          label="Fleet Code"
          placeholder="Fleet Code"
          description="The fleet code associated with the trailer."
          rules={{ required: true }}
          getOptionValue={(option) => option.id || ""}
          getDisplayValue={(option) => (
            <ColorOptionValue color={option.color} value={option.name} />
          )}
          renderOption={(option) => (
            <ColorOptionValue color={option.color} value={option.name} />
          )}
        />
      </FormControl>
    </FormGroup>
  );
}

function RegistrationInformationSecond({
  control,
}: {
  control: Control<TrailerSchema>;
}) {
  const usStates = useQuery({
    ...queries.usState.options(),
  });
  const usStateOptions = usStates.data?.results ?? [];

  return (
    <FormGroup cols={2}>
      <FormControl>
        <InputField
          control={control}
          name="vin"
          label="VIN"
          placeholder="VIN"
          description="The Vehicle Identification Number (VIN) of the trailer."
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          name="registrationNumber"
          label="Registration Number"
          placeholder="Registration Number"
          description="The unique registration number assigned to the trailer."
        />
      </FormControl>
      <FormControl>
        <SelectField
          control={control}
          name="registrationStateId"
          label="Registration State"
          placeholder="Registration State"
          description="The U.S. state where the trailer is registered."
          options={usStateOptions}
          isLoading={usStates.isLoading}
          isFetchError={usStates.isError}
        />
      </FormControl>
      <FormControl>
        <AutoCompleteDateField
          control={control}
          name="registrationExpiry"
          label="Registration Expiry"
          description="The expiration date of the trailer's registration."
          placeholder="Registration Expiry"
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          name="licensePlateNumber"
          label="License Plate Number"
          placeholder="License Plate Number"
          description="The license plate number associated with the trailer."
        />
      </FormControl>
      <FormControl>
        <AutoCompleteDateField
          control={control}
          clearable
          name="lastInspectionDate"
          label="Last Inspection Date"
          description="The date of the trailer's most recent inspection."
          placeholder="Last Inspection Date"
        />
      </FormControl>
    </FormGroup>
  );
}

export function TrailerForm() {
  const { control } = useFormContext<TrailerSchema>();

  return (
    <>
      <GeneralInformationSection control={control} />
      <RegistrationInformationSecond control={control} />
    </>
  );
}
