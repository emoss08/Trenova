import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import type { DataTablePanelProps } from "@/types/data-table";
import { userSchema, type User } from "@/types/user";
import { zodResolver } from "@hookform/resolvers/zod";
import { parseAsString, useQueryState } from "nuqs";
import { useForm } from "react-hook-form";
import { UserForm } from "./user-form";

export function UserPanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<User>) {
  const [panelEntityId] = useQueryState("panelEntityId", parseAsString);
  const editUserId = row?.id ?? panelEntityId ?? undefined;

  const form = useForm({
    resolver: zodResolver(userSchema),
    defaultValues: {
      status: "Active",
      name: "",
      username: "",
      emailAddress: "",
      isLocked: false,
      mustChangePassword: true,
      profilePicUrl: "",
      timezone: "",
    },
  });

  if (mode === "edit") {
    return (
      <FormEditPanel
        open={open}
        onOpenChange={onOpenChange}
        row={row}
        form={form}
        url="/users/"
        queryKey="user-list"
        title="User"
        fieldKey="username"
        formComponent={<UserForm isEdit editUserId={editUserId} />}
      />
    );
  }

  return (
    <FormCreatePanel
      open={open}
      onOpenChange={onOpenChange}
      form={form}
      url="/users/"
      queryKey="user-list"
      title="User"
      formComponent={<UserForm />}
    />
  );
}
