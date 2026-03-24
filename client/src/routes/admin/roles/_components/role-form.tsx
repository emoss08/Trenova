import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup } from "@/components/ui/form";
import { fieldSensitivityChoices } from "@/lib/choices";
import type { Role } from "@/types/role";
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
      <FormControl cols="full">
        <TextareaField
          control={control}
          name="description"
          label="Description"
          placeholder="Describe the purpose of this role"
          disabled={isSystemRole}
        />
      </FormControl>
      <FormControl cols="full">
        <SwitchField
          control={control}
          name="isBusinessUnitAdmin"
          label="Business Unit Administrator"
          description="Grants admin-level access across all organizations in this business unit."
          disabled={isSystemRole}
          outlined
          position="left"
        />
      </FormControl>
    </FormGroup>
  );
}
