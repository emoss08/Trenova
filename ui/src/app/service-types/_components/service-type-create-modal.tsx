import { FormCreateModal } from "@/components/ui/form-create-modal";
import { serviceTypeSchema } from "@/lib/schemas/service-type-schema";
import { Status } from "@/types/common";
import { type TableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { ServiceTypeForm } from "./service-type-form";

export function CreateServiceTypeModal({
  open,
  onOpenChange,
}: TableSheetProps) {
  const form = useForm({
    resolver: zodResolver(serviceTypeSchema),
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
      url="/service-types/"
      queryKey="service-type-list"
      className="max-w-[400px]"
    />
  );
}
