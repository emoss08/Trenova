import {
  EquipmentManufacturerAutocompleteField,
  EquipmentTypeAutocompleteField,
  FleetCodeAutocompleteField,
  UsStateAutocompleteField,
  WorkerAutocompleteField,
} from "@/components/autocomplete-fields";
import { CustomFieldsSection } from "@/components/custom-fields-section";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { FormControl, FormGroup, FormSection } from "@/components/ui/form";
import { equipmentStatusChoices } from "@/lib/choices";
import { equipmentClassSchema } from "@/types/equipment-type";
import { statusSchema } from "@/types/helpers";
import type { Tractor } from "@/types/tractor";
import { type Control, useFormContext } from "react-hook-form";

function GeneralInformationSection({ control }: { control: Control<Tractor> }) {
  return (
    <FormGroup cols={2} className="pb-2">
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
          maxLength={50}
        />
      </FormControl>
      <FormControl>
        <EquipmentTypeAutocompleteField<Tractor>
          name="equipmentTypeId"
          control={control}
          label="Equipment Type"
          rules={{ required: true }}
          placeholder="Equipment Type"
          description="The type of equipment the tractor is categorized under."
          extraSearchParams={{
            classes: [equipmentClassSchema.enum.Tractor],
          }}
        />
      </FormControl>
      <FormControl>
        <EquipmentManufacturerAutocompleteField<Tractor>
          name="equipmentManufacturerId"
          control={control}
          label="Equip. Manufacturer"
          rules={{ required: true }}
          placeholder="Equip. Manufacturer"
          description="The manufacturer of the tractor's equipment."
          extraSearchParams={{
            status: statusSchema.enum.Active,
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
          maxLength={50}
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          name="make"
          label="Make"
          placeholder="Make"
          description="The manufacturer of the tractor."
          maxLength={50}
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
        <FleetCodeAutocompleteField<Tractor>
          name="fleetCodeId"
          control={control}
          clearable
          label="Fleet Code"
          placeholder="Fleet Code"
          description="The fleet code associated with the tractor."
        />
      </FormControl>
    </FormGroup>
  );
}

function RegistrationInformationSection({
  control,
}: {
  control: Control<Tractor>;
}) {
  return (
    <FormSection title="Registration Information" className="border-t py-2">
      <FormGroup cols={2}>
        <FormControl>
          <InputField
            control={control}
            name="vin"
            label="VIN"
            placeholder="VIN"
            description="The Vehicle Identification Number (VIN) of the tractor."
            maxLength={17}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="registrationNumber"
            label="Registration Number"
            placeholder="Registration Number"
            description="The unique registration number assigned to the tractor."
            maxLength={50}
          />
        </FormControl>
        <FormControl>
          <UsStateAutocompleteField
            control={control}
            name="stateId"
            label="License State"
            placeholder="License State"
            description="The U.S. state where the tractor is licensed."
          />
        </FormControl>
        <FormControl>
          <AutoCompleteDateField
            control={control}
            name="registrationExpiry"
            label="Registration Expiry"
            description="The expiration date of the tractor's registration."
            placeholder="Registration Expiry"
          />
        </FormControl>
        <FormControl cols="full">
          <InputField
            control={control}
            name="licensePlateNumber"
            label="License Plate Number"
            placeholder="License Plate Number"
            description="The license plate number associated with the tractor."
            maxLength={50}
          />
        </FormControl>
      </FormGroup>
    </FormSection>
  );
}

function WorkerAssignmentSection({ control }: { control: Control<Tractor> }) {
  return (
    <FormSection title="Worker Assignment" className="border-t py-2">
      <FormGroup cols={2}>
        <FormControl>
          <WorkerAutocompleteField<Tractor>
            name="primaryWorkerId"
            control={control}
            label="Primary Worker"
            rules={{ required: true }}
            placeholder="Select Primary Worker"
            description="The primary worker assigned to this tractor."
          />
        </FormControl>
        <FormControl>
          <WorkerAutocompleteField<Tractor>
            name="secondaryWorkerId"
            control={control}
            clearable
            label="Secondary Worker"
            placeholder="Select Secondary Worker"
            description="An optional secondary worker assigned to this tractor."
          />
        </FormControl>
      </FormGroup>
    </FormSection>
  );
}

export function TractorForm() {
  const { control } = useFormContext<Tractor>();

  return (
    <>
      <GeneralInformationSection control={control} />
      <RegistrationInformationSection control={control} />
      <WorkerAssignmentSection control={control} />
      <CustomFieldsSection resourceType="tractor" control={control} />
    </>
  );
}
