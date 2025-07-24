/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

// src/hooks/use-api-mutation.ts
import { APIError } from "@/types/errors";
import { UseMutationOptions, useMutation } from "@tanstack/react-query";
import { FieldValues, Path, UseFormSetError } from "react-hook-form";
import { toast } from "sonner";

/**
 * Handles common API errors in mutations
 */
export function handleMutationError<TFormValues extends FieldValues>({
  error,
  setFormError,
  resourceName,
}: {
  error: APIError;
  setFormError?: UseFormSetError<TFormValues>;
  resourceName?: string;
}): void {
  const apiError = error instanceof APIError ? error : null;

  // Handle validation errors by setting form errors
  if (apiError?.isValidationError() && setFormError) {
    // * if it is a version mismatch error, we don't need to set the form error
    if (apiError.isVersionMismatchError()) {
      toast.error("Version mismatch", {
        description:
          "The version of the resource you are trying to update is outdated. Please refresh the page and try again.",
      });
      return;
    }

    apiError.getFieldErrors().forEach((fieldError) => {
      try {
        setFormError(fieldError.name as Path<TFormValues>, {
          message: fieldError.reason,
        });
      } catch (e) {
        console.error(
          `Error setting form error for field ${fieldError.name}:`,
          e,
        );
      }
    });
  }

  // Handle rate limit errors
  if (apiError?.isRateLimitError()) {
    toast.error("Rate limit exceeded", {
      description: "You have exceeded the rate limit. Please try again later.",
    });
  }

  // Handle business errors
  if (apiError?.isBusinessError()) {
    toast.error("Invalid Operation", {
      description: apiError.data?.detail,
    });
  }

  // Log with context if resourceName is provided
  if (resourceName) {
    console.error(`Error handling ${resourceName}:`, apiError);
  }
}

/**
 * Custom hook for API mutations with standardized error handling
 */
export function useApiMutation<
  TData, // Type of successful response data
  TVariables, // Type of mutation variables
  TContext = unknown, // Type of context data
  TFormValues extends FieldValues = FieldValues, // Type of form values
>({
  setFormError,
  resourceName,
  onError,
  ...options
}: {
  setFormError?: UseFormSetError<TFormValues>;
  resourceName?: string;
  onError?: (
    error: APIError,
    variables: TVariables,
    context: TContext | undefined,
  ) => Promise<unknown> | unknown;
} & Omit<
  UseMutationOptions<TData, APIError, TVariables, TContext>,
  "onError"
>) {
  return useMutation<TData, APIError, TVariables, TContext>({
    ...options,
    onError: (error: APIError, variables, context) => {
      // Standard error handling
      handleMutationError<TFormValues>({
        error,
        setFormError,
        resourceName,
      });

      // Custom error handling if provided
      if (onError) {
        onError(error, variables, context);
      }
    },
  });
}
