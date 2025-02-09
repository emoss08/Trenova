import { AsyncSelectField } from "@/components/fields/async-select";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { FormControl, FormGroup } from "@/components/ui/form";
import { equipmentStatusChoices } from "@/lib/choices";
import { type TractorSchema } from "@/lib/schemas/tractor-schema";
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
        <AsyncSelectField
          name="fleetCodeId"
          control={control}
          link="/fleet-codes/"
          label="Fleet Code"
          placeholder="Fleet Code"
          description="Select the fleet code of the tractor"
          hasPermission
          hasPopoutWindow
          popoutLink="/dispatch/configurations/fleet-codes/"
          popoutLinkLabel="Fleet Code"
          valueKey="name"
        />
      </FormControl>
      <FormControl>
        <AsyncSelectField
          name="equipmentTypeId"
          control={control}
          rules={{ required: true }}
          link="/equipment-types/"
          label="Equipment Type"
          valueKey="code"
          placeholder="Equipment Type"
          description="Select the equipment type of the tractor"
          hasPermission
          hasPopoutWindow
          popoutLink="/equipment/configurations/equipment-types/"
          popoutLinkLabel="Equipment Type"
        />
      </FormControl>
      <FormControl>
        <AsyncSelectField
          name="equipmentManufacturerId"
          control={control}
          rules={{ required: true }}
          link="/equipment-manufacturers/"
          label="Equipment Manufacturer"
          valueKey="name"
          placeholder="Equipment Manufacturer"
          description="Select the equipment manufacturer of the tractor"
          hasPermission
          hasPopoutWindow
          popoutLink="/equipment/configurations/equipment-manufacturers/"
          popoutLinkLabel="Equip Manu."
        />
      </FormControl>
      <FormControl>
        <AsyncSelectField
          name="primaryWorkerId"
          control={control}
          rules={{ required: true }}
          link="/workers/"
          label="Primary Worker"
          valueKey={["firstName", "lastName"]}
          placeholder="Primary Worker"
          description="Select the primary worker of the tractor"
          hasPermission
          hasPopoutWindow
          popoutLink="/dispatch/configurations/workers/"
          popoutLinkLabel="Worker"
        />
      </FormControl>
      <FormControl>
        <AsyncSelectField
          name="secondaryWorkerId"
          control={control}
          link="/workers/"
          label="Secondary Worker"
          valueKey={["firstName", "lastName"]}
          placeholder="Secondary Worker"
          description="Select the secondary worker of the tractor"
          hasPermission
          hasPopoutWindow
          popoutLink="/dispatch/configurations/workers/"
          popoutLinkLabel="Worker"
          isClearable
        />
      </FormControl>
    </FormGroup>
  );
}
