import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import type { DataTablePanelProps } from "@/types/data-table";
import {
  equipmentTypeSchema,
  type EquipmentType,
} from "@/types/equipment-type";
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
        url="/equipment-types/"
        queryKey="equipment-type-list"
        title="Equipment Type"
        fieldKey="code"
        formComponent={<EquipTypeForm />}
      />
    );
  }

  return (
    <FormCreatePanel
      open={open}
      onOpenChange={onOpenChange}
      form={form}
      url="/equipment-types/"
      queryKey="equipment-type-list"
      title="Equipment Type"
      formComponent={<EquipTypeForm />}
    />
  );
}
