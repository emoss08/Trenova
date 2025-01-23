import { FormCreateModal } from "@/components/ui/form-create-modal";
import {
  equipmentTypeSchema,
  EquipmentTypeSchema,
} from "@/lib/schemas/equipment-type-schema";
import { Status } from "@/types/common";
import { type TableSheetProps } from "@/types/data-table";
import { EquipmentClass } from "@/types/equipment-type";
import { yupResolver } from "@hookform/resolvers/yup";
import { useForm } from "react-hook-form";
import { EquipTypeForm } from "./equip-type-form";

export function CreateEquipTypeModal({ open, onOpenChange }: TableSheetProps) {
  const form = useForm<EquipmentTypeSchema>({
    resolver: yupResolver(equipmentTypeSchema),
    defaultValues: {
      code: "",
      status: Status.Active,
      description: "",
      class: EquipmentClass.Tractor,
      color: "",
    },
  });

  return (
    <FormCreateModal
      open={open}
      onOpenChange={onOpenChange}
      title="Equipment Type"
      formComponent={<EquipTypeForm />}
      form={form}
      schema={equipmentTypeSchema}
      url="/equipment-types/"
      queryKey={["equipment-type"]}
    />
  );
}
