import { AutoCompleteDateField } from "@/components/fields/date-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import {
  EquipmentManufacturerAutocompleteField,
  EquipmentTypeAutocompleteField,
  FleetCodeAutocompleteField,
  WorkerAutocompleteField,
} from "@/components/ui/autocomplete-fields";
import { FormControl, FormGroup, FormSection } from "@/components/ui/form";
import { NumberField } from "@/components/ui/number-input";
import { equipmentStatusChoices } from "@/lib/choices";
import { queries } from "@/lib/queries";
import { type TractorSchema } from "@/lib/schemas/tractor-schema";
import { Status } from "@/types/common";
import { EquipmentClass } from "@/types/equipment-type";
import { useQuery } from "@tanstack/react-query";
import { useFormContext } from "react-hook-form";

export function TractorForm() {
  const { control } = useFormContext<TractorSchema>();

  return (
    <>
      <FormGroup cols={2}>
        <FormControl>
          <SelectField
            control={control}
            options={equipmentStatusChoices}
            rules={{ required: true }}
            name="status"
            label="Status"
            placeholder="Status"
            description="Indicates the current operational status of the tractor."
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            rules={{ required: true }}
            name="code"
            label="Code"
            placeholder="Code"
            description="A unique code identifying the tractor."
          />
        </FormControl>
        <FormControl>
          <EquipmentTypeAutocompleteField<TractorSchema>
            name="equipmentTypeId"
            control={control}
            label="Equipment Type"
            rules={{ required: true }}
            placeholder="Equipment Type"
            extraSearchParams={{
              classes: [EquipmentClass.Tractor],
            }}
            description="The type of equipment the tractor is categorized under."
          />
        </FormControl>
        <FormControl>
          <EquipmentManufacturerAutocompleteField<TractorSchema>
            name="equipmentManufacturerId"
            control={control}
            label="Equip. Manufacturer"
            placeholder="Equip. Manufacturer"
            description="The manufacturer of the tractor's equipment."
            rules={{ required: true }}
            extraSearchParams={{
              status: [Status.Active],
            }}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="model"
            label="Model"
            placeholder="Model"
            description="The specific model of the tractor."
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="make"
            label="Make"
            placeholder="Make"
            description="The manufacturer of the tractor."
          />
        </FormControl>
        <FormControl>
          <NumberField
            control={control}
            name="year"
            label="Year"
            placeholder="Year"
            description="The production year of the tractor."
          />
        </FormControl>
        <FormControl>
          <FleetCodeAutocompleteField<TractorSchema>
            name="fleetCodeId"
            control={control}
            label="Fleet Code"
            placeholder="Fleet Code"
            description="The fleet code associated with the tractor"
            rules={{ required: true }}
          />
        </FormControl>
      </FormGroup>
      <RegistrationInformationSection />
      <WorkerAssignmentSection />
    </>
  );
}

function RegistrationInformationSection() {
  const { control } = useFormContext<TractorSchema>();
  const usStates = useQuery({
    ...queries.usState.options(),
  });
  const usStateOptions = usStates.data?.results ?? [];

  return (
    <FormSection title="Registration Information" className="border-t pt-4">
      <FormGroup cols={2}>
        <FormControl>
          <InputField
            control={control}
            name="licensePlateNumber"
            label="License Plate Number"
            placeholder="License Plate Number"
            description="The license plate number associated with the tractor."
          />
        </FormControl>
        <FormControl>
          <SelectField
            control={control}
            name="stateId"
            label="License State"
            placeholder="License State"
            description="The U.S. state where the tractor is licensed."
            isClearable
            options={usStateOptions}
            isLoading={usStates.isLoading}
            isFetchError={usStates.isError}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="registrationNumber"
            label="Registration Number"
            placeholder="Registration Number"
            description="The unique registration number assigned to the tractor."
          />
        </FormControl>
        <FormControl>
          <AutoCompleteDateField
            control={control}
            clearable
            name="registrationExpiry"
            label="Registration Expiry"
            description="The expiration date of the tractor's registration."
            placeholder="Registration Expiry"
          />
        </FormControl>
        <FormControl cols="full">
          <InputField
            control={control}
            name="vin"
            label="VIN"
            placeholder="VIN"
            description="The Vehicle Identification Number (VIN) of the tractor."
          />
        </FormControl>
      </FormGroup>
    </FormSection>
  );
}

function WorkerAssignmentSection() {
  const { control } = useFormContext<TractorSchema>();
  return (
    <FormSection title="Worker Assignment" className="border-t pt-4">
      <FormGroup cols={2}>
        <FormControl>
          <WorkerAutocompleteField<TractorSchema>
            name="primaryWorkerId"
            control={control}
            label="Primary Worker"
            rules={{ required: true }}
            placeholder="Select Primary Worker"
            description="Select the primary worker for the assignment."
          />
        </FormControl>
        <FormControl>
          <WorkerAutocompleteField<TractorSchema>
            name="secondaryWorkerId"
            control={control}
            clearable
            label="Secondary Worker"
            placeholder="Select Secondary Worker"
            description="Select the secondary worker for the assignment."
          />
        </FormControl>
      </FormGroup>
    </FormSection>
  );
}
