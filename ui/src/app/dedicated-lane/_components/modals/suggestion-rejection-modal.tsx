import { TextareaField } from "@/components/fields/textarea-field";
import { Button, FormSaveButton } from "@/components/ui/button";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Form, FormGroup } from "@/components/ui/form";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { broadcastQueryInvalidation } from "@/hooks/use-invalidate-query";
import { queries } from "@/lib/queries";
import {
  suggestionRejectRequestSchema,
  type DedicatedLaneSuggestionSchema,
  type SuggestionRejectRequestSchema,
} from "@/lib/schemas/dedicated-lane-schema";
import { api } from "@/services/api";
import type { TableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";
import { FormProvider, useForm, useFormContext } from "react-hook-form";
import { toast } from "sonner";

type RejectSuggestionDialogProps = TableSheetProps & {
  suggestion: DedicatedLaneSuggestionSchema;
};

export function RejectSuggestionDialog({
  open,
  onOpenChange,
  suggestion,
}: RejectSuggestionDialogProps) {
  const queryClient = useQueryClient();
  const form = useForm<SuggestionRejectRequestSchema>({
    resolver: zodResolver(suggestionRejectRequestSchema),
    defaultValues: {
      id: suggestion.id,
      rejectReason: "",
    },
  });

  const {
    setError,
    formState: { isSubmitting },
    handleSubmit,
    reset,
  } = form;

  const { mutateAsync: rejectSuggestion } = useApiMutation({
    setFormError: setError,
    resourceName: "Dedicated Lane Suggestion",
    mutationFn: async (values: SuggestionRejectRequestSchema) => {
      const response = await api.dedicatedLaneSuggestions.rejectSuggestion(
        suggestion.id,
        values,
      );

      return response;
    },
    onSuccess: () => {
      toast.success("Suggestion rejected");
      onOpenChange(false);
      reset();

      queryClient.invalidateQueries({
        queryKey: queries.dedicatedLaneSuggestion.getSuggestions._def,
      });

      // Invalidate the query to refresh the table
      broadcastQueryInvalidation({
        queryKey: ["dedicated-lane"],
        options: {
          correlationId: `accept-dedicated-lane-suggestion-${Date.now()}`,
        },
        config: {
          predicate: true,
          refetchType: "all",
        },
      });
    },
  });

  const onSubmit = useCallback(
    async (values: SuggestionRejectRequestSchema) => {
      await rejectSuggestion(values);
    },
    [rejectSuggestion],
  );

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{suggestion.suggestedName}</DialogTitle>
          <DialogDescription>
            Accept this suggestion to create a dedicated lane.
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form onSubmit={handleSubmit(onSubmit)}>
            <DialogBody>
              <RejectSuggestionForm />
            </DialogBody>
            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => onOpenChange(false)}
              >
                Cancel
              </Button>
              <FormSaveButton
                type="button"
                onClick={() => handleSubmit(onSubmit)()}
                isSubmitting={isSubmitting}
                title="dedicated lane suggestion"
                text="Reject"
              />
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}

function RejectSuggestionForm() {
  const { control } = useFormContext<SuggestionRejectRequestSchema>();

  return (
    <FormGroup cols={1}>
      <TextareaField
        name="rejectReason"
        label="Reject Reason"
        control={control}
        placeholder="Enter a reason for rejecting this suggestion"
        rows={4}
      />
    </FormGroup>
  );
}
