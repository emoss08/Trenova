/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { FormCreateModal } from "@/components/ui/form-create-modal";
import { serviceTypeSchema } from "@/lib/schemas/service-type-schema";
import { Status } from "@/types/common";
import { type TableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { ServiceTypeForm } from "./service-type-form";

export function CreateServiceTypeModal({
  open,
  onOpenChange,
}: TableSheetProps) {
  const form = useForm({
    resolver: zodResolver(serviceTypeSchema),
    defaultValues: {
      code: "",
      status: Status.Active,
      description: "",
      color: "",
    },
  });

  return (
    <FormCreateModal
      open={open}
      onOpenChange={onOpenChange}
      title="Service Type"
      formComponent={<ServiceTypeForm />}
      form={form}
      url="/service-types/"
      queryKey="service-type-list"
      className="max-w-[400px]"
    />
  );
}
