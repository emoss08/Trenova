import { FormCreateModal } from "@/components/ui/form-create-modal";
import { glAccountSchema } from "@/lib/schemas/gl-account-schema";
import { Status } from "@/types/common";
import { type TableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { GLAccountForm } from "./gl-account-form";

export function CreateGLAccountModal({ open, onOpenChange }: TableSheetProps) {
  const form = useForm({
    resolver: zodResolver(glAccountSchema),
    defaultValues: {
      status: Status.Active,
      accountCode: "",
      name: "",
      accountTypeId: undefined,
      parentId: undefined,
      description: "",
      isSystem: false,
      allowManualJE: true,
      requireProject: false,
      currentBalance: 0,
      debitBalance: 0,
      creditBalance: 0,
    },
  });

  return (
    <FormCreateModal
      open={open}
      onOpenChange={onOpenChange}
      title="GL Account"
      formComponent={<GLAccountForm />}
      form={form}
      url="/gl-accounts/"
      queryKey="gl-account-list"
      className="max-w-[550px]"
    />
  );
}
