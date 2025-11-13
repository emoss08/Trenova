import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { AccountTypeAutocompleteField } from "@/components/ui/autocomplete-fields";
import { FormControl, FormGroup } from "@/components/ui/form";
import { statusChoices } from "@/lib/choices";
import { GLAccountSchema } from "@/lib/schemas/gl-account-schema";
import { Control, useFormContext } from "react-hook-form";

function GeneralInformationSection({
  control,
}: {
  control: Control<GLAccountSchema>;
}) {
  return (
    <FormGroup cols={2}>
      <FormControl>
        <SelectField
          control={control}
          options={statusChoices}
          rules={{ required: true }}
          name="status"
          label="Status"
          placeholder="Status"
          description="Indicates the current operational status of the GL account."
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          rules={{ required: true }}
          name="accountCode"
          label="Account Code"
          placeholder="Code"
          description="A unique code identifying the trailer."
          maxLength={50}
        />
      </FormControl>
      <FormControl>
        <AccountTypeAutocompleteField<GLAccountSchema>
          name="accountTypeId"
          control={control}
          label="Account Type"
          rules={{ required: true }}
          placeholder="Account Type"
          description="The type of account the GL account is categorized under."
        />
      </FormControl>
    </FormGroup>
  );
}

export function GLAccountForm() {
  const { control } = useFormContext<GLAccountSchema>();

  return (
    <>
      <GeneralInformationSection control={control} />
    </>
  );
}
