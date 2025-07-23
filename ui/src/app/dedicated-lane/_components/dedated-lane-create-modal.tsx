/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { FormCreateModal } from "@/components/ui/form-create-modal";
import { dedicatedLaneSchema } from "@/lib/schemas/dedicated-lane-schema";
import { Status } from "@/types/common";
import { type TableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { DedicatedLaneForm } from "./dedicated-lane-form";

export function CreateDedicatedLaneModal({
  open,
  onOpenChange,
}: TableSheetProps) {
  const form = useForm({
    resolver: zodResolver(dedicatedLaneSchema),
    defaultValues: {
      createdAt: undefined,
      updatedAt: undefined,
      id: undefined,
      version: undefined,
      status: Status.Active,
      autoAssign: true,
      originLocationId: "",
      destinationLocationId: "",
      customerId: "",
      primaryWorkerId: "",
      secondaryWorkerId: "",
      trailerTypeId: "",
      tractorTypeId: "",
      shipmentTypeId: "",
      serviceTypeId: "",
      customer: undefined,
      destinationLocation: undefined,
      originLocation: undefined,
      primaryWorker: undefined,
      secondaryWorker: undefined,
      trailerType: undefined,
      tractorType: undefined,
      shipmentType: undefined,
      serviceType: undefined,
    },
  });

  return (
    <FormCreateModal
      open={open}
      onOpenChange={onOpenChange}
      title="Dedicated Lane"
      formComponent={<DedicatedLaneForm />}
      form={form}
      url="/dedicated-lanes/"
      queryKey="dedicated-lane-list"
      className="max-w-[500px]"
    />
  );
}
