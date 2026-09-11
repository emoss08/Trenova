import { useT } from "@trenova/shared/i18n/use-t";
import { ColorField } from "@/components/fields/color-field";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { equipmentClassChoices, statusChoices } from "@/lib/choices";
import type { EquipmentType } from "@/types/equipment-type";
import { useFormContext } from "react-hook-form";

export function EquipTypeForm() {
  const t = useT();

  const { control } = useFormContext<EquipmentType>();

  return (
    <FormGroup cols={2}>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="status"
          label={t("Status")}
          placeholder={t("Status")}
          description={t("The status of the equipment type")}
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
          description={t("The code of the equipment type")}
          maxLength={10}
        />
      </FormControl>
      <FormControl cols="full">
        <TextareaField
          control={control}
          name="description"
          label={t("Description")}
          placeholder={t("Description")}
          description={t("The description of the equipment type")}
        />
      </FormControl>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="class"
          label={t("Class")}
          placeholder={t("Class")}
          description={t("The class of the equipment type")}
          options={equipmentClassChoices}
        />
      </FormControl>
      <FormControl>
        <ColorField
          control={control}
          name="color"
          label={t("Color")}
          description={t("The color of the equipment type")}
        />
      </FormControl>
      <FormControl>
        <NumberField
          control={control}
          name="interiorLength"
          label={t("Interior Length (ft)")}
          placeholder="53"
          description={t("Interior length of the trailer in feet")}
        />
      </FormControl>
    </FormGroup>
  );
}
