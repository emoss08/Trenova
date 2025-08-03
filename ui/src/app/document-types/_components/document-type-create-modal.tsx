/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { FormCreateModal } from "@/components/ui/form-create-modal";
import {
  DocumentTypeSchema,
  documentTypeSchema,
} from "@/lib/schemas/document-type-schema";
import { TableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { DocumentTypeForm } from "./document-type-form";

export function DocumentTypeCreateModal({
  open,
  onOpenChange,
}: TableSheetProps) {
  const form = useForm<DocumentTypeSchema>({
    resolver: zodResolver(documentTypeSchema),
    defaultValues: {
      name: "",
      description: "",
      code: "",
      color: "",
    },
  });

  return (
    <FormCreateModal
      open={open}
      onOpenChange={onOpenChange}
      title="Document Type"
      formComponent={<DocumentTypeForm />}
      form={form}
      url="/document-types/"
      queryKey="document-type-list"
    />
  );
}
