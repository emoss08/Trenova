import { RoleSelectAutocompleteField } from "@/components/autocomplete-fields";
import { InputField } from "@/components/fields/input-field";
import { FormControl, FormGroup, FormSection } from "@/components/ui/form";
import type { SCIMGroupRoleMappingFormValues } from "@/types/iam";
import { useFormContext } from "react-hook-form";

export function SCIMGroupMappingForm() {
  const { control } = useFormContext<SCIMGroupRoleMappingFormValues>();

  return (
    <FormSection title="Group Mapping">
      <FormGroup cols={2}>
        <FormControl cols="full">
          <InputField
            control={control}
            rules={{ required: true }}
            name="externalGroupId"
            label="External Group ID"
            placeholder="00g1abcd2EFGH3ijk4l5"
            description="Immutable group identifier sent by the external SCIM directory."
            maxLength={160}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="displayName"
            label="Display Name"
            placeholder="Operations Managers"
            description="Readable group name shown in provisioning reviews."
            maxLength={160}
          />
        </FormControl>
        <FormControl>
          <RoleSelectAutocompleteField<SCIMGroupRoleMappingFormValues>
            control={control}
            name="roleId"
            label="Role"
            placeholder="Select role"
            description="Application role assigned to users in this external group."
            rules={{ required: true }}
          />
        </FormControl>
      </FormGroup>
    </FormSection>
  );
}
