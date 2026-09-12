import { useT } from "@trenova/shared/i18n/use-t";
import { ColorField } from "@/components/fields/color-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { accountCategoryChoices, statusChoices } from "@/lib/choices";
import type { AccountType } from "@/types/account-type";
import { useFormContext } from "react-hook-form";

export function AccountTypeForm() {
  const t = useT();

  const { control } = useFormContext<AccountType>();

  return (
    <FormGroup cols={2}>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="status"
          label={t("Status")}
          placeholder={t("Status")}
          description={t("The status of the account type")}
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
          description={t("The code of the account type")}
          maxLength={10}
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          rules={{ required: true }}
          name="name"
          label={t("Name")}
          placeholder={t("Name")}
          description={t("The name of the account type")}
          maxLength={100}
        />
      </FormControl>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="category"
          label={t("Category")}
          placeholder={t("Category")}
          description={t("The category of the account type")}
          options={accountCategoryChoices}
        />
      </FormControl>
      <FormControl cols="full">
        <TextareaField
          control={control}
          name="description"
          label={t("Description")}
          placeholder={t("Description")}
          description={t("The description of the account type")}
        />
      </FormControl>
      <FormControl cols="full">
        <ColorField
          control={control}
          name="color"
          label={t("Color")}
          description={t("The color of the account type")}
        />
      </FormControl>
    </FormGroup>
  );
}
