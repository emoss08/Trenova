import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import type { DataTablePanelProps } from "@/types/data-table";
import { accountTypeSchema, type AccountType } from "@/types/account-type";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { AccountTypeForm } from "./account-type-form";

export function AccountTypePanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<AccountType>) {
  const form = useForm({
    resolver: zodResolver(accountTypeSchema),
    defaultValues: {
      status: "Active",
      code: "",
      name: "",
      description: "",
      category: "Asset",
      color: "",
    },
    mode: "onChange",
  });

  if (mode === "edit") {
    return (
      <FormEditPanel
        open={open}
        onOpenChange={onOpenChange}
        row={row}
        form={form}
        url="/account-types/"
        queryKey="account-type-list"
        title="Account Type"
        fieldKey="code"
        formComponent={<AccountTypeForm />}
      />
    );
  }

  return (
    <FormCreatePanel
      open={open}
      onOpenChange={onOpenChange}
      form={form}
      url="/account-types/"
      queryKey="account-type-list"
      title="Account Type"
      formComponent={<AccountTypeForm />}
    />
  );
}
