import { FormCreateModal } from "@/components/ui/form-create-modal";
import { customerSchema, CustomerSchema } from "@/lib/schemas/customer-schema";
import { Status } from "@/types/common";
import { type TableSheetProps } from "@/types/data-table";
import { yupResolver } from "@hookform/resolvers/yup";
import { useForm } from "react-hook-form";
import { CustomerForm } from "./customer-form";

export function CreateCustomerModal({ open, onOpenChange }: TableSheetProps) {
  const form = useForm<CustomerSchema>({
    resolver: yupResolver(customerSchema),
    defaultValues: {
      status: Status.Active,
      name: "",
      code: "",
      description: "",
      addressLine1: "",
      addressLine2: "",
      city: "",
      postalCode: "",
      stateId: "",
    },
  });

  return (
    <FormCreateModal
      open={open}
      onOpenChange={onOpenChange}
      title="Customer"
      formComponent={<CustomerForm />}
      className="max-w-xl"
      form={form}
      schema={customerSchema}
      url="/customers/"
      queryKey="customer-list"
    />
  );
}
