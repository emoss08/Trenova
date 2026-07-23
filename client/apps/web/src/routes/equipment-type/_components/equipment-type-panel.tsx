import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import { apiService } from "@/services/api";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { equipmentTypeSchema, type EquipmentType } from "@/types/equipment-type";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { EquipTypeForm } from "./equipment-type-form";

export function EquipmentTypePanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<EquipmentType>) {
  const form = useForm({
    resolver: zodResolver(equipmentTypeSchema),
    defaultValues: {
      status: "Active",
      code: "",
      description: "",
      class: "Tractor",
      color: "",
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
        queryKey="equipment-type-list"
        title="Equipment Type"
        fieldKey="code"
        formComponent={<EquipTypeForm />}
        mutationFn={(values, currentRow) => {
          if (!currentRow.id) {
            throw new Error("No Equipment Type ID selected");
          }

          return apiService.equipmentTypeService.update(currentRow.id, values);
        }}
      />
    );
  }

  return (
    <FormCreatePanel
      open={open}
      onOpenChange={onOpenChange}
      form={form}
      queryKey="equipment-type-list"
      title="Equipment Type"
      formComponent={<EquipTypeForm />}
      mutationFn={(values) => apiService.equipmentTypeService.create(values)}
    />
  );
}
