import { FormCreateModal } from "@/components/ui/form-create-modal";
import { trailerSchema } from "@/lib/schemas/trailer-schema";
import { type TableSheetProps } from "@/types/data-table";
import { EquipmentStatus } from "@/types/tractor";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { TrailerForm } from "./trailer-form";

export function CreateTrailerModal({ open, onOpenChange }: TableSheetProps) {
  const form = useForm({
    resolver: zodResolver(trailerSchema),
    defaultValues: {
      status: EquipmentStatus.Available,
      code: "",
      model: "",
      make: "",
      year: undefined,
      licensePlateNumber: "",
      vin: "",
      registrationNumber: "",
      maxLoadWeight: undefined,
      lastInspectionDate: undefined,
      registrationExpiry: undefined,
      equipmentTypeId: undefined,
      equipmentManufacturerId: undefined,
      fleetCodeId: undefined,
      registrationStateId: undefined,
    },
  });

  return (
    <FormCreateModal
      open={open}
      onOpenChange={onOpenChange}
      title="Trailer"
      formComponent={<TrailerForm />}
      form={form}
      url="/trailers/"
      queryKey="trailer-list"
      className="max-w-[500px]"
    />
  );
}
