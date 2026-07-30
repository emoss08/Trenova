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
import { isRegisteredLeaf, normalizeFieldPath, type FieldRegistry } from "@/lib/form-errors";

// Only the field registry and the error setter are needed, so the parameter asks for
// exactly those. Taking the whole UseFormReturn would drag in handleSubmit, whose
// transformed-values generic makes a form declared with an output type unassignable here.
export type MutationFormTarget<T extends FieldValues> = {
  // Deliberately not Control<T>. react-hook-form's Control reaches deep enough into its
  // own internals that TypeScript stops relating two instantiations whose
  // transformed-values generic differs, so any form declared with an output type would be
  // rejected here. Only the field registry is read, so only that is asked for.
  control: FieldRegistry;
  setError: UseFormSetError<T>;
};

type MutationErrorOptions<T extends FieldValues> = {
  error: unknown;
  form?: MutationFormTarget<T>;
  resourceName?: string;
};

// applyFieldErrors distributes server validation errors across a form.
//
// First-wins per field: the server emits presence checks before semantic ones, so when a
// field is blank "Pro number is required" precedes "PRO 1234 is already in use" and the
// former is the message that tells the user what to do. Skipping paths already set is
// also what makes setError, getFieldError and normalize().detail agree — a forEach that
// overwrote would leave the field showing the last message while both accessors reported
// the first.
//
// Anything that is not a registered leaf goes to the form root: a non-leaf path like
// `lines[0]`, the empty field the server emits for a whole-object failure, or a field the
// form simply does not have such as `status`. Those must never be dropped and must never
// be set on a path no input renders, because either way the user is left with a form that
// refuses to submit and gives no reason.
function applyFieldErrors<T extends FieldValues>(
  form: MutationFormTarget<T>,
  fieldErrors: ValidationError[],
): void {
  const assigned = new Set<string>();
  // A Set both dedupes and preserves insertion order, so the joined message keeps the
  // server's ordering without an O(n) membership scan per error.
  const rootMessages = new Set<string>();

  for (const fieldError of fieldErrors) {
    const path = normalizeFieldPath(fieldError.field);

    if (!isRegisteredLeaf(form.control, path)) {
      rootMessages.add(fieldError.message);
      continue;
    }

    if (assigned.has(path)) {
      continue;
    }
    assigned.add(path);

    try {
      form.setError(path as Path<T>, { message: fieldError.message, type: "validation" });
    } catch (e) {
      console.error(`Error setting form error for field ${fieldError.field}:`, e);
      rootMessages.add(fieldError.message);
    }
  }

  if (rootMessages.size > 0) {
    const message = [...rootMessages].join(" ");
    // Root is react-hook-form's channel for form-wide errors: FormRootError renders it,
    // and handleSubmit discards it on the next attempt so it cannot outlive the failure.
    // Setting these on their reported paths instead would land them on nodes no input
    // renders, where they are invisible AND block submission forever.
    //
    // The toast is not redundant. Inline rendering needs the form inside a FormProvider,
    // and 23 of the forms reaching this code render their fields through a parent's
    // provider rather than their own. Render-tree topology must not decide whether the
    // user learns why the save failed.
    form.setError("root", { message, type: "validation" });
    toast.error("This change could not be saved", { description: message });
  }
}

function normalizeMutationError(error: unknown): NormalizedApiError | null {
  if (error instanceof ApiRequestError || error instanceof GraphQLRequestError) {
    return error.normalize();
  }
  return null;
}

export function handleMutationError<T extends FieldValues>({
  error,
  form,
  resourceName,
}: MutationErrorOptions<T>): void {
  const normalized = normalizeMutationError(error);

  if (!normalized) {
    if (resourceName) {
      console.error(`Error handling ${resourceName}:`, error);
    }

    const description = error instanceof Error ? error.message : "An unexpected error occurred";

    toast.error("Error", {
      description,
    });
    return;
  }

  if (apiProblem.isVersionMismatchError(normalized)) {
    toast.error("Version mismatch", {
      description: "The resource has been modified. Please refresh and try again.",
    });
    return;
  }

  if (apiProblem.isValidationError(normalized) && form) {
    applyFieldErrors(form, normalized.fieldErrors);
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

type UseApiMutationOptions<TData, TVariables, TContext, TFormValues extends FieldValues> = {
  form?: MutationFormTarget<TFormValues>;
  resourceName?: string;
  onError?: (error: unknown, variables: TVariables, context: TContext | undefined) => unknown;
} & Omit<UseMutationOptions<TData, unknown, TVariables, TContext>, "onError">;

export function useApiMutation<
  TData,
  TVariables,
  TContext = unknown,
  TFormValues extends FieldValues = FieldValues,
>({
  form,
  resourceName,
  onError,
  ...options
}: UseApiMutationOptions<TData, TVariables, TContext, TFormValues>) {
  return useMutation<TData, unknown, TVariables, TContext>({
    ...options,
    onError: (error: unknown, variables, context) => {
      handleMutationError<TFormValues>({
        error,
        form,
        resourceName,
      });

      if (onError) {
        onError(error, variables, context);
      }
    },
  });
}
