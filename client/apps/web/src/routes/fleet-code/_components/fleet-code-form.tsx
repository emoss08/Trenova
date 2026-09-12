import { useT } from "@trenova/shared/i18n/use-t";
import { UserAutocompleteField } from "@/components/autocomplete-fields";
import { ColorField } from "@/components/fields/color-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { statusChoices } from "@/lib/choices";
import type { FleetCode } from "@trenova/shared/types/fleet-code";
import { useFormContext } from "react-hook-form";

export function FleetCodeForm() {
  const t = useT();

  const { control } = useFormContext<FleetCode>();

  return (
    <FormGroup cols={2}>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="status"
          label={t("Status")}
          placeholder={t("Status")}
          description={t("The status of the fleet code")}
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
          description={t("The code of the fleet code")}
          maxLength={10}
        />
      </FormControl>
      <FormControl cols="full">
        <TextareaField
          control={control}
          name="description"
          label={t("Description")}
          placeholder={t("Description")}
          description={t("The description of the fleet code")}
        />
      </FormControl>
      <FormControl>
        <InputField
          type="number"
          control={control}
          name="deadheadGoal"
          label={t("Deadhead Goal")}
          placeholder={t("Deadhead Goal")}
          description={t("The deadhead goal of the fleet code")}
        />
      </FormControl>
      <FormControl>
        <InputField
          type="number"
          control={control}
          name="revenueGoal"
          label={t("Revenue Goal")}
          placeholder={t("Revenue Goal")}
          description={t("The revenue goal of the fleet code")}
        />
      </FormControl>
      <FormControl>
        <ColorField
          control={control}
          name="color"
          label={t("Color")}
          description={t("The color of the fleet code")}
        />
      </FormControl>
      <FormControl>
        <UserAutocompleteField
          rules={{ required: true }}
          name="managerId"
          control={control}
          label={t("Manager")}
          placeholder={t("Select Manager")}
          description={t("Select the manager of the fleet code")}
        />
      </FormControl>
    </FormGroup>
  );
}
