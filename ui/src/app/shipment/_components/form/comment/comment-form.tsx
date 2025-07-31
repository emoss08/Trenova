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
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback, useState } from "react";
import { Controller, FormProvider, useForm } from "react-hook-form";
import { toast } from "sonner";
import { MentionTextarea } from "./mention-textarea";
import { CommentType } from "./utils";

interface CommentFormProps {
  searchUsers: (query: string) => Promise<UserSchema[]>;
  shipmentId: ShipmentSchema["id"];
  isReply?: boolean;
}

export function CommentForm({
  searchUsers,
  shipmentId,
  isReply,
}: CommentFormProps) {
  const queryClient = useQueryClient();
  const [mentionedUserIds, setMentionedUserIds] = useState<string[]>([]);
  const [commentType, setCommentType] = useState<CommentType | null>(null);

  const form = useForm({
    resolver: zodResolver(shipmentCommentSchema),
    defaultValues: {
      comment: "",
      commentType: null,
    },
  });

  const {
    control,
    handleSubmit,
    reset,
    watch,
    setError,
    formState: { isSubmitting, errors },
  } = form;

  const { mutateAsync } = useApiMutation({
    setFormError: setError,
    resourceName: "Shipment Comment",
    mutationFn: async (values: ShipmentCommentSchema) => {
      const response = await api.shipments.addComment(shipmentId, values);
      return response;
    },
    onSuccess: () => {
      toast.success("Comment added successfully");

      queryClient.invalidateQueries({
        queryKey: queries.shipment.listComments(shipmentId).queryKey,
      });

      reset();
      setMentionedUserIds([]);
      setCommentType(null);
    },
  });

  const commentValue = watch("comment");
  const hasContent = commentValue.trim().length > 0;

  const onSubmit = useCallback(
    async (values: ShipmentCommentSchema) => {
      try {
        const payload = {
          ...values,
          mentionedUsers: mentionedUserIds.map((userId) => ({
            mentionedUserId: userId,
          })),
          commentType: commentType,
        };
        await mutateAsync(payload);
      } catch (error) {
        console.error("Failed to submit comment:", error);
      }
    },
    [mutateAsync, mentionedUserIds, commentType],
  );

  const handleCancel = () => {
    reset();
    setMentionedUserIds([]);
  };

  return (
    <FormProvider {...form}>
      <Form className="flex flex-col gap-2" onSubmit={handleSubmit(onSubmit)}>
        <FormGroup cols={1}>
          <FormControl>
            <Controller
              name="comment"
              control={control}
              render={({ field, fieldState }) => (
                <MentionTextarea
                  isReply={isReply}
                  value={field.value}
                  onChange={field.onChange}
                  onMentionedUsersChange={setMentionedUserIds}
                  onCommentTypeChange={setCommentType}
                  searchUsers={searchUsers}
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
            disabled={!hasContent || isSubmitting}
            onClick={(e) => {
              e.preventDefault();
              e.stopPropagation();
              handleSubmit(onSubmit)(e);
            }}
          >
            {isSubmitting ? "Posting..." : "Post Comment"}
          </Button>
        </div>
      </Form>
    </FormProvider>
  );
}
