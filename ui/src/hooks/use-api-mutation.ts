import { APIError, InvalidParam } from "@/types/errors";
import { UseMutationOptions, useMutation } from "@tanstack/react-query";
import { FieldValues, Path, UseFormSetError } from "react-hook-form";
import { toast } from "sonner";

interface MutationErrorOptions<TFormValues extends FieldValues> {
  error: APIError;
  setFormError?: UseFormSetError<TFormValues>;
  resourceName?: string;
  allowLowPrioritySubmission?: boolean;
  onLowPriorityErrors?: (errors: InvalidParam[]) => void;
}

/**
 * Handles common API errors in mutations
 */
export function handleMutationError<TFormValues extends FieldValues>({
  error,
  setFormError,
  resourceName,
  allowLowPrioritySubmission = false,
  onLowPriorityErrors,
}: MutationErrorOptions<TFormValues>): void {
  const apiError = error instanceof APIError ? error : null;

  if (apiError?.isValidationError() && setFormError) {
    if (apiError.isVersionMismatchError()) {
      toast.error("Version mismatch", {
        description:
          "The version of the resource you are trying to update is outdated. Please refresh the page and try again.",
      });
      return;
    }

    const highPriorityErrors = apiError.getFieldErrorsByPriority("HIGH");
    const mediumPriorityErrors = apiError.getFieldErrorsByPriority("MEDIUM");
    const lowPriorityErrors = apiError.getFieldErrorsByPriority("LOW");

    [...highPriorityErrors, ...mediumPriorityErrors].forEach((fieldError) => {
      try {
        setFormError(fieldError.name as Path<TFormValues>, {
          message: fieldError.reason,
          type: "validation",
        });
      } catch (e) {
        console.error(
          `Error setting form error for field ${fieldError.name}:`,
          e,
        );
      }
    });

    if (lowPriorityErrors.length > 0) {
      if (allowLowPrioritySubmission && onLowPriorityErrors) {
        onLowPriorityErrors(lowPriorityErrors);
      } else {
        lowPriorityErrors.forEach((warning) => {
          toast.warning("Validation Warning", {
            description: `${warning.name}: ${warning.reason}`,
          });
        });
      }
    }

    apiError
      .getFieldErrors()
      .filter((e) => !e.priority)
      .forEach((fieldError) => {
        try {
          setFormError(fieldError.name as Path<TFormValues>, {
            message: fieldError.reason,
            type: "validation",
          });
        } catch (e) {
          console.error(
            `Error setting form error for field ${fieldError.name}:`,
            e,
          );
        }
      });
  }

  if (apiError?.isRateLimitError()) {
    toast.error("Rate limit exceeded", {
      description: "You have exceeded the rate limit. Please try again later.",
    });
  }

  if (apiError?.isBusinessError()) {
    toast.error("Invalid Operation", {
      description: apiError.data?.detail,
    });
  }

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
  allowLowPrioritySubmission = false,
  onLowPriorityErrors,
  ...options
}: {
  setFormError?: UseFormSetError<TFormValues>;
  resourceName?: string;
  allowLowPrioritySubmission?: boolean;
  onLowPriorityErrors?: (errors: InvalidParam[]) => void;
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
        allowLowPrioritySubmission,
        onLowPriorityErrors,
      });

      // Custom error handling if provided
      if (onError) {
        onError(error, variables, context);
      }
    },
  });
}
