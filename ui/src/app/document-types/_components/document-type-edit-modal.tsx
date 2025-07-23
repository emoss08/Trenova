/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

import { FormEditModal } from "@/components/ui/form-edit-modal";
import {
  documentTypeSchema,
  type DocumentTypeSchema,
} from "@/lib/schemas/document-type-schema";
import { type EditTableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { DocumentTypeForm } from "./document-type-form";

export function EditDocumentTypeModal({
  currentRecord,
}: EditTableSheetProps<DocumentTypeSchema>) {
  const form = useForm<DocumentTypeSchema>({
    resolver: zodResolver(documentTypeSchema),
    defaultValues: currentRecord,
  });

  return (
    <FormEditModal
      currentRecord={currentRecord}
      url="/document-types/"
      title="Document Type"
      queryKey="document-type-list"
      formComponent={<DocumentTypeForm />}
      fieldKey="name"
      form={form}
    />
  );
}
