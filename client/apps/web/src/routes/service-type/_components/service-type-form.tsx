import { useT } from "@trenova/shared/i18n/use-t";
import { ColorField } from "@/components/fields/color-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { statusChoices } from "@/lib/choices";
import type { ServiceType } from "@/types/service-type";
import { useFormContext } from "react-hook-form";

export function ServiceTypeForm() {
  const t = useT();

  const { control } = useFormContext<ServiceType>();

  return (
    <FormGroup cols={2}>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="status"
          label={t("Status")}
          placeholder={t("Status")}
          description={t("The status of the service type")}
          options={statusChoices}
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          rules={{ required: true }}
          name="code"
          label={t("Code")}
          placeholder={t("Code")}
          description={t("The code of the service type")}
          maxLength={10}
        />
      </FormControl>
      <FormControl cols="full">
        <TextareaField
          control={control}
          name="description"
          label={t("Description")}
          placeholder={t("Description")}
          description={t("The description of the service type")}
        />
      </FormControl>
      <FormControl cols="full">
        <ColorField
          control={control}
          name="color"
          label={t("Color")}
          description={t("The color of the service type")}
        />
      </FormControl>
    </FormGroup>
  );
}
