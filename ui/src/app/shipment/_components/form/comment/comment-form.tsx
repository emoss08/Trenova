/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { Button } from "@/components/ui/button";
import { Form, FormControl, FormGroup } from "@/components/ui/form";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { queries } from "@/lib/queries";
import {
  ShipmentCommentSchema,
  shipmentCommentSchema,
} from "@/lib/schemas/shipment-comment-schema";
import { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { UserSchema } from "@/lib/schemas/user-schema";
import { api } from "@/services/api";
import { useCommentEditStore } from "@/stores/comment-edit-store";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Controller, FormProvider, useForm } from "react-hook-form";
import { toast } from "sonner";
import { TiptapEditor } from "./tiptap-editor";
import { type CommentType, COMMENT_TYPES } from "./utils";

interface CommentFormProps {
  searchUsers: (query: string) => Promise<UserSchema[]>;
  shipmentId: ShipmentSchema["id"];
}

export function CommentForm({ searchUsers, shipmentId }: CommentFormProps) {
  const queryClient = useQueryClient();
  const { editingComment, isEditMode, clearEditMode } = useCommentEditStore();
  const [mentionedUserIds, setMentionedUserIds] = useState<string[]>([]);
  const [commentType, setCommentType] = useState<CommentType | null>(null);
  const [commentJson, setCommentJson] = useState<Record<string, any> | null>(
    null,
  );

  const form = useForm({
    resolver: zodResolver(shipmentCommentSchema),
    defaultValues: {
      comment: "",
      commentType: null,
    },
  });

  useEffect(() => {
    if (isEditMode && editingComment) {
      form.setValue("comment", editingComment.comment);

      if (editingComment.metadata?.editorContent) {
        setCommentJson(editingComment.metadata.editorContent);
      }

      if (editingComment.commentType) {
        setCommentType(editingComment.commentType);
      }

      if (editingComment.mentionedUsers) {
        const userIds = editingComment.mentionedUsers
          .map((mu) => mu.mentionedUserId)
          .filter(Boolean) as string[];
        setMentionedUserIds(userIds);
      }
    }
  }, [isEditMode, editingComment, form]);

  const {
    control,
    handleSubmit,
    reset,
    watch,
    setError,
    formState: { isSubmitting, errors, isSubmitSuccessful },
  } = form;

  const { mutateAsync } = useApiMutation({
    setFormError: setError,
    resourceName: "Shipment Comment",
    mutationFn: async (values: ShipmentCommentSchema) => {
      if (isEditMode && editingComment?.id) {
        const response = await api.shipments.updateComment(
          editingComment.id,
          values,
        );
        return response;
      } else {
        const response = await api.shipments.addComment(shipmentId, values);
        return response;
      }
    },
    onSuccess: () => {
      toast.success(
        isEditMode
          ? "Comment updated successfully"
          : "Comment added successfully",
      );

      queryClient.invalidateQueries({
        queryKey: queries.shipment.listComments(shipmentId).queryKey,
      });

      if (isEditMode) {
        clearEditMode();
      }
    },
  });

  const commentValue = watch("comment");
  const hasContent = commentValue.trim().length > 0;

  // Check if there's an incomplete slash command (when a comment type already exists)
  const hasIncompleteSlashCommand = useMemo(() => {
    if (!commentType || !commentValue) return false;

    // Check if the comment contains a slash command pattern
    const slashPattern = /\/\w*(?:\s|$)/;
    return slashPattern.test(commentValue);
  }, [commentValue, commentType]);

  const onSubmit = useCallback(
    async (values: ShipmentCommentSchema) => {
      try {
        // Only prepend comment type to text if we're NOT using JSON
        let finalComment = values.comment;
        if (commentType && !commentJson) {
          const type = COMMENT_TYPES.find((t) => t.value === commentType);
          if (type) {
            finalComment = `/${type.label} ${finalComment}`;
          }
        }

        const payload = {
          ...values,
          comment: finalComment,
          mentionedUsers: mentionedUserIds.map((userId) => ({
            mentionedUserId: userId,
          })),
          commentType: commentType,
          shipmentId: editingComment?.shipmentId,
          metadata: {
            editorContent: commentJson,
            version: "1.0", // Version the schema for future compatibility
          },
        };
        await mutateAsync(payload);
      } catch (error) {
        console.error("Failed to submit comment:", error);
      }
    },
    [mutateAsync, mentionedUserIds, commentType, commentJson, editingComment],
  );

  const handleCancel = () => {
    reset();
    setMentionedUserIds([]);
    setCommentType(null);
    setCommentJson(null);
    clearEditMode();
  };

  useEffect(() => {
    if (isSubmitSuccessful) {
      reset();
      setMentionedUserIds([]);
      setCommentType(null);
      setCommentJson(null);
      console.log("isEditMode", isEditMode);
      // Clear edit mode after successful submission
      if (isEditMode) {
        clearEditMode();
      }
    }
  }, [isSubmitSuccessful, reset, isEditMode, clearEditMode]);

  // Clear edit mode and reset form when shipment changes
  useEffect(() => {
    clearEditMode();
    reset();
    setMentionedUserIds([]);
    setCommentType(null);
    setCommentJson(null);
  }, [shipmentId, clearEditMode, reset]);

  return (
    <FormProvider {...form}>
      <Form
        className="flex flex-col px-2 gap-2"
        onSubmit={handleSubmit(onSubmit)}
      >
        <FormGroup cols={1}>
          <FormControl>
            <Controller
              name="comment"
              control={control}
              render={({ field, fieldState }) => (
                <TiptapEditor
                  value={field.value}
                  onChange={field.onChange}
                  onJsonChange={setCommentJson}
                  onMentionedUsersChange={setMentionedUserIds}
                  onCommentTypeChange={setCommentType}
                  searchUsers={searchUsers}
                  hasIncompleteSlashCommand={hasIncompleteSlashCommand}
                  placeholder="Add a comment... Use @ to mention users, / for comment types"
                  disabled={isSubmitting}
                  isInvalid={!!fieldState.error}
                />
              )}
            />
            {errors.comment && (
              <p className="text-sm text-red-500 mt-1">
                {errors.comment.message}
              </p>
            )}
          </FormControl>
        </FormGroup>
        <div className="flex justify-end gap-2">
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={handleCancel}
            disabled={!hasContent || isSubmitting}
          >
            Cancel
          </Button>
          <Button
            type="submit"
            size="sm"
            disabled={!hasContent || isSubmitting || hasIncompleteSlashCommand}
            onClick={(e) => {
              e.preventDefault();
              e.stopPropagation();

              handleSubmit(onSubmit)(e);
            }}
          >
            {isSubmitting
              ? isEditMode
                ? "Updating..."
                : "Posting..."
              : isEditMode
                ? "Update Comment"
                : "Post Comment"}
          </Button>
        </div>
      </Form>
    </FormProvider>
  );
}
