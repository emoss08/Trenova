/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { FormCreateModal } from "@/components/ui/form-create-modal";
import {
  equipmentManufacturerSchema,
  EquipmentManufacturerSchema,
} from "@/lib/schemas/equipment-manufacturer-schema";
import { Status } from "@/types/common";
import { type TableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { EquipManufacturerForm } from "./equip-manufacturer-form";

export function CreateEquipManufacturerModal({
  open,
  onOpenChange,
}: TableSheetProps) {
  const form = useForm<EquipmentManufacturerSchema>({
    resolver: zodResolver(equipmentManufacturerSchema),
    defaultValues: {
      name: "",
      status: Status.Active,
      description: "",
    },
  });

  return (
    <FormCreateModal
      open={open}
      onOpenChange={onOpenChange}
      title="Equipment Manufacturer"
      formComponent={<EquipManufacturerForm />}
      form={form}
      url="/equipment-manufacturers/"
      queryKey="equip-manufacturer-list"
      className="max-w-[400px]"
    />
  );
}
