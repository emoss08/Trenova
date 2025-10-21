/*
 * Copyright 2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { useFormSave } from "@/components/form/form-save-context";
import { handleMutationError, useApiMutation } from "@/hooks/use-api-mutation";
import { APIError } from "@/types/errors";
import { useCallback } from "react";
import { FieldValues, useForm, UseFormProps } from "react-hook-form";
import { toast } from "sonner";

interface UseFormWithSaveOptions<
  TFormValues extends FieldValues,
  TData,
  TContext,
> {
  /** The name of the resource to use for error handling */
  resourceName: string;

  /** react-hook-form options */
  formOptions: UseFormProps<any>;

  /** Mutation function to call when the form is submitted */
  mutationFn: (data: TFormValues) => Promise<TData>;

  /** Optional function to call before submitting the form */
  onBeforeSubmit?: (data: TFormValues) => TFormValues;

  /** Optional function to call after form submission is successful */
  onSuccess?: (
    data: TData,
    variables: TFormValues,
    context: TContext | undefined,
  ) => Promise<unknown> | unknown;

  /** Optional function to call after form submission is settled */
  onSettled?: (
    data: TData | undefined,
    error: APIError | null,
    variables: TFormValues,
    context: TContext | undefined,
  ) => Promise<unknown> | unknown;

  /** Optional function to call if form submission fails */
  onError?: (
    error: APIError,
    variables: TFormValues,
    context: TContext | undefined,
  ) => Promise<unknown> | unknown;

  /** Optional success message to display */
  successMessage?: string;

  /** Optional success description to display */
  successDescription?: string;
}

/**
 * useFormWithSave - A hook that combines react-hook-form with save functionality
 *
 * This hook wraps useForm from react-hook-form and adds save functionality,
 * including tracking when the form was last saved and handling form submission.
 */
export function useFormWithSave<
  TFormValues extends FieldValues,
  TData = unknown,
  TContext = unknown,
>({
  resourceName,
  formOptions,
  mutationFn,
  onBeforeSubmit,
  onSuccess,
  onError,
  onSettled,
  successMessage = "Changes have been saved",
  successDescription = "Your changes have been saved successfully",
}: UseFormWithSaveOptions<TFormValues, TData, TContext>) {
  const { setLastSavedNow } = useFormSave();

  const form = useForm({
    ...formOptions,
    mode: formOptions.mode || "onChange",
  });

  const { mutateAsync, isPending } = useApiMutation<
    TData,
    TFormValues,
    TContext
  >({
    mutationFn,
    onSuccess: (data, variables, context) => {
      // Show success toast
      toast.success(successMessage, {
        description: successDescription,
      });

      // Update last saved timestamp
      setLastSavedNow();

      // Call custom onSuccess handler if provided
      if (onSuccess) {
        onSuccess(data, variables, context);
      }
    },
    onError: (error: APIError, variables, context) => {
      // Call custom onError handler if provided
      if (onError) {
        onError(error, variables, context);
      }

      handleMutationError<TFormValues>({
        error,
        setFormError: form.setError,
        resourceName,
      });
    },
    onSettled: (data, error, variables, context) => {
      if (onSettled) {
        onSettled(data, error, variables, context);
      }
    },
  });

  const handleSubmit = useCallback(
    async (values: TFormValues) => {
      try {
        // Process data before submission if needed
        const processedValues = onBeforeSubmit
          ? onBeforeSubmit(values)
          : values;

        // Submit the form
        await mutateAsync(processedValues);
      } catch (error) {
        console.error("Form submission error:", error);
      }
    },
    [mutateAsync, onBeforeSubmit],
  );

  return {
    ...form,
    isSubmitting: isPending,
    onSubmit: handleSubmit,
  };
}
