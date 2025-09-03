/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { FormCreateModal } from "@/components/ui/form-create-modal";
import { fleetCodeSchema } from "@/lib/schemas/fleet-code-schema";
import { Status } from "@/types/common";
import { type TableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { FleetCodeForm } from "./fleet-code-form";

export function CreateFleetCodeModal({ open, onOpenChange }: TableSheetProps) {
  const form = useForm({
    resolver: zodResolver(fleetCodeSchema),
    defaultValues: {
      name: "",
      status: Status.Active,
      description: "",
      managerId: "",
      revenueGoal: undefined,
      deadheadGoal: undefined,
      color: "",
    },
  });

  return (
    <FormCreateModal
      open={open}
      onOpenChange={onOpenChange}
      title="Fleet Code"
      formComponent={<FleetCodeForm />}
      form={form}
      url="/fleet-codes/"
      queryKey="fleet-code-list"
    />
  );
}
