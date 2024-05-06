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
import React from "react";
import { useForm } from "react-hook-form";
import { EmailProfileForm } from "./email-profile-table-dialog";
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
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);

  const { control, reset, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(emailProfileSchema),
    defaultValues: emailProfile,
  });

  const mutation = useCustomMutation<FormValues>(
    control,
    {
      method: "PUT",
      path: `/email-profiles/${emailProfile.id}/`,
      successMessage: "Email Profile updated successfully.",
      queryKeysToInvalidate: ["email-profile-table-data"],
      closeModal: true,
      errorMessage: "Failed to update email profile.",
    },
    () => setIsSubmitting(false),
    reset,
  );

  const onSubmit = (values: FormValues) => {
    setIsSubmitting(true);
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
          <Button type="submit" isLoading={isSubmitting}>
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

  if (!emailProfile) {
    return null;
  }

  return (
    <Credenza open={open} onOpenChange={onOpenChange}>
      <CredenzaContent>
        <CredenzaHeader>
          <CredenzaTitle>{emailProfile.name} </CredenzaTitle>
        </CredenzaHeader>
        <CredenzaDescription>
          Last updated on&nbsp;
          {emailProfile && formatToUserTimezone(emailProfile.updatedAt)}
        </CredenzaDescription>
        {emailProfile && <EmailProfileEditForm emailProfile={emailProfile} />}
      </CredenzaContent>
    </Credenza>
  );
}
