import { CommentTypeForm } from "@/components/comment-type-table-dialog";
import { Button } from "@/components/ui/button";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { formatDate } from "@/lib/date";
import { commentTypeSchema } from "@/lib/validations/DispatchSchema";
import { useTableStore } from "@/stores/TableStore";
import type {
  CommentType,
  CommentTypeFormValues as FormValues,
} from "@/types/dispatch";
import { type TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import React from "react";
import { useForm } from "react-hook-form";
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

function CommentTypeEditForm({ commentType }: { commentType: CommentType }) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);
  const { control, reset, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(commentTypeSchema),
    defaultValues: commentType,
  });

  const mutation = useCustomMutation<FormValues>(
    control,
    {
      method: "PUT",
      path: `/comment-types/${commentType.id}/`,
      successMessage: "Comment Type updated successfully.",
      queryKeysToInvalidate: ["comment-types-table-data"],
      closeModal: true,
      errorMessage: "Failed to create update charge type.",
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
        <CommentTypeForm control={control} />
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

export function CommentTypeEditSheet({ onOpenChange, open }: TableSheetProps) {
  const [commentType] = useTableStore.use("currentRecord") as CommentType[];

  return (
    <Credenza open={open} onOpenChange={onOpenChange}>
      <CredenzaContent>
        <CredenzaHeader>
          <CredenzaTitle>{commentType && commentType.name}</CredenzaTitle>
        </CredenzaHeader>
        <CredenzaDescription>
          Last updated on {commentType && formatDate(commentType.updatedAt)}
        </CredenzaDescription>
        {commentType && <CommentTypeEditForm commentType={commentType} />}
      </CredenzaContent>
    </Credenza>
  );
}
