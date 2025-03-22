import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup } from "@/components/ui/form";
import { statusChoices } from "@/lib/choices";
import { type EquipmentManufacturerSchema } from "@/lib/schemas/equipment-manufacturer-schema";
import { useFormContext } from "react-hook-form";

export function EquipManufacturerForm() {
  const { control } = useFormContext<EquipmentManufacturerSchema>();

  return (
    <FormGroup cols={2}>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="status"
          label="Status"
          placeholder="Status"
          description="The status of the equipment manufacturer"
          options={statusChoices}
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          rules={{ required: true }}
          name="name"
          label="Name"
          placeholder="Name"
          description="The name of the equipment manufacturer"
        />
      </FormControl>
      <FormControl cols="full">
        <TextareaField
          control={control}
          name="description"
          label="Description"
          placeholder="Description"
          description="The description of the equipment manufacturer"
        />
      </FormControl>
    </FormGroup>
  );
}
