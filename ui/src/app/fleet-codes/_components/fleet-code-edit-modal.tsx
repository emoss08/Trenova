/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { FormEditModal } from "@/components/ui/form-edit-modal";
import {
  fleetCodeSchema,
  type FleetCodeSchema,
} from "@/lib/schemas/fleet-code-schema";
import { type EditTableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { FleetCodeForm } from "./fleet-code-form";

export function EditFleetCodeModal({
  currentRecord,
}: EditTableSheetProps<FleetCodeSchema>) {
  const form = useForm({
    resolver: zodResolver(fleetCodeSchema),
    defaultValues: currentRecord,
  });

  return (
    <FormEditModal
      currentRecord={currentRecord}
      url="/fleet-codes/"
      title="Fleet Code"
      queryKey="fleet-code-list"
      formComponent={<FleetCodeForm />}
      fieldKey="name"
      form={form}
    />
  );
}
