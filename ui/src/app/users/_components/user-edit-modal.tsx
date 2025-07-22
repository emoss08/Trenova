import { FormEditModal } from "@/components/ui/form-edit-modal";
import { userSchema, type UserSchema } from "@/lib/schemas/user-schema";
import { type EditTableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { UserForm } from "./user-form";

export function EditUserModal({
  currentRecord,
}: EditTableSheetProps<UserSchema>) {
  const form = useForm({
    resolver: zodResolver(userSchema),
    defaultValues: currentRecord,
  });

  return (
    <FormEditModal
      currentRecord={currentRecord}
      url="/users/"
      title="User"
      queryKey="user-list"
      formComponent={<UserForm />}
      fieldKey="username"
      form={form}
      className="max-w-[500px]"
    />
  );
}
