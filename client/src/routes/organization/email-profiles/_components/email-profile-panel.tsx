import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import type { DataTablePanelProps } from "@/types/data-table";
import { emailProfileSchema, type EmailProfile } from "@/types/email";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import {
  emailProfileDefaults,
  emailProfileQueryKey,
  type EmailProfileFormValues,
} from "./email-profile-constants";
import { EmailProfileForm } from "./email-profile-form";

export function EmailProfilePanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<EmailProfile>) {
  const form = useForm<EmailProfileFormValues>({
    resolver: zodResolver(emailProfileSchema),
    defaultValues: emailProfileDefaults,
  });

  if (mode === "edit") {
    return (
      <FormEditPanel<EmailProfileFormValues, EmailProfile>
        open={open}
        onOpenChange={onOpenChange}
        row={row}
        form={form}
        url="/email-profiles/"
        queryKey={emailProfileQueryKey}
        title="Email Profile"
        fieldKey="name"
        formComponent={<EmailProfileForm />}
      />
    );
  }

  return (
    <FormCreatePanel<EmailProfileFormValues, EmailProfile>
      open={open}
      onOpenChange={onOpenChange}
      form={form}
      url="/email-profiles/"
      queryKey={emailProfileQueryKey}
      title="Email Profile"
      formComponent={<EmailProfileForm />}
    />
  );
}
