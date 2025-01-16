import { FormCreateModal } from "@/components/ui/form-create-modal";
import {
    serviceTypeSchema,
    ServiceTypeSchema,
} from "@/lib/schemas/service-type-schema";
import { Status } from "@/types/common";
import { type TableSheetProps } from "@/types/data-table";
import { yupResolver } from "@hookform/resolvers/yup";
import { useForm } from "react-hook-form";
import { ServiceTypeForm } from "./service-type-form";

export function CreateServiceTypeModal({
  open,
  onOpenChange,
}: TableSheetProps) {
  const form = useForm<ServiceTypeSchema>({
    resolver: yupResolver(serviceTypeSchema),
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
      schema={serviceTypeSchema}
      url="/service-types/"
      queryKey="service-type-list"
      className="max-w-[400px]"
    />
  );
}
