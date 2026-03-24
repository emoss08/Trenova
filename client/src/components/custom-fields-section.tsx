import { apiService } from "@/services/api";
import type { CustomFieldDefinition, SelectOption } from "@/types/custom-field";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { type Control, type FieldValues, type Path } from "react-hook-form";
import { AutoCompleteDateField } from "./fields/date-field/date-field";
import { InputField } from "./fields/input-field";
import { NumberField } from "./fields/number-field";
import { SelectField } from "./fields/select-field";
import { SwitchField } from "./fields/switch-field";
import { FormControl, FormGroup, FormSection } from "./ui/form";

interface CustomFieldsSectionProps<T extends FieldValues> {
  resourceType: string;
  control: Control<T>;
  fieldPrefix?: string;
}

function mapSelectOptions(options: SelectOption[]) {
  return options.map((opt) => ({
    value: opt.value,
    label: opt.label,
    color: opt.color,
  }));
}

function CustomFieldRenderer<T extends FieldValues>({
  definition,
  control,
  fieldPrefix = "customFields",
}: {
  definition: CustomFieldDefinition;
  control: Control<T>;
  fieldPrefix: string;
}) {
  const fieldName = `${fieldPrefix}.${definition.id}` as Path<T>;
  const rules = {
    required: definition.isRequired ? `${definition.label} is required` : false,
  };

  switch (definition.fieldType) {
    case "text":
      return (
        <InputField
          control={control}
          name={fieldName}
          label={definition.label}
          description={definition.description}
          placeholder={definition.uiAttributes?.placeholder || definition.label}
          rules={rules}
          maxLength={definition.validationRules?.maxLength}
        />
      );

    case "number":
      return (
        <NumberField
          control={control}
          name={fieldName}
          label={definition.label}
          description={definition.description}
          placeholder={definition.uiAttributes?.placeholder || definition.label}
          rules={rules}
        />
      );

    case "date":
      return (
        <AutoCompleteDateField
          control={control}
          name={fieldName}
          label={definition.label}
          description={definition.description}
          placeholder={definition.uiAttributes?.placeholder || definition.label}
          rules={rules}
        />
      );

    case "boolean":
      return (
        <SwitchField
          control={control}
          name={fieldName}
          label={definition.label}
          description={definition.description}
        />
      );

    case "select":
      return (
        <SelectField
          control={control}
          name={fieldName}
          label={definition.label}
          description={definition.description}
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
          label={definition.label}
          description={definition.description}
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
  const { data: customFields, isLoading } = useQuery({
    queryKey: ["custom-fields", resourceType],
    queryFn: () =>
      apiService.customFieldService.getByResourceType(resourceType),
    staleTime: 5 * 60 * 1000,
  });

  const sortedFields = useMemo(() => {
    if (!customFields) return [];
    return [...customFields]
      .filter((field) => field.isActive)
      .sort((a, b) => a.displayOrder - b.displayOrder);
  }, [customFields]);

  if (isLoading) {
    return null;
  }

  if (!sortedFields || sortedFields.length === 0) {
    return null;
  }

  return (
    <FormSection title="Custom Fields" className="border-t pt-2">
      <FormGroup cols={2}>
        {sortedFields.map((definition) => (
          <FormControl
            key={definition.id}
            cols={definition.fieldType === "boolean" ? "full" : 1}
          >
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
