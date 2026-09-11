import { useT } from "@trenova/shared/i18n/use-t";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { coreResponsibilityChoices, fieldSensitivityChoices } from "@/lib/choices";
import type { Role } from "@trenova/shared/types/role";
import { useFormContext } from "react-hook-form";

export function RoleForm({ isSystemRole }: { isSystemRole?: boolean }) {
  const t = useT();

  const { control } = useFormContext<Role>();

  return (
    <FormGroup cols={2}>
      <FormControl>
        <InputField
          control={control}
          rules={{ required: true }}
          name="name"
          label={t("Name")}
          placeholder={t("Enter role name")}
          disabled={isSystemRole}
        />
      </FormControl>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="maxSensitivity"
          label={t("Max Sensitivity Level")}
          options={fieldSensitivityChoices}
          isReadOnly={isSystemRole}
        />
      </FormControl>
      <FormControl>
        <SelectField
          control={control}
          name="coreResponsibility"
          label={t("Core Responsibility")}
          options={coreResponsibilityChoices}
          isClearable
          isReadOnly={isSystemRole}
          placeholder={t("Select responsibility...")}
        />
      </FormControl>
      <FormControl cols="full">
        <TextareaField
          control={control}
          name="description"
          label={t("Description")}
          placeholder={t("Describe the purpose of this role")}
          disabled={isSystemRole}
        />
      </FormControl>
    </FormGroup>
  );
}
