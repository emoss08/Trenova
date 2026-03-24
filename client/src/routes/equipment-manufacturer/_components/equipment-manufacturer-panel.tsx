import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import type { DataTablePanelProps } from "@/types/data-table";
import {
  equipmentManufacturerSchema,
  type EquipmentManufacturer,
} from "@/types/equipment-manufacturer";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { EquipmentManufacturerForm } from "./equipment-manufacturer-form";

export function EquipmentManufacturerPanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<EquipmentManufacturer>) {
  const form = useForm({
    resolver: zodResolver(equipmentManufacturerSchema),
    defaultValues: {
      status: "Active",
      name: "",
      description: "",
    },
  });

  if (mode === "edit") {
    return (
      <FormEditPanel
        open={open}
        onOpenChange={onOpenChange}
        row={row}
        form={form}
        url="/equipment-manufacturers/"
        queryKey="equipment-manufacturer-list"
        title="Equipment Manufacturer"
        fieldKey="name"
        formComponent={<EquipmentManufacturerForm />}
      />
    );
  }

  return (
    <FormCreatePanel
      open={open}
      onOpenChange={onOpenChange}
      form={form}
      url="/equipment-manufacturers/"
      queryKey="equipment-manufacturer-list"
      title="Equipment Manufacturer"
      formComponent={<EquipmentManufacturerForm />}
    />
  );
}
