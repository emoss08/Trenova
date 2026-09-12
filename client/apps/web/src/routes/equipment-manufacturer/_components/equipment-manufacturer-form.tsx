import { useT } from "@trenova/shared/i18n/use-t";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { statusChoices } from "@/lib/choices";
import type { EquipmentManufacturer } from "@/types/equipment-manufacturer";
import { useFormContext } from "react-hook-form";

export function EquipmentManufacturerForm() {
  const t = useT();

  const { control } = useFormContext<EquipmentManufacturer>();

  return (
    <FormGroup cols={2}>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="status"
          label={t("Status")}
          placeholder={t("Status")}
          description={t("The status of the equipment manufacturer")}
          options={statusChoices}
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          rules={{ required: true }}
          name="name"
          label={t("Name")}
          placeholder={t("Name")}
          description={t("The name of the equipment manufacturer")}
          maxLength={100}
        />
      </FormControl>
      <FormControl cols="full">
        <TextareaField
          control={control}
          name="description"
          label={t("Description")}
          placeholder={t("Description")}
          description={t("The description of the equipment manufacturer")}
        />
      </FormControl>
    </FormGroup>
  );
}
