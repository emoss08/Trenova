import { useT } from "@trenova/shared/i18n/use-t";
import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import { apiService } from "@/services/api";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { emailProfileSchema, type EmailProfile } from "@trenova/shared/types/email";
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
  const t = useT();

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
        queryKey={emailProfileQueryKey}
        title={t("Email profile")}
        fieldKey="name"
        formComponent={<EmailProfileForm />}
        mutationFn={async (values, current) => {
          await apiService.emailService.updateProfile(current.id ?? "", values as EmailProfile);
          return values;
        }}
      />
    );
  }

  return (
    <FormCreatePanel<EmailProfileFormValues, EmailProfile>
      open={open}
      onOpenChange={onOpenChange}
      form={form}
      queryKey={emailProfileQueryKey}
      title={t("Email profile")}
      formComponent={<EmailProfileForm />}
      mutationFn={async (values) => {
        await apiService.emailService.createProfile(values as EmailProfile);
        return values;
      }}
    />
  );
}
