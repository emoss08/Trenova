import { FormEditModal } from "@/components/ui/form-edit-modal";
import {
  emailProfileSchema,
  type EmailProfileSchema,
} from "@/lib/schemas/email-profile-schema";
import { type EditTableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { EmailProfileForm } from "./email-profile-form";

export function EditEmailProfileModal({
  currentRecord,
}: EditTableSheetProps<EmailProfileSchema>) {
  const form = useForm({
    resolver: zodResolver(emailProfileSchema),
    defaultValues: currentRecord,
  });

  return (
    <FormEditModal
      currentRecord={currentRecord}
      url="/email-profiles/"
      title="Email Profile"
      queryKey="email-profile-list"
      formComponent={<EmailProfileForm />}
      fieldKey="name"
      form={form}
    />
  );
}
