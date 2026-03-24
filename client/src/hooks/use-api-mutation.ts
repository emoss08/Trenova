import { ApiRequestError } from "@/lib/api";
import type { ValidationError } from "@/types/errors";
import { useMutation, type UseMutationOptions } from "@tanstack/react-query";
import type { FieldValues, Path, UseFormSetError } from "react-hook-form";
import { toast } from "sonner";

type MutationErrorOptions<T extends FieldValues> = {
  error: ApiRequestError;
  setFormError?: UseFormSetError<T>;
  resourceName?: string;
};

export function handleMutationError<T extends FieldValues>({
  error,
  setFormError,
  resourceName,
}: MutationErrorOptions<T>): void {
  if (error.isVersionMismatchError()) {
    toast.error("Version mismatch", {
      description:
        "The resource has been modified. Please refresh and try again.",
    });
    return;
  }

  if (error.isValidationError() && setFormError) {
    const fieldErrors = error.getFieldErrors();
    fieldErrors.forEach((fieldError: ValidationError) => {
      try {
        setFormError(fieldError.field as Path<T>, {
          message: fieldError.message,
          type: "validation",
        });
      } catch (e) {
        console.error(
          `Error setting form error for field ${fieldError.field}:`,
          e,
        );
      }
    });
    return;
  }

  if (error.isBusinessError()) {
    toast.error("Invalid Operation", {
      description: error.data.detail || error.data.title,
    });
    return;
  }

  if (error.isRateLimitError()) {
    toast.error("Rate limit exceeded", {
      description: "Please wait a moment and try again.",
    });
    return;
  }

  if (error.isAuthenticationError()) {
    toast.error(error.data.title, {
      description: error.data?.detail,
    });
    return;
  }

  if (error.isAuthorizationError()) {
    toast.error("Access denied", {
      description: "You don't have permission to perform this action.",
    });
    return;
  }

  if (error.isNotFoundError()) {
    toast.error("Not found", {
      description: error.data.detail || "The requested resource was not found.",
    });
    return;
  }

  if (resourceName) {
    console.error(`Error handling ${resourceName}:`, error);
  }

  toast.error("Error", {
    description: error.data.detail || error.data.title || "An error occurred",
  });
}

type UseApiMutationOptions<
  TData,
  TVariables,
  TContext,
  TFormValues extends FieldValues,
> = {
  setFormError?: UseFormSetError<TFormValues>;
  resourceName?: string;
  onError?: (
    error: ApiRequestError,
    variables: TVariables,
    context: TContext | undefined,
  ) => unknown;
} & Omit<
  UseMutationOptions<TData, ApiRequestError, TVariables, TContext>,
  "onError"
>;

export function useApiMutation<
  TData,
  TVariables,
  TContext = unknown,
  TFormValues extends FieldValues = FieldValues,
>({
  setFormError,
  resourceName,
  onError,
  ...options
}: UseApiMutationOptions<TData, TVariables, TContext, TFormValues>) {
  return useMutation<TData, ApiRequestError, TVariables, TContext>({
    ...options,
    onError: (error: ApiRequestError, variables, context) => {
      handleMutationError<TFormValues>({
        error,
        setFormError,
        resourceName,
      });

      if (onError) {
        onError(error, variables, context);
      }
    },
  });
}
