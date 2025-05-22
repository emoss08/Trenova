import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup } from "@/components/ui/form";
import { visibilityChoices } from "@/lib/choices";
import { TableConfigurationSchema } from "@/lib/schemas/table-configuration-schema";
import { useFormContext } from "react-hook-form";

export function TableConfigurationForm() {
  const { control, register } = useFormContext<TableConfigurationSchema>();

  return (
    <FormGroup cols={2}>
      <FormControl>
        <SelectField
          control={control}
          options={visibilityChoices}
          rules={{ required: true }}
          name="visibility"
          label="Visibility"
          description="The visibility of the table configuration."
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          rules={{ required: true }}
          name="name"
          label="Name"
          placeholder="Name"
          description="The name of the table configuration."
        />
      </FormControl>
      <FormControl cols="full">
        <TextareaField
          control={control}
          name="description"
          label="Description"
          placeholder="Description"
          description="The description of the table configuration."
        />
      </FormControl>
      <FormControl cols="full">
        <SwitchField
          control={control}
          outlined
          name="isDefault"
          label="Default"
          description="When enabled, the system will automatically apply this table configuration to the table when the user first navigates to it."
          position="left"
        />
      </FormControl>
      <input type="hidden" {...register("tableConfig")} />
    </FormGroup>
  );
}
