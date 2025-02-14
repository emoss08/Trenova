import { AutocompleteField } from "@/components/fields/autocomplete";
import { InputField } from "@/components/fields/input-field";
import { ColorOptionValue } from "@/components/fields/select-components";
import { SelectField } from "@/components/fields/select-field";
import { FormControl, FormGroup } from "@/components/ui/form";
import { equipmentStatusChoices } from "@/lib/choices";
import { EquipmentManufacturerSchema } from "@/lib/schemas/equipment-manufacturer-schema";
import { EquipmentTypeSchema } from "@/lib/schemas/equipment-type-schema";
import { FleetCodeSchema } from "@/lib/schemas/fleet-code-schema";
import { type TractorSchema } from "@/lib/schemas/tractor-schema";
import { WorkerSchema } from "@/lib/schemas/worker-schema";
import { useFormContext } from "react-hook-form";

export function TractorForm() {
  const { control } = useFormContext<TractorSchema>();

  return (
    <FormGroup cols={2}>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="status"
          label="Status"
          placeholder="Status"
          description="The status of the tractors"
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
          description="The code of the tractor"
        />
      </FormControl>
      <FormControl cols="full">
        <InputField
          control={control}
          name="model"
          label="Model"
          placeholder="Model"
          description="The model of the tractor"
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          name="make"
          label="Make"
          placeholder="Make"
          description="The make of the tractor"
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          type="number"
          name="year"
          label="Year"
          placeholder="Year"
          description="The year of the tractor"
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          name="licensePlateNumber"
          label="License Plate Number"
          placeholder="License Plate Number"
          description="The license plate number of the tractor"
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          name="vin"
          label="VIN"
          placeholder="VIN"
          description="The VIN of the tractor"
        />
      </FormControl>
      <FormControl cols="full">
        <AutocompleteField<FleetCodeSchema, TractorSchema>
          name="fleetCodeId"
          control={control}
          link="/fleet-codes/"
          label="Fleet Code"
          placeholder="Fleet Code"
          description="The fleet code associated with the tractor"
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
      <FormControl>
        <AutocompleteField<EquipmentTypeSchema, TractorSchema>
          name="equipmentTypeId"
          control={control}
          link="/equipment-types/"
          label="Equipment Type"
          rules={{ required: true }}
          placeholder="Equipment Type"
          description="The type of equipment the tractor is categorized under."
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
        <AutocompleteField<EquipmentManufacturerSchema, TractorSchema>
          name="equipmentManufacturerId"
          control={control}
          link="/equipment-manufacturers/"
          label="Equip. Manufacturer"
          placeholder="Equip. Manufacturer"
          description="The manufacturer of the tractor's equipment."
          getOptionValue={(option) => option.id || ""}
          getDisplayValue={(option) => option.name}
          renderOption={(option) => option.name}
        />
      </FormControl>
      <FormControl>
        <AutocompleteField<WorkerSchema, TractorSchema>
          name="primaryWorkerId"
          control={control}
          link="/workers/"
          label="Primary Worker"
          rules={{ required: true }}
          placeholder="Select Primary Worker"
          description="Select the primary worker for the assignment."
          getOptionValue={(option) => option.id || ""}
          getDisplayValue={(option) => `${option.firstName} ${option.lastName}`}
          renderOption={(option) => `${option.firstName} ${option.lastName}`}
        />
      </FormControl>
      <FormControl>
        <AutocompleteField<WorkerSchema, TractorSchema>
          name="secondaryWorkerId"
          control={control}
          link="/workers/"
          label="Secondary Worker"
          placeholder="Select Secondary Worker"
          description="Select the secondary worker for the assignment."
          getOptionValue={(option) => option.id || ""}
          getDisplayValue={(option) => `${option.firstName} ${option.lastName}`}
          renderOption={(option) => `${option.firstName} ${option.lastName}`}
        />
      </FormControl>
    </FormGroup>
  );
}
