import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { coreResponsibilityChoices, fieldSensitivityChoices } from "@/lib/choices";
import type { Role } from "@trenova/shared/types/role";
import { useFormContext } from "react-hook-form";

export function RoleForm({ isSystemRole }: { isSystemRole?: boolean }) {
  const { control } = useFormContext<Role>();

  return (
    <FormGroup cols={2}>
      <FormControl>
        <InputField
          control={control}
          rules={{ required: true }}
          name="name"
          label="Name"
          placeholder="Enter role name"
          disabled={isSystemRole}
        />
      </FormControl>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="maxSensitivity"
          label="Max Sensitivity Level"
          options={fieldSensitivityChoices}
          isReadOnly={isSystemRole}
        />
      </FormControl>
      <FormControl>
        <SelectField
          control={control}
          name="coreResponsibility"
          label="Core Responsibility"
          options={coreResponsibilityChoices}
          isClearable
          isReadOnly={isSystemRole}
          placeholder="Select responsibility..."
        />
      </FormControl>
      <FormControl cols="full">
        <TextareaField
          control={control}
          name="description"
          label="Description"
          placeholder="Describe the purpose of this role"
          disabled={isSystemRole}
        />
      </FormControl>
    </FormGroup>
  );
}
