import { Button } from "@/components/ui/button";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { formatToUserTimezone } from "@/lib/date";
import { emailProfileSchema } from "@/lib/validations/OrganizationSchema";
import { useTableStore } from "@/stores/TableStore";
import type {
  EmailProfile,
  EmailProfileFormValues as FormValues,
} from "@/types/organization";
import { yupResolver } from "@hookform/resolvers/yup";
import { useForm } from "react-hook-form";
import { EmailProfileForm } from "./email-profile-table-dialog";
import { Badge } from "./ui/badge";
import {
  Credenza,
  CredenzaBody,
  CredenzaClose,
  CredenzaContent,
  CredenzaDescription,
  CredenzaFooter,
  CredenzaHeader,
  CredenzaTitle,
} from "./ui/credenza";

function EmailProfileEditForm({
  emailProfile,
}: {
  emailProfile: EmailProfile;
}) {
  const { control, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(emailProfileSchema),
    defaultValues: emailProfile,
  });

  const mutation = useCustomMutation<FormValues>(control, {
    method: "PUT",
    path: `/email-profiles/${emailProfile.id}/`,
    successMessage: "Email Profile updated successfully.",
    queryKeysToInvalidate: ["email-profile-table-data"],
    closeModal: true,
    errorMessage: "Failed to update email profile.",
  });

  const onSubmit = (values: FormValues) => {
    mutation.mutate(values);
  };

  return (
    <CredenzaBody>
      <form onSubmit={handleSubmit(onSubmit)}>
        <EmailProfileForm control={control} />
        <CredenzaFooter>
          <CredenzaClose asChild>
            <Button variant="outline" type="button">
              Cancel
            </Button>
          </CredenzaClose>
          <Button type="submit" isLoading={mutation.isPending}>
            Save Changes
          </Button>
        </CredenzaFooter>
      </form>
    </CredenzaBody>
  );
}

export function EmailProfileTableEditDialog({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const [emailProfile] = useTableStore.use("currentRecord") as EmailProfile[];

  if (!emailProfile) return null;

  return (
    <Credenza open={open} onOpenChange={onOpenChange}>
      <CredenzaContent>
        <CredenzaHeader>
          <CredenzaTitle className="flex">
            <span>{emailProfile.name}</span>
            <Badge className="ml-5" variant="purple">
              {emailProfile.id}
            </Badge>
          </CredenzaTitle>
        </CredenzaHeader>
        <CredenzaDescription>
          Last updated on {formatToUserTimezone(emailProfile.updatedAt)}
        </CredenzaDescription>
        <EmailProfileEditForm emailProfile={emailProfile} />
      </CredenzaContent>
    </Credenza>
  );
}
