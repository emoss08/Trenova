/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { FormEditModal } from "@/components/ui/form-edit-modal";
import {
  serviceTypeSchema,
  ServiceTypeSchema,
} from "@/lib/schemas/service-type-schema";
import { type EditTableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { ServiceTypeForm } from "./service-type-form";

export function EditServiceTypeModal({
  currentRecord,
}: EditTableSheetProps<ServiceTypeSchema>) {
  const form = useForm({
    resolver: zodResolver(serviceTypeSchema),
    defaultValues: currentRecord,
  });

  return (
    <FormEditModal
      currentRecord={currentRecord}
      url="/service-types/"
      title="Service Type"
      queryKey="service-type-list"
      formComponent={<ServiceTypeForm />}
      fieldKey="code"
      form={form}
    />
  );
}
