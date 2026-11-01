import { useT } from "@trenova/shared/i18n/use-t";
import { customField } from "@/lib/queries/custom-field";
import type { CustomFieldFormFieldsQuery } from "@trenova/graphql/generated/graphql";
import { useQuery } from "@tanstack/react-query";
import { type Control, type FieldValues, type Path } from "react-hook-form";
import { AutoCompleteDateField } from "./fields/date-field/date-field";
import { InputField } from "./fields/input-field";
import { NumberField } from "./fields/number-field";
import { SelectField } from "./fields/select-field";
import { SwitchField } from "./fields/switch-field";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";

interface CustomFieldsSectionProps<T extends FieldValues> {
  resourceType: string;
  control: Control<T>;
  fieldPrefix?: string;
}

type CustomFieldFormField = CustomFieldFormFieldsQuery["customFieldFormFields"][number];

function mapSelectOptions(options: CustomFieldFormField["options"]) {
  return options.map((opt) => ({
    value: opt.value,
    label: opt.label,
    color: opt.color,
  }));
}

function CustomFieldRenderer<T extends FieldValues>({
  definition,
  control,
  fieldPrefix,
}: {
  definition: CustomFieldFormField;
  control: Control<T>;
  fieldPrefix: string;
}) {
  const t = useT();

  const fieldName = `${fieldPrefix}.${definition.id}` as Path<T>;
  const rules = {
    required: definition.isRequired ? t("{0} is required", definition.label) : false,
  };

  switch (definition.fieldType) {
    case "text":
      return (
        <InputField
          control={control}
          name={fieldName}
          label={t(definition.label)}
          description={t(definition.description)}
          placeholder={definition.uiAttributes?.placeholder || definition.label}
          rules={rules}
          maxLength={definition.validationRules?.maxLength ?? undefined}
        />
      );

    case "number":
      // A custom field's kind lives in its definition, not in the form's static
      // shape, so the field settles number vs decimal string from the value.
      return (
        <NumberField<FieldValues>
          control={control as Control<FieldValues>}
          name={fieldName}
          label={t(definition.label)}
          description={t(definition.description)}
          placeholder={definition.uiAttributes?.placeholder || definition.label}
          rules={rules}
        />
      );

    case "date":
      return (
        <AutoCompleteDateField
          control={control}
          name={fieldName}
          label={t(definition.label)}
          description={t(definition.description)}
          placeholder={definition.uiAttributes?.placeholder || definition.label}
          rules={rules}
        />
      );

    case "boolean":
      return (
        <SwitchField
          control={control}
          name={fieldName}
          label={t(definition.label)}
          description={t(definition.description)}
        />
      );

    case "select":
      return (
        <SelectField
          control={control}
          name={fieldName}
          label={t(definition.label)}
          description={t(definition.description)}
          placeholder={definition.uiAttributes?.placeholder || definition.label}
          rules={rules}
          options={mapSelectOptions(definition.options)}
        />
      );

    case "multiSelect":
      return (
        <SelectField
          control={control}
          name={fieldName}
          label={t(definition.label)}
          description={t(definition.description)}
          placeholder={definition.uiAttributes?.placeholder || definition.label}
          rules={rules}
          options={mapSelectOptions(definition.options)}
        />
      );

    default:
      return null;
  }
}

export function CustomFieldsSection<T extends FieldValues>({
  resourceType,
  control,
  fieldPrefix = "customFields",
}: CustomFieldsSectionProps<T>) {
  const t = useT();

  const { data: fields, isLoading } = useQuery({
    ...customField.formFields(resourceType),
    staleTime: 5 * 60 * 1000,
  });

  if (isLoading || !fields || fields.length === 0) {
    return null;
  }

  return (
    <FormSection title={t("Custom fields")}>
      <FormGroup cols={2}>
        {fields.map((definition) => (
          <FormControl key={definition.id} cols={definition.fieldType === "boolean" ? "full" : 1}>
            <CustomFieldRenderer
              definition={definition}
              control={control}
              fieldPrefix={fieldPrefix}
            />
          </FormControl>
        ))}
      </FormGroup>
    </FormSection>
  );
}
