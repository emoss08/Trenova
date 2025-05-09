import { FormCreateModal } from "@/components/ui/form-create-modal";
import { tractorSchema } from "@/lib/schemas/tractor-schema";
import { type TableSheetProps } from "@/types/data-table";
import { EquipmentStatus } from "@/types/tractor";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { TractorForm } from "./tractor-form";

export function CreateTractorModal({ open, onOpenChange }: TableSheetProps) {
  const form = useForm({
    resolver: zodResolver(tractorSchema),
    defaultValues: {
      status: EquipmentStatus.Available,
      vin: "",
      equipmentManufacturerId: "",
      secondaryWorkerId: "",
      equipmentTypeId: "",
      primaryWorkerId: "",
      fleetCodeId: "",
      stateId: "",
      code: "",
      model: "",
      make: "",
      year: undefined,
      registrationNumber: "",
      licensePlateNumber: "",
      registrationExpiry: undefined,
      createdAt: undefined,
      updatedAt: undefined,
      version: undefined,
      id: undefined,
    },
  });

  return (
    <FormCreateModal
      open={open}
      onOpenChange={onOpenChange}
      title="Tractor"
      formComponent={<TractorForm />}
      form={form}
      url="/tractors/"
      queryKey="tractor-list"
      className="max-w-[500px]"
    />
  );
}
