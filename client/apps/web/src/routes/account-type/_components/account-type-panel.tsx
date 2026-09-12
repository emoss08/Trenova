import { useT } from "@trenova/shared/i18n/use-t";
import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import type { AccountTypeRow } from "@/lib/graphql/account-type-table";
import { accountTypeSchema } from "@/types/account-type";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { AccountTypeForm } from "./account-type-form";

export function AccountTypePanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<AccountTypeRow>) {
  const t = useT();

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
        title={t("Account Type")}
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
      title={t("Account Type")}
      formComponent={<AccountTypeForm />}
    />
  );
}
