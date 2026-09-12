import { useT } from "@trenova/shared/i18n/use-t";
import { InputField } from "@/components/fields/input-field";
import { SwitchField } from "@/components/fields/switch-field";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import type { SCIMDirectoryFormValues } from "@trenova/shared/types/iam";
import { useFormContext } from "react-hook-form";

export function SCIMDirectoryForm() {
  const t = useT();

  const { control } = useFormContext<SCIMDirectoryFormValues>();

  return (
    <FormSection title={t("Directory Details")}>
      <FormGroup cols={2}>
        <FormControl cols="full">
          <InputField
            control={control}
            rules={{ required: true }}
            name="tenantSlug"
            label={t("Tenant Slug")}
            placeholder="acme-directory"
            description={t("Stable SCIM tenant identifier used by directory sync clients.")}
            maxLength={80}
          />
        </FormControl>
        <FormControl cols="full">
          <SwitchField
            control={control}
            name="enabled"
            label={t("Enabled")}
            description={t("Allow SCIM API calls for this directory.")}
            outlined
          />
        </FormControl>
      </FormGroup>
    </FormSection>
  );
}
