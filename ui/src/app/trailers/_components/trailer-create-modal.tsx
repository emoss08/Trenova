import { FormCreateModal } from "@/components/ui/form-create-modal";
import { TrailerSchema, trailerSchema } from "@/lib/schemas/trailer-schema";
import { type TableSheetProps } from "@/types/data-table";
import { EquipmentStatus } from "@/types/tractor";
import { yupResolver } from "@hookform/resolvers/yup";
import { useForm } from "react-hook-form";
import { TrailerForm } from "./trailer-form";

export function CreateTrailerModal({ open, onOpenChange }: TableSheetProps) {
  const form = useForm<TrailerSchema>({
    resolver: yupResolver(trailerSchema),
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
      schema={trailerSchema}
      url="/trailers/"
      queryKey="trailer-list"
      className="max-w-[500px]"
    />
  );
}
