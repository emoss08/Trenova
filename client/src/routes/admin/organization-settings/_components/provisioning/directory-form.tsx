import { InputField } from "@/components/fields/input-field";
import { SwitchField } from "@/components/fields/switch-field";
import { FormControl, FormGroup, FormSection } from "@/components/ui/form";
import type { SCIMDirectoryFormValues } from "@/types/iam";
import { useFormContext } from "react-hook-form";

export function SCIMDirectoryForm() {
  const { control } = useFormContext<SCIMDirectoryFormValues>();

  return (
    <FormSection title="Directory Details">
      <FormGroup cols={2}>
        <FormControl cols="full">
          <InputField
            control={control}
            rules={{ required: true }}
            name="tenantSlug"
            label="Tenant Slug"
            placeholder="acme-directory"
            description="Stable SCIM tenant identifier used by directory sync clients."
            maxLength={80}
          />
        </FormControl>
        <FormControl cols="full">
          <SwitchField
            control={control}
            name="enabled"
            label="Enabled"
            description="Allow SCIM API calls for this directory."
            outlined
          />
        </FormControl>
      </FormGroup>
    </FormSection>
  );
}
