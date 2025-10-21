import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { SQLEditorField } from "@/components/fields/sql-editor-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup } from "@/components/ui/form";
import {
  variableContextChoices,
  variableValueTypeChoices,
} from "@/lib/choices";
import { VariableSchema } from "@/lib/schemas/variable-schema";
import { useFormContext } from "react-hook-form";

export function VariableForm() {
  const { control } = useFormContext<VariableSchema>();

  return (
    <FormGroup cols={2}>
      <FormControl cols="full">
        <SwitchField
          control={control}
          name="isActive"
          label="Enabled"
          description="Turn on to make this variable available for use in templates."
          position="left"
          outlined
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          name="key"
          label="Variable Key"
          placeholder="e.g., customerName, invoiceNumber"
          rules={{ required: true }}
          maxLength={100}
          description="Unique identifier for the variable. Use camelCase without spaces."
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          name="displayName"
          label="Display Name"
          placeholder="Enter Display Name"
          rules={{ required: true }}
          maxLength={255}
          description="Human-readable name shown in the UI."
        />
      </FormControl>
      <FormControl cols="full">
        <InputField
          control={control}
          name="category"
          label="Category"
          placeholder="e.g., Customer, Billing, System"
          rules={{ required: true }}
          maxLength={100}
          description="Group or category this variable belongs to."
        />
      </FormControl>
      <FormControl cols="full">
        <TextareaField
          control={control}
          name="description"
          label="Description"
          placeholder="Explain what this variable does and when to use it"
          description="Help text for users to understand this variable."
        />
      </FormControl>
      <FormControl cols="full">
        <SQLEditorField
          control={control}
          name="query"
          rules={{ required: true }}
          label="SQL Query"
          placeholder="-- Example: SELECT name FROM customers WHERE id = :customerId"
          description="SQL query to fetch the variable value. Use parameters like :customerId for dynamic values."
          height="250px"
        />
      </FormControl>
      <FormControl>
        <SelectField
          control={control}
          name="appliesTo"
          label="Applies To"
          placeholder="Select Context"
          description="Where this variable can be used."
          options={variableContextChoices}
          rules={{ required: true }}
        />
      </FormControl>
      <FormControl>
        <SelectField
          control={control}
          name="valueType"
          label="Value Type"
          placeholder="Select Value Type"
          description="Data type of the variable's value."
          options={variableValueTypeChoices}
          rules={{ required: true }}
        />
      </FormControl>
      <FormControl cols="full">
        <InputField
          control={control}
          name="defaultValue"
          label="Default Value"
          placeholder="Enter default value (optional)"
          description="Value to use if the query returns no results."
        />
      </FormControl>
    </FormGroup>
  );
}
