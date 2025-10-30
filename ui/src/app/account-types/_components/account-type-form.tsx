import { ColorField } from "@/components/fields/color-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup } from "@/components/ui/form";
import { statusChoices } from "@/lib/choices";
import { AccountTypeSchema } from "@/lib/schemas/account-type-schema";
import { useFormContext } from "react-hook-form";

export function AccountTypeForm() {
  const { control } = useFormContext<AccountTypeSchema>();

  return (
    <FormGroup cols={2}>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="status"
          label="Status"
          placeholder="Status"
          description="The status of the account type"
          options={statusChoices}
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          rules={{ required: true }}
          name="code"
          label="Code"
          placeholder="Code"
          description="The code of the account type"
          maxLength={10}
        />
      </FormControl>
      <FormControl cols="full">
        <InputField
          control={control}
          rules={{ required: true }}
          name="name"
          label="Name"
          placeholder="Name"
          description="The name of the account type"
          maxLength={100}
        />
      </FormControl>
      <FormControl cols="full">
        <TextareaField
          control={control}
          name="description"
          label="Description"
          placeholder="Description"
          description="The description of the account type"
        />
      </FormControl>
      <FormControl cols="full">
        <ColorField
          control={control}
          name="color"
          label="Color"
          description="The color of the account type"
        />
      </FormControl>
    </FormGroup>
  );
}
