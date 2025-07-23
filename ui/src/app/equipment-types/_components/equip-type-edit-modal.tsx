/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { FormEditModal } from "@/components/ui/form-edit-modal";
import {
  equipmentTypeSchema,
  type EquipmentTypeSchema,
} from "@/lib/schemas/equipment-type-schema";
import { type EditTableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { EquipTypeForm } from "./equip-type-form";

export function EditEquipTypeModal({
  currentRecord,
}: EditTableSheetProps<EquipmentTypeSchema>) {
  const form = useForm<EquipmentTypeSchema>({
    resolver: zodResolver(equipmentTypeSchema),
    defaultValues: currentRecord,
  });

  return (
    <FormEditModal
      currentRecord={currentRecord}
      url="/equipment-types/"
      title="Equipment Type"
      queryKey="equipment-type-list"
      formComponent={<EquipTypeForm />}
      fieldKey="code"
      form={form}
    />
  );
}
