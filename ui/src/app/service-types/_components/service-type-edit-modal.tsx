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
  open,
  onOpenChange,
  currentRecord,
}: EditTableSheetProps<ServiceTypeSchema>) {
  const form = useForm({
    resolver: zodResolver(serviceTypeSchema),
    defaultValues: currentRecord,
  });

  return (
    <FormEditModal
      currentRecord={currentRecord}
      open={open}
      onOpenChange={onOpenChange}
      url="/service-types/"
      title="Service Type"
      queryKey="service-type-list"
      formComponent={<ServiceTypeForm />}
      fieldKey="code"
      form={form}
    />
  );
}
