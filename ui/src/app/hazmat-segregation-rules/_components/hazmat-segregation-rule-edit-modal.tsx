/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { FormEditModal } from "@/components/ui/form-edit-modal";
import {
  hazmatSegregationRuleSchema,
  type HazmatSegregationRuleSchema,
} from "@/lib/schemas/hazmat-segregation-rule-schema";
import { type EditTableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { HazmatSegregationRuleForm } from "./hazmat-segregation-rule-form";

export function EditHazmatSegregationRuleModal({
  currentRecord,
}: EditTableSheetProps<HazmatSegregationRuleSchema>) {
  const form = useForm({
    resolver: zodResolver(hazmatSegregationRuleSchema),
    defaultValues: currentRecord,
  });

  return (
    <FormEditModal
      currentRecord={currentRecord}
      url="/hazmat-segregation-rules/"
      title="Hazmat Segregation Rule"
      queryKey="hazmat-segregation-rule-list"
      formComponent={<HazmatSegregationRuleForm />}
      fieldKey="name"
      className="max-w-[500px]"
      form={form}
    />
  );
}
