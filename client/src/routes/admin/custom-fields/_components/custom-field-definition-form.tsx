import { ColorField } from "@/components/fields/color-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { FormControl, FormGroup } from "@/components/ui/form";
import { fieldTypeChoices } from "@/lib/choices";
import { apiService } from "@/services/api";
import type { CustomFieldDefinition } from "@/types/custom-field";
import { useQuery } from "@tanstack/react-query";
import { useFormContext, useWatch } from "react-hook-form";
import { SelectOptionsField } from "./select-options-field";

export function CustomFieldDefinitionForm() {
  const { control } = useFormContext<CustomFieldDefinition>();
  const fieldType = useWatch({ control, name: "fieldType" });

  const { data: resourceTypes } = useQuery({
    queryKey: ["custom-field-resource-types"],
    queryFn: () => apiService.customFieldService.getResourceTypes(),
  });

  const resourceTypeChoices = (resourceTypes?.resourceTypes || []).map(
    (rt) => ({
      value: rt,
      label: rt.charAt(0).toUpperCase() + rt.slice(1),
    }),
  );

  const showOptionsField =
    fieldType === "select" || fieldType === "multiSelect";

  return (
    <FormGroup cols={2}>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="resourceType"
          label="Resource Type"
          placeholder="Select resource type"
          description="The entity type this field applies to"
          options={resourceTypeChoices}
        />
      </FormControl>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="fieldType"
          label="Field Type"
          placeholder="Select field type"
          description="The data type for this field"
          options={fieldTypeChoices}
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          rules={{ required: true }}
          name="name"
          label="Name"
          placeholder="field_name"
          description="Internal name (lowercase, underscores only)"
          maxLength={100}
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          rules={{ required: true }}
          name="label"
          label="Label"
          placeholder="Display Label"
          description="Display label shown to users"
          maxLength={150}
        />
      </FormControl>
      <FormControl cols="full">
        <TextareaField
          control={control}
          name="description"
          label="Description"
          placeholder="Optional description"
          description="Help text for this field"
        />
      </FormControl>
      <FormControl>
        <SwitchField
          control={control}
          name="isRequired"
          label="Required"
          description="Users must provide a value"
        />
      </FormControl>
      <FormControl>
        <SwitchField
          control={control}
          name="isActive"
          label="Active"
          description="Field is visible and usable"
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          name="displayOrder"
          label="Display Order"
          type="number"
          placeholder="0"
          description="Sort order for display"
        />
      </FormControl>
      <FormControl>
        <ColorField
          hideHeader
          control={control}
          name="color"
          label="Color"
          description="Optional color for visual distinction"
        />
      </FormControl>
      {showOptionsField && (
        <FormControl cols="full" className="mt-2">
          <SelectOptionsField control={control} />
        </FormControl>
      )}
    </FormGroup>
  );
}
