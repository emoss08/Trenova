import { useT } from "@trenova/shared/i18n/use-t";
import {
  EquipmentManufacturerAutocompleteField,
  EquipmentTypeAutocompleteField,
  FleetCodeAutocompleteField,
  IftaFuelTypeAutocompleteField,
  UsStateAutocompleteField,
  WorkerAutocompleteField,
} from "@/components/autocomplete-fields";
import { CustomFieldsSection } from "@/components/custom-fields-section";
import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { equipmentStatusChoices } from "@/lib/choices";
import { equipmentClassSchema } from "@/types/equipment-type";
import type { Tractor } from "@/types/tractor";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import { statusSchema } from "@trenova/shared/types/helpers";
import { type Control, useFormContext } from "react-hook-form";

function GeneralInformationSection({ control }: { control: Control<Tractor> }) {
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
          description={t("Indicates the current operational status of the tractor.")}
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          rules={{ required: true }}
          name="code"
          label={t("Code")}
          placeholder={t("Code")}
          description={t("A unique code identifying the tractor.")}
          maxLength={50}
        />
      </FormControl>
      <FormControl>
        <EquipmentTypeAutocompleteField<Tractor>
          name="equipmentTypeId"
          control={control}
          label={t("Equipment Type")}
          rules={{ required: true }}
          placeholder={t("Equipment Type")}
          description={t("The type of equipment the tractor is categorized under.")}
          extraSearchParams={{
            classes: [equipmentClassSchema.enum.Tractor],
          }}
        />
      </FormControl>
      <FormControl>
        <EquipmentManufacturerAutocompleteField<Tractor>
          name="equipmentManufacturerId"
          control={control}
          label={t("Equip. Manufacturer")}
          rules={{ required: true }}
          placeholder={t("Equip. Manufacturer")}
          description={t("The manufacturer of the tractor's equipment.")}
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
          placeholder={t("Model")}
          description={t("The specific model of the tractor.")}
          maxLength={50}
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          name="make"
          label={t("Make")}
          placeholder={t("Make")}
          description={t("The manufacturer of the tractor.")}
          maxLength={50}
        />
      </FormControl>
      <FormControl>
        <NumberField
          control={control}
          name="year"
          label={t("Year")}
          placeholder={t("Year")}
          description={t("The production year of the tractor.")}
        />
      </FormControl>
      <FormControl>
        <FleetCodeAutocompleteField<Tractor>
          name="fleetCodeId"
          control={control}
          clearable
          label={t("Fleet Code")}
          placeholder={t("Fleet Code")}
          description={t("The fleet code associated with the tractor.")}
        />
      </FormControl>
    </FormGroup>
  );
}

function RegistrationInformationSection({ control }: { control: Control<Tractor> }) {
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
            description={t("The Vehicle Identification Number (VIN) of the tractor.")}
            maxLength={17}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="registrationNumber"
            label={t("Registration Number")}
            placeholder={t("Registration Number")}
            description={t("The unique registration number assigned to the tractor.")}
            maxLength={50}
          />
        </FormControl>
        <FormControl>
          <UsStateAutocompleteField
            control={control}
            name="stateId"
            label={t("License State")}
            placeholder={t("License State")}
            description={t("The U.S. state where the tractor is licensed.")}
          />
        </FormControl>
        <FormControl>
          <AutoCompleteDateField
            control={control}
            name="registrationExpiry"
            label={t("Registration Expiry")}
            description={t("The expiration date of the tractor's registration.")}
            placeholder={t("Registration Expiry")}
          />
        </FormControl>
        <FormControl cols="full">
          <InputField
            control={control}
            name="licensePlateNumber"
            label={t("License Plate Number")}
            placeholder={t("License Plate Number")}
            description={t("The license plate number associated with the tractor.")}
            maxLength={50}
          />
        </FormControl>
      </FormGroup>
    </FormSection>
  );
}

function TelematicsSection({ control }: { control: Control<Tractor> }) {
  const t = useT();

  return (
    <FormSection title={t("Telematics")} className="border-t py-2">
      <FormGroup cols={1}>
        <FormControl cols="full">
          <InputField
            control={control}
            name="externalId"
            label={t("Samsara Vehicle ID")}
            placeholder={t("Samsara Vehicle ID")}
            description={t(
              "Links this tractor to its Samsara vehicle for live telematics. Leave blank to auto-match by VIN.",
            )}
            maxLength={100}
          />
        </FormControl>
      </FormGroup>
    </FormSection>
  );
}

function FuelTaxSection({ control }: { control: Control<Tractor> }) {
  const t = useT();

  return (
    <FormSection title={t("Fuel & IFTA")} className="border-t py-2">
      <FormGroup cols={1}>
        <FormControl>
          <IftaFuelTypeAutocompleteField<Tractor>
            control={control}
            name="fuelType"
            label={t("Fuel Type")}
            rules={{ required: true }}
            placeholder={t("Select a fuel type")}
            description={t(
              "The fuel this unit burns. Its miles land on this fuel type's lines of the quarterly IFTA return.",
            )}
          />
        </FormControl>
        <FormControl>
          <SwitchField
            control={control}
            name="iftaQualified"
            label={t("IFTA qualified")}
            description={t("Count this unit's miles and fuel on the quarterly return.")}
            tooltip={t(
              "Only IFTA-qualified units count toward the quarterly return. Turn this off for yard tractors, pickups under 26,001 lb GVW, and units that never leave the base jurisdiction.",
            )}
            position="left"
            outlined
          />
        </FormControl>
      </FormGroup>
    </FormSection>
  );
}

function WorkerAssignmentSection({ control }: { control: Control<Tractor> }) {
  const t = useT();

  return (
    <FormSection title={t("Worker Assignment")} className="border-t py-2">
      <FormGroup cols={2}>
        <FormControl>
          <WorkerAutocompleteField<Tractor>
            name="primaryWorkerId"
            control={control}
            label={t("Primary Worker")}
            rules={{ required: true }}
            placeholder={t("Select Primary Worker")}
            description={t("The primary worker assigned to this tractor.")}
          />
        </FormControl>
        <FormControl>
          <WorkerAutocompleteField<Tractor>
            name="secondaryWorkerId"
            control={control}
            clearable
            label={t("Secondary Worker")}
            placeholder={t("Select Secondary Worker")}
            description={t("An optional secondary worker assigned to this tractor.")}
          />
        </FormControl>
      </FormGroup>
    </FormSection>
  );
}

export function TractorForm() {
  const {
    control,
    formState: { errors },
  } = useFormContext<Tractor>();

  console.info("tractor form errors", errors);

  return (
    <>
      <GeneralInformationSection control={control} />
      <RegistrationInformationSection control={control} />
      <TelematicsSection control={control} />
      <FuelTaxSection control={control} />
      <WorkerAssignmentSection control={control} />
      <CustomFieldsSection resourceType="tractor" control={control} />
    </>
  );
}
