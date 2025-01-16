import { FormEditModal } from "@/components/ui/form-edit-model";
import {
  equipmentTypeSchema,
  type EquipmentTypeSchema,
} from "@/lib/schemas/equipment-type-schema";
import { type EditTableSheetProps } from "@/types/data-table";
import { yupResolver } from "@hookform/resolvers/yup";
import { useForm } from "react-hook-form";
import { EquipTypeForm } from "./equip-type-form";

export function EditEquipTypeModal({
  open,
  onOpenChange,
  currentRecord,
}: EditTableSheetProps<EquipmentTypeSchema>) {
  const form = useForm<EquipmentTypeSchema>({
    resolver: yupResolver(equipmentTypeSchema),
    defaultValues: currentRecord,
  });

  return (
    <FormEditModal
      currentRecord={currentRecord}
      open={open}
      onOpenChange={onOpenChange}
      url="/equipment-types/"
      title="Equipment Type"
      queryKey="equip-type-list"
      formComponent={<EquipTypeForm />}
      fieldKey="code"
      form={form}
      schema={equipmentTypeSchema}
    />
  );
}
