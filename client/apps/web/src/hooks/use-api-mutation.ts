import { ApiRequestError } from "@trenova/shared/lib/api";
import { GraphQLRequestError } from "@trenova/shared/lib/graphql";
import {
  type NormalizedApiError,
  type ValidationError,
  apiProblem,
} from "@trenova/shared/types/errors";
import { useMutation, type UseMutationOptions } from "@tanstack/react-query";
import type { FieldValues, Path, UseFormSetError } from "react-hook-form";
import { toast } from "sonner";

type MutationErrorOptions<T extends FieldValues> = {
  error: unknown;
  setFormError?: UseFormSetError<T>;
  resourceName?: string;
};

function normalizeMutationError(error: unknown): NormalizedApiError | null {
  if (error instanceof ApiRequestError || error instanceof GraphQLRequestError) {
    return error.normalize();
  }
  return null;
}

export function handleMutationError<T extends FieldValues>({
  error,
  setFormError,
  resourceName,
}: MutationErrorOptions<T>): void {
  const normalized = normalizeMutationError(error);

  if (!normalized) {
    if (resourceName) {
      console.error(`Error handling ${resourceName}:`, error);
    }

    const description =
      error instanceof Error ? error.message : "An unexpected error occurred";

    toast.error("Error", {
      description,
    });
    return;
  }

  if (apiProblem.isVersionMismatchError(normalized)) {
    toast.error("Version mismatch", {
      description:
        "The resource has been modified. Please refresh and try again.",
    });
    return;
  }

  if (apiProblem.isValidationError(normalized) && setFormError) {
    normalized.fieldErrors.forEach((fieldError: ValidationError) => {
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

  if (apiProblem.isBusinessError(normalized)) {
    toast.error("Invalid Operation", {
      description: normalized.message,
    });
    return;
  }

  if (apiProblem.isRateLimitError(normalized)) {
    toast.error("Rate limit exceeded", {
      description: "Please wait a moment and try again.",
    });
    return;
  }

  if (apiProblem.isAuthenticationError(normalized)) {
    toast.error(normalized.title ?? "Authentication required", {
      description: normalized.detail ?? normalized.message,
    });
    return;
  }

  if (apiProblem.isAuthorizationError(normalized)) {
    toast.error("Access denied", {
      description: "You don't have permission to perform this action.",
    });
    return;
  }

  if (apiProblem.isNotFoundError(normalized)) {
    toast.error("Not found", {
      description: normalized.detail || "The requested resource was not found.",
    });
    return;
  }

  if (resourceName) {
    console.error(`Error handling ${resourceName}:`, error);
  }

  toast.error("Error", {
    description: normalized.message,
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
    error: unknown,
    variables: TVariables,
    context: TContext | undefined,
  ) => unknown;
} & Omit<
  UseMutationOptions<TData, unknown, TVariables, TContext>,
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
  return useMutation<TData, unknown, TVariables, TContext>({
    ...options,
    onError: (error: unknown, variables, context) => {
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
