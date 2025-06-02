import { FormEditModal } from "@/components/ui/form-edit-modal";
import { roleSchema, type RoleSchema } from "@/lib/schemas/user-schema";
import { type EditTableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { RoleForm } from "./role-form";

export function EditRoleModal({
  currentRecord,
}: EditTableSheetProps<RoleSchema>) {
  const form = useForm({
    resolver: zodResolver(roleSchema),
    defaultValues: currentRecord,
  });

  const {
    formState: { errors },
  } = form;
  console.info("errors", errors);

  return (
    <FormEditModal
      currentRecord={currentRecord}
      url="/roles/"
      title="Role"
      queryKey="role-list"
      formComponent={<RoleForm />}
      form={form}
      className="max-w-[500px]"
    />
  );
}
