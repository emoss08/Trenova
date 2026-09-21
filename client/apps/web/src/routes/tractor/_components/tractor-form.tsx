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
    <FormGroup cols={2}>
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
          label={t("Equipment type")}
          rules={{ required: true }}
          placeholder={t("Equipment type")}
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
          label={t("Equip. manufacturer")}
          rules={{ required: true }}
          placeholder={t("Equip. manufacturer")}
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
          label={t("Fleet code")}
          placeholder={t("Fleet code")}
          description={t("The fleet code associated with the tractor.")}
        />
      </FormControl>
    </FormGroup>
  );
}

function RegistrationInformationSection({ control }: { control: Control<Tractor> }) {
  const t = useT();

  return (
    <FormSection title={t("Registration information")}>
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
            label={t("Registration number")}
            placeholder={t("Registration number")}
            description={t("The unique registration number assigned to the tractor.")}
            maxLength={50}
          />
        </FormControl>
        <FormControl>
          <UsStateAutocompleteField
            control={control}
            name="stateId"
            label={t("License state")}
            placeholder={t("License state")}
            description={t("The U.S. state where the tractor is licensed.")}
          />
        </FormControl>
        <FormControl>
          <AutoCompleteDateField
            control={control}
            name="registrationExpiry"
            label={t("Registration expiry")}
            description={t("The expiration date of the tractor's registration.")}
            placeholder={t("Registration expiry")}
          />
        </FormControl>
        <FormControl cols="full">
          <InputField
            control={control}
            name="licensePlateNumber"
            label={t("License plate number")}
            placeholder={t("License plate number")}
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
    <FormSection title={t("Telematics")}>
      <FormGroup cols={1}>
        <FormControl cols="full">
          <InputField
            control={control}
            name="externalId"
            label={t("Samsara vehicle ID")}
            placeholder={t("Samsara vehicle ID")}
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
    <FormSection title={t("Fuel & IFTA")}>
      <FormGroup cols={1}>
        <FormControl>
          <IftaFuelTypeAutocompleteField<Tractor>
            control={control}
            name="fuelType"
            label={t("Fuel type")}
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
    <FormSection title={t("Worker assignment")}>
      <FormGroup cols={2}>
        <FormControl>
          <WorkerAutocompleteField<Tractor>
            name="primaryWorkerId"
            control={control}
            label={t("Primary worker")}
            rules={{ required: true }}
            placeholder={t("Select primary worker")}
            description={t("The primary worker assigned to this tractor.")}
          />
        </FormControl>
        <FormControl>
          <WorkerAutocompleteField<Tractor>
            name="secondaryWorkerId"
            control={control}
            clearable
            label={t("Secondary worker")}
            placeholder={t("Select secondary worker")}
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
    <div className="flex flex-col gap-6">
      <GeneralInformationSection control={control} />
      <RegistrationInformationSection control={control} />
      <TelematicsSection control={control} />
      <FuelTaxSection control={control} />
      <WorkerAssignmentSection control={control} />
      <CustomFieldsSection resourceType="tractor" control={control} />
    </div>
  );
}
