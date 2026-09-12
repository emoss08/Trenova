import { useT } from "@trenova/shared/i18n/use-t";
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
import { equipmentStatusChoices } from "@/lib/choices";
import { equipmentClassSchema } from "@/types/equipment-type";
import type { Trailer } from "@/types/trailer";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import { statusSchema } from "@trenova/shared/types/helpers";
import { type Control, useFormContext } from "react-hook-form";

function GeneralInformationSection({ control }: { control: Control<Trailer> }) {
  const t = useT();

  return (
    <FormGroup cols={2} className="pb-2">
      <FormControl>
        <SelectField
          control={control}
          options={equipmentStatusChoices}
          rules={{ required: true }}
          name="status"
          label={t("Status")}
          placeholder={t("Status")}
          description={t("Indicates the current operational status of the trailer.")}
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          rules={{ required: true }}
          name="code"
          label={t("Code")}
          placeholder={t("Code")}
          description={t("A unique code identifying the trailer.")}
          maxLength={50}
        />
      </FormControl>
      <FormControl>
        <EquipmentTypeAutocompleteField<Trailer>
          name="equipmentTypeId"
          control={control}
          label={t("Equipment Type")}
          rules={{ required: true }}
          placeholder={t("Equipment Type")}
          description={t("The type of equipment the trailer is categorized under.")}
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
          label={t("Equip. Manufacturer")}
          rules={{ required: true }}
          placeholder={t("Equip. Manufacturer")}
          description={t("The manufacturer of the trailer's equipment.")}
          extraSearchParams={{
            status: statusSchema.enum.Active,
          }}
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          name="model"
          label={t("Model")}
          rules={{ required: true }}
          placeholder={t("Model")}
          description={t("The specific model of the trailer.")}
          maxLength={50}
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          name="make"
          label={t("Make")}
          rules={{ required: true }}
          placeholder={t("Make")}
          description={t("The manufacturer of the trailer.")}
          maxLength={50}
        />
      </FormControl>
      <FormControl>
        <NumberField
          control={control}
          name="year"
          label={t("Year")}
          rules={{ required: true }}
          placeholder={t("Year")}
          description={t("The production year of the trailer.")}
        />
      </FormControl>
      <FormControl>
        <NumberField
          control={control}
          name="maxLoadWeight"
          label={t("Max Load Weight")}
          sideText="lbs"
          placeholder={t("Max Load Weight")}
          description={t("The maximum load weight the trailer can carry.")}
        />
      </FormControl>
      <FormControl cols="full">
        <FleetCodeAutocompleteField<Trailer>
          name="fleetCodeId"
          control={control}
          clearable
          label={t("Fleet Code")}
          placeholder={t("Fleet Code")}
          description={t("The fleet code associated with the trailer.")}
        />
      </FormControl>
    </FormGroup>
  );
}

function RegistrationInformationSecond({ control }: { control: Control<Trailer> }) {
  const t = useT();

  return (
    <FormSection title={t("Registration Information")} className="border-t py-2">
      <FormGroup cols={2}>
        <FormControl>
          <InputField
            control={control}
            name="vin"
            label={t("VIN")}
            placeholder={t("VIN")}
            description={t("The Vehicle Identification Number (VIN) of the trailer.")}
            maxLength={17}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="registrationNumber"
            label={t("Registration Number")}
            placeholder={t("Registration Number")}
            description={t("The unique registration number assigned to the trailer.")}
            maxLength={50}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="externalId"
            label={t("Samsara Asset ID")}
            placeholder={t("Samsara Asset ID")}
            description={t(
              "Links the trailer to its Samsara asset for telematics. Auto-matched by VIN when left blank.",
            )}
          />
        </FormControl>
        <FormControl>
          <UsStateAutocompleteField
            control={control}
            name="registrationStateId"
            label={t("Registration State")}
            placeholder={t("Registration State")}
            description={t("The U.S. state where the trailer is registered.")}
          />
        </FormControl>
        <FormControl>
          <AutoCompleteDateField
            control={control}
            name="registrationExpiry"
            label={t("Registration Expiry")}
            description={t("The expiration date of the trailer's registration.")}
            placeholder={t("Registration Expiry")}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="licensePlateNumber"
            label={t("License Plate Number")}
            placeholder={t("License Plate Number")}
            description={t("The license plate number associated with the trailer.")}
            maxLength={50}
          />
        </FormControl>
        <FormControl>
          <AutoCompleteDateField
            control={control}
            clearable
            name="lastInspectionDate"
            label={t("Last Inspection Date")}
            description={t("The date of the trailer's most recent inspection.")}
            placeholder={t("Last Inspection Date")}
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
