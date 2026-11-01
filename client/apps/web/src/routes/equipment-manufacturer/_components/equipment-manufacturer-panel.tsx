import { useT } from "@trenova/shared/i18n/use-t";
import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import { apiService } from "@/services/api";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
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
  const t = useT();

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
        queryKey="equipment-manufacturer-list"
        title={t("Equipment manufacturer")}
        fieldKey="name"
        formComponent={<EquipmentManufacturerForm />}
        mutationFn={(values, currentRow) => {
          if (!currentRow.id) {
            throw new Error("No Equipment Manufacturer ID selected");
          }

          return apiService.equipmentManufacturerService.update(currentRow.id, values);
        }}
      />
    );
  }

  return (
    <FormCreatePanel
      open={open}
      onOpenChange={onOpenChange}
      form={form}
      queryKey="equipment-manufacturer-list"
      title={t("Equipment manufacturer")}
      formComponent={<EquipmentManufacturerForm />}
      mutationFn={(values) => apiService.equipmentManufacturerService.create(values)}
    />
  );
}
