import { FormCreateModal } from "@/components/ui/form-create-modal";
import {
  AccountTypeCategorySchema,
  accountTypeSchema,
} from "@/lib/schemas/account-type-schema";
import { Status } from "@/types/common";
import { type TableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { AccountTypeForm } from "./account-type-form";

export function CreateAccountTypeModal({
  open,
  onOpenChange,
}: TableSheetProps) {
  const form = useForm({
    resolver: zodResolver(accountTypeSchema),
    defaultValues: {
      code: "",
      status: Status.Active,
      name: "",
      category: AccountTypeCategorySchema.enum.Asset,
      isSystem: false,
      description: "",
      color: "",
    },
  });

  return (
    <FormCreateModal
      open={open}
      onOpenChange={onOpenChange}
      title="Account Type"
      formComponent={<AccountTypeForm />}
      form={form}
      url="/account-types/"
      queryKey="account-type-list"
      className="max-w-[400px]"
    />
  );
}
