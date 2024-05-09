import { Button } from "@/components/ui/button";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { formatToUserTimezone } from "@/lib/date";
import { cn } from "@/lib/utils";
import { customerSchema } from "@/lib/validations/CustomerSchema";
import { useTableStore } from "@/stores/TableStore";
import type {
  Customer,
  CustomerFormValues as FormValues,
} from "@/types/customer";
import { type TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import { FormProvider, useForm } from "react-hook-form";
import { CustomerForm } from "./customer-table-dialog";
import { Badge } from "./ui/badge";

export function CustomerEditForm({
  customer,
  open,
  onOpenChange,
}: {
  customer: Customer;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  if (!customer) return null;

  const customerForm = useForm<FormValues>({
    resolver: yupResolver(customerSchema),
    defaultValues: customer,
  });

  const { control, handleSubmit } = customerForm;

  const mutation = useCustomMutation<FormValues>(control, {
    method: "PUT",
    path: `/customers/${customer.id}/`,
    successMessage: "Customer updated successfully.",
    queryKeysToInvalidate: ["customers-table-data"],
    closeModal: true,
    errorMessage: "Failed to update existing customer.",
  });

  function onSubmit(values: FormValues) {
    mutation.mutate(values);
  }

  return (
    <FormProvider {...customerForm}>
      <form
        onSubmit={handleSubmit(onSubmit)}
        className="flex h-full flex-col overflow-y-auto"
      >
        <CustomerForm open={open} />
        <SheetFooter className="mb-12">
          <Button
            type="reset"
            variant="secondary"
            onClick={() => onOpenChange(false)}
            className="w-full"
          >
            Cancel
          </Button>
          <Button
            type="submit"
            isLoading={mutation.isPending}
            className="w-full"
          >
            Save
          </Button>
        </SheetFooter>
      </form>
    </FormProvider>
  );
}

export function CustomerTableEditSheet({
  onOpenChange,
  open,
}: TableSheetProps) {
  const [customer] = useTableStore.use("currentRecord") as Customer[];

  if (!customer) return null;

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className={cn("w-full xl:w-1/2")}>
        <SheetHeader>
          <SheetTitle className="flex">
            <span>{customer.name}</span>
            <Badge className="ml-5" variant="purple">
              {customer.id}
            </Badge>
          </SheetTitle>
          <SheetDescription>
            Last updated on {formatToUserTimezone(customer.updatedAt)}
          </SheetDescription>
        </SheetHeader>
        <CustomerEditForm
          customer={customer}
          open={open}
          onOpenChange={onOpenChange}
        />
      </SheetContent>
    </Sheet>
  );
}
