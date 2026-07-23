import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import {
  customFieldDefinitionSchema,
  type CustomFieldDefinition,
} from "@/types/custom-field";
import type { DataTablePanelProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { CustomFieldDefinitionForm } from "./custom-field-definition-form";

export function CustomFieldDefinitionPanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<CustomFieldDefinition>) {
  const form = useForm({
    resolver: zodResolver(customFieldDefinitionSchema),
    defaultValues: {
      resourceType: "trailer",
      name: "",
      label: "",
      description: "",
      fieldType: "text",
      isRequired: false,
      isActive: true,
      displayOrder: 0,
      color: "",
      options: [],
      validationRules: null,
      defaultValue: null,
      uiAttributes: null,
    },
    mode: "onChange",
  });

  if (mode === "edit") {
    return (
      <FormEditPanel
        open={open}
        onOpenChange={onOpenChange}
        row={row}
        form={form}
        url="/custom-fields/definitions/"
        queryKey="custom-field-definition-list"
        title="Custom Field Definition"
        fieldKey="label"
        formComponent={<CustomFieldDefinitionForm />}
      />
    );
  }

  return (
    <FormCreatePanel
      open={open}
      onOpenChange={onOpenChange}
      form={form}
      url="/custom-fields/definitions/"
      queryKey="custom-field-definition-list"
      title="Custom Field Definition"
      formComponent={<CustomFieldDefinitionForm />}
    />
  );
}
