import { useT } from "@trenova/shared/i18n/use-t";
import { RoleSelectAutocompleteField } from "@/components/autocomplete-fields";
import { InputField } from "@/components/fields/input-field";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import type { SCIMGroupRoleMappingFormValues } from "@trenova/shared/types/iam";
import { useFormContext } from "react-hook-form";

export function SCIMGroupMappingForm() {
  const t = useT();

  const { control } = useFormContext<SCIMGroupRoleMappingFormValues>();

  return (
    <FormSection title={t("Group Mapping")}>
      <FormGroup cols={2}>
        <FormControl cols="full">
          <InputField
            control={control}
            rules={{ required: true }}
            name="externalGroupId"
            label={t("External Group ID")}
            placeholder={t("00g1abcd2EFGH3ijk4l5")}
            description={t("Immutable group identifier sent by the external SCIM directory.")}
            maxLength={160}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name="displayName"
            label={t("Display Name")}
            placeholder={t("Operations Managers")}
            description={t("Readable group name shown in provisioning reviews.")}
            maxLength={160}
          />
        </FormControl>
        <FormControl>
          <RoleSelectAutocompleteField<SCIMGroupRoleMappingFormValues>
            control={control}
            name="roleId"
            label={t("Role")}
            placeholder={t("Select role")}
            description={t("Application role assigned to users in this external group.")}
            rules={{ required: true }}
          />
        </FormControl>
      </FormGroup>
    </FormSection>
  );
}
