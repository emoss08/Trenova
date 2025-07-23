/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { FormEditModal } from "@/components/ui/form-edit-modal";
import {
  equipmentManufacturerSchema,
  EquipmentManufacturerSchema,
} from "@/lib/schemas/equipment-manufacturer-schema";
import { type EditTableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { EquipManufacturerForm } from "./equip-manufacturer-form";

export function EditEquipManufacturerModal({
  currentRecord,
}: EditTableSheetProps<EquipmentManufacturerSchema>) {
  const form = useForm<EquipmentManufacturerSchema>({
    resolver: zodResolver(equipmentManufacturerSchema),
    defaultValues: currentRecord,
  });

  return (
    <FormEditModal
      currentRecord={currentRecord}
      url="/equipment-manufacturers/"
      title="Equipment Manufacturer"
      queryKey="equip-manufacturer-list"
      formComponent={<EquipManufacturerForm />}
      fieldKey="name"
      form={form}
    />
  );
}
