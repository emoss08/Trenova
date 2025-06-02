import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup } from "@/components/ui/form";
import { roleTypeChoices, statusChoices } from "@/lib/choices";
import { RoleSchema } from "@/lib/schemas/user-schema";
import { useFormContext } from "react-hook-form";

export function RoleForm() {
  const { control } = useFormContext<RoleSchema>();

  return (
    <FormGroup cols={2}>
      <FormControl cols="full">
        <SelectField
          control={control}
          rules={{ required: true }}
          name="status"
          label="Status"
          placeholder="Status"
          description="Current status of the role"
          options={statusChoices}
        />
      </FormControl>
      <FormControl cols="full">
        <InputField
          control={control}
          rules={{ required: true }}
          name="name"
          label="Name"
          placeholder="Name"
          description="Unique name of the role"
        />
      </FormControl>
      <FormControl cols="full">
        <SelectField
          control={control}
          rules={{ required: true }}
          name="roleType"
          label="Type"
          placeholder="Type"
          description="Type of the role"
          options={roleTypeChoices}
          isReadOnly
        />
      </FormControl>
      <FormControl cols="full">
        <TextareaField
          control={control}
          rules={{ required: true }}
          name="description"
          label="Description"
          placeholder="Description"
          description="Description of the role"
        />
      </FormControl>
    </FormGroup>
  );
}
