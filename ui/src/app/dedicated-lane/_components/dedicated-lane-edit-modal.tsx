/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { FormEditModal } from "@/components/ui/form-edit-modal";
import {
  dedicatedLaneSchema,
  type DedicatedLaneSchema,
} from "@/lib/schemas/dedicated-lane-schema";
import { type EditTableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { DedicatedLaneForm } from "./dedicated-lane-form";

export function EditDedicatedLaneModal({
  currentRecord,
}: EditTableSheetProps<DedicatedLaneSchema>) {
  const form = useForm({
    resolver: zodResolver(dedicatedLaneSchema),
    defaultValues: currentRecord,
  });

  const {
    formState: { errors },
  } = form;

  console.log("errors", errors);

  return (
    <FormEditModal
      currentRecord={currentRecord}
      url="/dedicated-lanes/"
      title="Dedicated Lane"
      queryKey="dedicated-lane-list"
      formComponent={<DedicatedLaneForm />}
      form={form}
      className="max-w-[500px]"
    />
  );
}
