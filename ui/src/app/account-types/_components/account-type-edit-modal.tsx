import { FormEditModal } from "@/components/ui/form-edit-modal";
import {
    accountTypeSchema,
    AccountTypeSchema,
} from "@/lib/schemas/account-type-schema";
import { type EditTableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { AccountTypeForm } from "./account-type-form";

export function EditAccountTypeModal({
  currentRecord,
}: EditTableSheetProps<AccountTypeSchema>) {
  const form = useForm({
    resolver: zodResolver(accountTypeSchema),
    defaultValues: currentRecord,
  });

  return (
    <FormEditModal
      currentRecord={currentRecord}
      url="/account-types/"
      title="Account Type"
      queryKey="account-type-list"
      formComponent={<AccountTypeForm />}
      fieldKey="code"
      form={form}
    />
  );
}
