import { useT } from "@trenova/shared/i18n/use-t";
import { ColorField } from "@/components/fields/color-field";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { fieldTypeChoices } from "@/lib/choices";
import { apiService } from "@/services/api";
import type { CustomFieldDefinition } from "@/types/custom-field";
import { useQuery } from "@tanstack/react-query";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { useFormContext, useWatch } from "react-hook-form";
import { SelectOptionsField } from "./select-options-field";

export function CustomFieldDefinitionForm() {
  const t = useT();

  const { control } = useFormContext<CustomFieldDefinition>();
  const fieldType = useWatch({ control, name: "fieldType" });

  const { data: resourceTypes } = useQuery({
    queryKey: ["custom-field-resource-types"],
    queryFn: () => apiService.customFieldService.getResourceTypes(),
  });

  const resourceTypeChoices = (resourceTypes?.resourceTypes || []).map((rt) => ({
    value: rt,
    label: rt.charAt(0).toUpperCase() + rt.slice(1),
  }));

  const showOptionsField = fieldType === "select" || fieldType === "multiSelect";

  return (
    <FormGroup cols={2}>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="resourceType"
          label={t("Resource Type")}
          placeholder={t("Select resource type")}
          description={t("The entity type this field applies to")}
          options={resourceTypeChoices}
        />
      </FormControl>
      <FormControl>
        <SelectField
          control={control}
          rules={{ required: true }}
          name="fieldType"
          label={t("Field Type")}
          placeholder={t("Select field type")}
          description={t("The data type for this field")}
          options={fieldTypeChoices}
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          rules={{ required: true }}
          name="name"
          label={t("Name")}
          placeholder="field_name"
          description={t("Internal name (lowercase, underscores only)")}
          maxLength={100}
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          rules={{ required: true }}
          name="label"
          label={t("Label")}
          placeholder={t("Display Label")}
          description={t("Display label shown to users")}
          maxLength={150}
        />
      </FormControl>
      <FormControl cols="full">
        <TextareaField
          control={control}
          name="description"
          label={t("Description")}
          placeholder={t("Optional description")}
          description={t("Help text for this field")}
        />
      </FormControl>
      <FormControl>
        <SwitchField
          control={control}
          name="isRequired"
          label={t("Required")}
          outlined
          description={t("Users must provide a value")}
        />
      </FormControl>
      <FormControl>
        <SwitchField
          control={control}
          name="isActive"
          label={t("Active")}
          outlined
          description={t("Field is visible and usable")}
        />
      </FormControl>
      <FormControl>
        <InputField
          control={control}
          name="displayOrder"
          label={t("Display Order")}
          type="number"
          placeholder="0"
          description={t("Sort order for display")}
        />
      </FormControl>
      <FormControl>
        <ColorField
          hideHeader
          control={control}
          name="color"
          label={t("Color")}
          description={t("Optional color for visual distinction")}
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
