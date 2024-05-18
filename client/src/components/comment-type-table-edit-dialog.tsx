import { CommentTypeForm } from "@/components/comment-type-table-dialog";
import { Button } from "@/components/ui/button";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { formatToUserTimezone } from "@/lib/date";
import { commentTypeSchema } from "@/lib/validations/DispatchSchema";
import { useTableStore } from "@/stores/TableStore";
import type {
  CommentType,
  CommentTypeFormValues as FormValues,
} from "@/types/dispatch";
import { type TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import { useForm } from "react-hook-form";
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

function CommentTypeEditForm({ commentType }: { commentType: CommentType }) {
  const { control, reset, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(commentTypeSchema),
    defaultValues: commentType,
  });

  const mutation = useCustomMutation<FormValues>(control, {
    method: "PUT",
    path: `/comment-types/${commentType.id}/`,
    successMessage: "Comment Type updated successfully.",
    queryKeysToInvalidate: "commentTypes",
    closeModal: true,
    reset,
    errorMessage: "Failed to create update charge type.",
  });

  const onSubmit = (values: FormValues) => {
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
          <Button type="submit" isLoading={mutation.isPending}>
            Save Changes
          </Button>
        </CredenzaFooter>
      </form>
    </CredenzaBody>
  );
}

export function CommentTypeEditSheet({ onOpenChange, open }: TableSheetProps) {
  const [commentType] = useTableStore.use("currentRecord") as CommentType[];

  if (!commentType) return null;

  return (
    <Credenza open={open} onOpenChange={onOpenChange}>
      <CredenzaContent>
        <CredenzaHeader>
          <CredenzaTitle className="flex">
            <span>{commentType.name}</span>
            <Badge className="ml-5" variant="purple">
              {commentType.id}
            </Badge>
          </CredenzaTitle>
        </CredenzaHeader>
        <CredenzaDescription>
          Last updated on {formatToUserTimezone(commentType.updatedAt)}
        </CredenzaDescription>
        <CommentTypeEditForm commentType={commentType} />
      </CredenzaContent>
    </Credenza>
  );
}
