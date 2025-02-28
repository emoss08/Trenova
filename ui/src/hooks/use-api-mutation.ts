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
  // Handle validation errors by setting form errors
  if (error.isValidationError() && setFormError) {
    error.getFieldErrors().forEach((fieldError) => {
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
  if (error.isRateLimitError()) {
    toast.error("Rate limit exceeded", {
      description: "You have exceeded the rate limit. Please try again later.",
    });
  }

  // Handle business errors
  if (error.isBusinessError()) {
    toast.error("Invalid Operation", {
      description: error.data?.detail,
    });
  }

  // Log with context if resourceName is provided
  if (resourceName) {
    console.error(`Error handling ${resourceName}:`, error);
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
  onError?: (error: APIError) => void;
} & Omit<
  UseMutationOptions<TData, APIError, TVariables, TContext>,
  "onError"
>) {
  return useMutation<TData, APIError, TVariables, TContext>({
    ...options,
    onError: (error: APIError) => {
      // Standard error handling
      handleMutationError<TFormValues>({
        error,
        setFormError,
        resourceName,
      });

      // Custom error handling if provided
      if (onError) {
        onError(error);
      }
    },
  });
}
