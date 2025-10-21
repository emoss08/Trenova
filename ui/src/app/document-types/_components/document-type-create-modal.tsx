import { FormCreateModal } from "@/components/ui/form-create-modal";
import {
  DocumentTypeSchema,
  documentTypeSchema,
} from "@/lib/schemas/document-type-schema";
import { DocumentCategory, DocumentClassification } from "@/types/billing";
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
      documentCategory: DocumentCategory.Shipment,
      documentClassification: DocumentClassification.Public,
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
