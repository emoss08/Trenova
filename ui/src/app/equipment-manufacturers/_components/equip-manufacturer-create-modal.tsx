import { FormCreateModal } from "@/components/ui/form-create-modal";
import {
  equipmentManufacturerSchema,
  EquipmentManufacturerSchema,
} from "@/lib/schemas/equipment-manufacturer-schema";
import { Status } from "@/types/common";
import { type TableSheetProps } from "@/types/data-table";
import { yupResolver } from "@hookform/resolvers/yup";
import { useForm } from "react-hook-form";
import { EquipManufacturerForm } from "./equip-manufacturer-form";

export function CreateEquipManufacturerModal({
  open,
  onOpenChange,
}: TableSheetProps) {
  const form = useForm<EquipmentManufacturerSchema>({
    resolver: yupResolver(equipmentManufacturerSchema),
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
      schema={equipmentManufacturerSchema}
      url="/equipment-manufacturers/"
      queryKey="equip-manufacturer-list"
      className="max-w-[400px]"
    />
  );
}
