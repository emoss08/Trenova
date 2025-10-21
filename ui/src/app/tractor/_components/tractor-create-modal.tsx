/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { FormCreateModal } from "@/components/ui/form-create-modal";
import { EquipmentStatus, tractorSchema } from "@/lib/schemas/tractor-schema";
import { type TableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { TractorForm } from "./tractor-form";

export function CreateTractorModal({ open, onOpenChange }: TableSheetProps) {
  const form = useForm({
    resolver: zodResolver(tractorSchema),
    defaultValues: {
      id: undefined,
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
      year: 0,
      registrationNumber: "",
      licensePlateNumber: "",
      registrationExpiry: undefined,
      createdAt: undefined,
      updatedAt: undefined,
      version: undefined,
      fleetCode: null,
      equipmentType: null,
      equipmentManufacturer: null,
      primaryWorker: null,
      secondaryWorker: null,
      organizationId: undefined,
      businessUnitId: undefined,
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
