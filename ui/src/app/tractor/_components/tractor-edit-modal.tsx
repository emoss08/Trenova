/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { FormEditModal } from "@/components/ui/form-edit-modal";
import {
  tractorSchema,
  type TractorSchema,
} from "@/lib/schemas/tractor-schema";
import { type EditTableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { TractorForm } from "./tractor-form";

export function EditTractorModal({
  currentRecord,
}: EditTableSheetProps<TractorSchema>) {
  const form = useForm({
    resolver: zodResolver(tractorSchema),
    defaultValues: currentRecord,
  });

  return (
    <FormEditModal
      currentRecord={currentRecord}
      url="/tractors/"
      title="Tractor"
      queryKey="tractor-list"
      formComponent={<TractorForm />}
      fieldKey="code"
      form={form}
      className="max-w-[500px]"
    />
  );
}
