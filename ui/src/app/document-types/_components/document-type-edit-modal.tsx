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
  open,
  onOpenChange,
  currentRecord,
}: EditTableSheetProps<DocumentTypeSchema>) {
  const form = useForm<DocumentTypeSchema>({
    resolver: zodResolver(documentTypeSchema),
    defaultValues: currentRecord,
  });

  return (
    <FormEditModal
      currentRecord={currentRecord}
      open={open}
      onOpenChange={onOpenChange}
      url="/document-types/"
      title="Document Type"
      queryKey="document-type-list"
      formComponent={<DocumentTypeForm />}
      fieldKey="name"
      form={form}
    />
  );
}
