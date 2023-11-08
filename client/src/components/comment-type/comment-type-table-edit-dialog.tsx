/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */

import React from "react";
import { yupResolver } from "@hookform/resolvers/yup";
import { useForm } from "react-hook-form";
import { useCustomMutation } from "@/hooks/useCustomMutation";

import { TableSheetProps } from "@/types/tables";
import { useTableStore } from "@/stores/TableStore";
import { formatDate } from "@/lib/date";
import { commentTypeSchema } from "@/lib/validations/DispatchSchema";
import {
  CommentType,
  CommentTypeFormValues as FormValues,
} from "@/types/dispatch";
import { toast } from "@/components/ui/use-toast";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { CommentTypeForm } from "@/components/comment-type/comment-type-table-dialog";
import { Button } from "@/components/ui/button";

function CommentTypeEditForm({ commentType }: { commentType: CommentType }) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);
  const { control, reset, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(commentTypeSchema),
    defaultValues: {
      status: commentType.status,
      name: commentType.name,
      description: commentType.description,
    },
  });

  const mutation = useCustomMutation<FormValues>(
    control,
    toast,
    {
      method: "PUT",
      path: `/comment_types/${commentType.id}/`,
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
    <form onSubmit={handleSubmit(onSubmit)}>
      <CommentTypeForm control={control} />
      <DialogFooter className="mt-6">
        <Button
          type="submit"
          isLoading={isSubmitting}
          loadingText="Saving Changes..."
        >
          Save
        </Button>
      </DialogFooter>
    </form>
  );
}

export function CommentTypeEditSheet({ onOpenChange, open }: TableSheetProps) {
  const [commentType] = useTableStore.use("currentRecord");

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{commentType && commentType.name}</DialogTitle>
        </DialogHeader>
        <DialogDescription>
          Last updated on {commentType && formatDate(commentType.modified)}
        </DialogDescription>
        {commentType && <CommentTypeEditForm commentType={commentType} />}
      </DialogContent>
    </Dialog>
  );
}
