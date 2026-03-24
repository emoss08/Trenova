import {
  EquipmentManufacturerAutocompleteField,
  EquipmentTypeAutocompleteField,
  FleetCodeAutocompleteField,
  UsStateAutocompleteField,
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
import type { Trailer } from "@/types/trailer";
import { type Control, useFormContext } from "react-hook-form";

function GeneralInformationSection({ control }: { control: Control<Trailer> }) {
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
          description="Indicates the current operational status of the trailer."
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
          maxLength={50}
        />
      </FormControl>
      <FormControl>
        <EquipmentTypeAutocompleteField<Trailer>
          name="equipmentTypeId"
          control={control}
          label="Equipment Type"
          rules={{ required: true }}
          placeholder="Equipment Type"
          description="The type of equipment the trailer is categorized under."
          extraSearchParams={{
            classes: [
              equipmentClassSchema.enum.Trailer,
              equipmentClassSchema.enum.Container,
              equipmentClassSchema.enum.Other, // May contain things like a flatbed trailer, etc.
            ],
          }}
        />
      </FormControl>
      <FormControl>
        <EquipmentManufacturerAutocompleteField<Trailer>
          name="equipmentManufacturerId"
          control={control}
          label="Equip. Manufacturer"
          rules={{ required: true }}
          placeholder="Equip. Manufacturer"
          description="The manufacturer of the trailer's equipment."
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
          rules={{ required: true }}
          placeholder="Model"
          description="The specific model of the trailer."
          maxLength={50}
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          name="make"
          label="Make"
          rules={{ required: true }}
          placeholder="Make"
          description="The manufacturer of the trailer."
          maxLength={50}
        />
      </FormControl>
      <FormControl>
        <NumberField
          control={control}
          name="year"
          label="Year"
          rules={{ required: true }}
          placeholder="Year"
          description="The production year of the trailer."
        />
      </FormControl>
      <FormControl>
        <NumberField
          control={control}
          name="maxLoadWeight"
          label="Max Load Weight"
          sideText="lbs"
          placeholder="Max Load Weight"
          description="The maximum load weight the trailer can carry."
        />
      </FormControl>
      <FormControl cols="full">
        <FleetCodeAutocompleteField<Trailer>
          name="fleetCodeId"
          control={control}
          clearable
          label="Fleet Code"
          placeholder="Fleet Code"
          description="The fleet code associated with the trailer."
        />
      </FormControl>
    </FormGroup>
  );
}

function RegistrationInformationSecond({
  control,
}: {
  control: Control<Trailer>;
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
            description="The Vehicle Identification Number (VIN) of the trailer."
            maxLength={17}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="registrationNumber"
            label="Registration Number"
            placeholder="Registration Number"
            description="The unique registration number assigned to the trailer."
            maxLength={50}
          />
        </FormControl>
        <FormControl>
          <UsStateAutocompleteField
            control={control}
            name="registrationStateId"
            label="Registration State"
            placeholder="Registration State"
            description="The U.S. state where the trailer is registered."
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
            maxLength={50}
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
    </FormSection>
  );
}

export function TrailerForm() {
  const { control } = useFormContext<Trailer>();

  return (
    <>
      <GeneralInformationSection control={control} />
      <RegistrationInformationSecond control={control} />
      <CustomFieldsSection resourceType="trailer" control={control} />
    </>
  );
}
