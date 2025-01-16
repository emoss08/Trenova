import { FormEditModal } from "@/components/ui/form-edit-model";
import {
  equipmentManufacturerSchema,
  EquipmentManufacturerSchema,
} from "@/lib/schemas/equipment-manufacturer-schema";
import { type EditTableSheetProps } from "@/types/data-table";
import { yupResolver } from "@hookform/resolvers/yup";
import { useForm } from "react-hook-form";
import { EquipManufacturerForm } from "./equip-menufacturer-form";

export function EditEquipManufacturerModal({
  open,
  onOpenChange,
  currentRecord,
}: EditTableSheetProps<EquipmentManufacturerSchema>) {
  const form = useForm<EquipmentManufacturerSchema>({
    resolver: yupResolver(equipmentManufacturerSchema),
    defaultValues: currentRecord,
  });

  return (
    <FormEditModal
      currentRecord={currentRecord}
      open={open}
      onOpenChange={onOpenChange}
      url="/equipment-manufacturers/"
      title="Equipment Manufacturer"
      queryKey="equip-manufacturer-list"
      formComponent={<EquipManufacturerForm />}
      fieldKey="name"
      form={form}
      schema={equipmentManufacturerSchema}
    />
  );
}
