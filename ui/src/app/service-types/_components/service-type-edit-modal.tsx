import { FormEditModal } from "@/components/ui/form-edit-model";
import {
    serviceTypeSchema,
    ServiceTypeSchema,
} from "@/lib/schemas/service-type-schema";
import { type EditTableSheetProps } from "@/types/data-table";
import { yupResolver } from "@hookform/resolvers/yup";
import { useForm } from "react-hook-form";
import { ServiceTypeForm } from "./service-type-form";

export function EditServiceTypeModal({
  open,
  onOpenChange,
  currentRecord,
}: EditTableSheetProps<ServiceTypeSchema>) {
  const form = useForm<ServiceTypeSchema>({
    resolver: yupResolver(serviceTypeSchema),
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
      schema={serviceTypeSchema}
    />
  );
}
