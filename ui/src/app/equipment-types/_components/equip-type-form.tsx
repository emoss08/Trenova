import { ColorField } from "@/components/fields/color-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup } from "@/components/ui/form";
import { equipmentClassChoices, statusChoices } from "@/lib/choices";
import { type EquipmentTypeSchema } from "@/lib/schemas/equipment-type-schema";
import { useFormContext } from "react-hook-form";

export function EquipTypeForm() {
  const { control } = useFormContext<EquipmentTypeSchema>();

  return (
    <FormGroup cols={2}>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="status"
          label="Status"
          placeholder="Status"
          description="The status of the equipment type"
          options={statusChoices}
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          rules={{ required: true }}
          name="code"
          label="Code"
          placeholder="Code"
          description="The code of the equipment type"
        />
      </FormControl>
      <FormControl cols="full">
        <TextareaField
          control={control}
          name="description"
          label="Description"
          placeholder="Description"
          description="The description of the equipment type"
        />
      </FormControl>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="class"
          label="Class"
          placeholder="Class"
          description="The class of the equipment type"
          options={equipmentClassChoices}
        />
      </FormControl>
      <FormControl>
        <ColorField
          control={control}
          name="color"
          label="Color"
          description="The color of the equipment type"
        />
      </FormControl>
    </FormGroup>
  );
}
