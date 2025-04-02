import { FormEditModal } from "@/components/ui/form-edit-model";
import {
  customerSchema,
  type CustomerSchema,
} from "@/lib/schemas/customer-schema";
import { type EditTableSheetProps } from "@/types/data-table";
import { yupResolver } from "@hookform/resolvers/yup";
import { useForm } from "react-hook-form";
import { CustomerForm } from "./customer-form";

export function EditCustomerModal({
  open,
  onOpenChange,
  currentRecord,
}: EditTableSheetProps<CustomerSchema>) {
  const form = useForm<CustomerSchema>({
    resolver: yupResolver(customerSchema),
    defaultValues: currentRecord,
  });

  return (
    <FormEditModal
      currentRecord={currentRecord}
      open={open}
      onOpenChange={onOpenChange}
      className="max-w-xl"
      title="Customer"
      formComponent={<CustomerForm />}
      form={form}
      schema={customerSchema}
      url="/customers/"
      queryKey="customer-list"
      fieldKey="name"
    />
  );
}
