/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { FormEditModal } from "@/components/ui/form-edit-modal";
import {
  accessorialChargeSchema,
  type AccessorialChargeSchema,
} from "@/lib/schemas/accessorial-charge-schema";
import { type EditTableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { AccessorialChargeForm } from "./accessorial-charge-form";

export function EditAccessorialChargeModal({
  currentRecord,
}: EditTableSheetProps<AccessorialChargeSchema>) {
  const form = useForm({
    resolver: zodResolver(accessorialChargeSchema),
    defaultValues: currentRecord,
  });

  return (
    <FormEditModal
      currentRecord={currentRecord}
      url="/accessorial-charges/"
      title="Accessorial Charge"
      queryKey="accessorial-charge-list"
      formComponent={<AccessorialChargeForm />}
      fieldKey="code"
      form={form}
    />
  );
}
