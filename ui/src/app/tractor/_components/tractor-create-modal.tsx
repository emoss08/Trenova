import { FormCreateModal } from "@/components/ui/form-create-modal";
import { TractorSchema, tractorSchema } from "@/lib/schemas/tractor-schema";
import { type TableSheetProps } from "@/types/data-table";
import { EquipmentStatus } from "@/types/tractor";
import { yupResolver } from "@hookform/resolvers/yup";
import { useForm } from "react-hook-form";
import { TractorForm } from "./tractor-form";

export function CreateTractorModal({ open, onOpenChange }: TableSheetProps) {
  const form = useForm<TractorSchema>({
    resolver: yupResolver(tractorSchema),
    defaultValues: {
      status: EquipmentStatus.Available,
      vin: "",
      equipmentManufacturerId: undefined,
      secondaryWorkerId: undefined,
    },
  });

  return (
    <FormCreateModal
      open={open}
      onOpenChange={onOpenChange}
      title="Tractor"
      formComponent={<TractorForm />}
      form={form}
      schema={tractorSchema}
      url="/tractors/"
      queryKey={["tractor"]}
    />
  );
}
