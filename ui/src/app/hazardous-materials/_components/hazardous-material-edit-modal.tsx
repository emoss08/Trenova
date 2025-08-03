/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { FormEditModal } from "@/components/ui/form-edit-modal";
import {
  hazardousMaterialSchema,
  HazardousMaterialSchema,
} from "@/lib/schemas/hazardous-material-schema";
import { type EditTableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { HazardousMaterialForm } from "./hazardous-material-form";

export function EditHazardousMaterialModal({
  currentRecord,
}: EditTableSheetProps<HazardousMaterialSchema>) {
  const form = useForm({
    resolver: zodResolver(hazardousMaterialSchema),
    defaultValues: currentRecord,
  });

  return (
    <FormEditModal
      currentRecord={currentRecord}
      url="/hazardous-materials/"
      title="Hazardous Material"
      queryKey="hazardous-material-list"
      formComponent={<HazardousMaterialForm />}
      fieldKey="code"
      className="max-w-[550px]"
      form={form}
    />
  );
}
