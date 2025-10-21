/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { FormCreateModal } from "@/components/ui/form-create-modal";
import { EquipmentStatus } from "@/lib/schemas/tractor-schema";
import { trailerSchema } from "@/lib/schemas/trailer-schema";
import { type TableSheetProps } from "@/types/data-table";
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
      equipmentTypeId: "",
      equipmentManufacturerId: "",
      fleetCodeId: "",
      registrationStateId: "",
      createdAt: undefined,
      updatedAt: undefined,
      id: undefined,
      version: undefined,
      equipmentManufacturer: undefined,
      equipmentType: undefined,
      fleetCode: undefined,
      registrationState: undefined,
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
