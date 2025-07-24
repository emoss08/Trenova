/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { FormEditModal } from "@/components/ui/form-edit-modal";
import {
  trailerSchema,
  type TrailerSchema,
} from "@/lib/schemas/trailer-schema";
import { type EditTableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { TrailerForm } from "./trailer-form";

export function EditTrailerModal({
  currentRecord,
}: EditTableSheetProps<TrailerSchema>) {
  const form = useForm({
    resolver: zodResolver(trailerSchema),
    defaultValues: currentRecord,
  });

  return (
    <FormEditModal
      currentRecord={currentRecord}
      url="/trailers/"
      title="Trailer"
      queryKey="trailer-list"
      formComponent={<TrailerForm />}
      fieldKey="code"
      form={form}
      className="max-w-[500px]"
    />
  );
}
