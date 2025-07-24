/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { InputField } from "@/components/fields/input-field";
import { SwitchField } from "@/components/fields/switch-field";
import { WorkerAutocompleteField } from "@/components/ui/autocomplete-fields";
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
import { Form, FormControl, FormGroup } from "@/components/ui/form";
import { Icon } from "@/components/ui/icons";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { broadcastQueryInvalidation } from "@/hooks/use-invalidate-query";
import { queries } from "@/lib/queries";
import {
  suggestionAcceptRequestSchema,
  SuggestionAcceptRequestSchema,
  type DedicatedLaneSuggestionSchema,
} from "@/lib/schemas/dedicated-lane-schema";
import { api } from "@/services/api";
import type { TableSheetProps } from "@/types/data-table";
import { faSparkle } from "@fortawesome/pro-solid-svg-icons";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";
import {
  FormProvider,
  useForm,
  useFormContext,
  useWatch,
} from "react-hook-form";
import { toast } from "sonner";

type AcceptSuggestionDialogProps = TableSheetProps & {
  suggestion: DedicatedLaneSuggestionSchema;
};

export function AcceptSuggestionDialog({
  open,
  onOpenChange,
  suggestion,
}: AcceptSuggestionDialogProps) {
  const queryClient = useQueryClient();
  const form = useForm<SuggestionAcceptRequestSchema>({
    resolver: zodResolver(suggestionAcceptRequestSchema),
    defaultValues: {
      id: suggestion.id,
      autoAssign: false,
      dedicatedLaneName: "",
      primaryWorkerId: "",
      secondaryWorkerId: "",
    },
  });

  const {
    setError,
    formState: { isSubmitting },
    handleSubmit,
    reset,
  } = form;

  const { mutateAsync: acceptSuggestion } = useApiMutation({
    setFormError: setError,
    resourceName: "Dedicated Lane Suggestion",
    mutationFn: async (values: SuggestionAcceptRequestSchema) => {
      const response = await api.dedicatedLaneSuggestions.acceptSuggestion(
        suggestion.id,
        values,
      );

      return response;
    },
    onSuccess: () => {
      toast.success("Suggestion accepted");
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

  const onAcceptSuggestion = useCallback(
    async (values: SuggestionAcceptRequestSchema) => {
      await acceptSuggestion(values);
    },
    [acceptSuggestion],
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
          <Form onSubmit={handleSubmit(onAcceptSuggestion)}>
            <DialogBody>
              <AcceptSuggestionForm suggestion={suggestion} />
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
                onClick={() => handleSubmit(onAcceptSuggestion)()}
                isSubmitting={isSubmitting}
                title="dedicated lane suggestion"
                text="Accept"
              />
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}

function AcceptSuggestionForm({
  suggestion,
}: {
  suggestion: DedicatedLaneSuggestionSchema;
}) {
  const { control, setValue } = useFormContext<SuggestionAcceptRequestSchema>();
  const autoAssign = useWatch({
    control,
    name: "autoAssign",
  });

  const handleSuggestionClick = useCallback(() => {
    setValue("dedicatedLaneName", suggestion.suggestedName);
  }, [suggestion.suggestedName, setValue]);

  return (
    <FormGroup>
      <FormControl>
        <InputField
          name="dedicatedLaneName"
          control={control}
          rightElement={
            <TooltipProvider>
              <Tooltip>
                <TooltipTrigger asChild>
                  <span
                    className="[&>svg]:size-3 size-5 mr-1 rounded-md flex items-center justify-center hover:bg-purple-500/30 text-muted-foreground hover:text-foreground transition-colors duration-200 ease-in-out cursor-pointer"
                    onClick={(e) => {
                      e.stopPropagation();
                      handleSuggestionClick();
                    }}
                  >
                    <Icon icon={faSparkle} className="size-4 text-purple-500" />
                  </span>
                </TooltipTrigger>
                <TooltipContent>
                  <p>Use system suggested name</p>
                </TooltipContent>
              </Tooltip>
            </TooltipProvider>
          }
          label="Dedicated Lane Name"
          placeholder="Enter dedicated lane name"
          rules={{ required: true }}
          description="The name of the dedicated lane to create."
        />
      </FormControl>
      <FormControl>
        <SwitchField
          name="autoAssign"
          control={control}
          outlined
          label="Auto Assign"
          recommended
          description="Automatically assign the primary and secondary workers to the dedicated lane."
        />
      </FormControl>
      {autoAssign && (
        <>
          <FormControl>
            <WorkerAutocompleteField<SuggestionAcceptRequestSchema>
              name="primaryWorkerId"
              control={control}
              rules={{ required: autoAssign }}
              label="Primary Worker"
              placeholder="Select Primary Worker"
              description="The primary worker for the dedicated lane."
            />
          </FormControl>
          <FormControl>
            <WorkerAutocompleteField<SuggestionAcceptRequestSchema>
              name="secondaryWorkerId"
              control={control}
              label="Secondary Worker"
              clearable
              placeholder="Select Secondary Worker"
              description="The secondary worker for the dedicated lane."
            />
          </FormControl>
        </>
      )}
    </FormGroup>
  );
}
