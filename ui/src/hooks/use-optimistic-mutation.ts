import type { APIError } from "@/types/errors";
import {
  useMutation,
  useQueryClient,
  type QueryKey,
} from "@tanstack/react-query";
import type {
  DefaultValues,
  FieldValues,
  UseFormReset,
  UseFormSetError,
} from "react-hook-form";
import { toast } from "sonner";
import { handleMutationError } from "./use-api-mutation";
import { broadcastQueryInvalidation } from "./use-invalidate-query";

type OptimisticContext<TData = unknown> = {
  previousData: TData;
  newValues: unknown;
};

export type OptimisticMutationOptions<
  TData,
  TVariables,
  TContext = unknown,
  TFormValues extends FieldValues = FieldValues,
> = {
  queryKey: QueryKey;
  mutationFn: (data: TVariables) => Promise<TData>;
  successMessage: string;
  resourceName: string;
  setFormError?: UseFormSetError<TFormValues>;
  resetForm?: UseFormReset<TFormValues>;
  // * Optional: Additional query keys to invalidate on success/settle
  invalidateQueries?: QueryKey[];
  // * Optional: Override default optimistic update behavior
  optimisticUpdate?: (variables: TVariables, currentData: unknown) => unknown;
  // * Optional: Additional callbacks for custom behavior
  onMutate?: (
    variables: TVariables,
  ) => Promise<TContext | undefined> | TContext | undefined;
  onSuccess?: (
    data: TData,
    variables: TVariables,
    context: TContext | undefined,
  ) => Promise<unknown> | unknown;
  onError?: (
    error: APIError,
    variables: TVariables,
    context: TContext | undefined,
  ) => Promise<unknown> | unknown;
  onSettled?: (
    data: TData | undefined,
    error: APIError | null,
    variables: TVariables,
    context: TContext | undefined,
  ) => Promise<unknown> | unknown;
};

export function useOptimisticMutation<
  TData,
  TVariables,
  TContext = OptimisticContext<TData>,
  TFormValues extends FieldValues = FieldValues,
>(
  options: OptimisticMutationOptions<TData, TVariables, TContext, TFormValues>,
) {
  const {
    setFormError,
    resourceName,
    invalidateQueries,
    optimisticUpdate,
    onMutate,
    onSuccess,
    onError,
    onSettled,
    resetForm,
  } = options;

  const queryClient = useQueryClient();

  return useMutation<TData, APIError, TVariables, any>({
    mutationFn: options.mutationFn,
    onMutate: async (variables) => {
      // * Call custom onMutate if provided - allows complete override
      if (onMutate) {
        const customContext = await onMutate(variables);
        if (customContext !== undefined) {
          return customContext;
        }
      }

      // * Default optimistic update behavior
      // * Cancel any outgoing refetches so they don't overwrite our optimistic update
      await queryClient.cancelQueries({
        queryKey: options.queryKey,
      });

      // * Snapshot the previous value
      const previousData = queryClient.getQueryData(options.queryKey);

      // * Optimistically update to the new value
      if (optimisticUpdate) {
        // * Use custom optimistic update function if provided
        const newData = optimisticUpdate(variables, previousData);
        queryClient.setQueryData(options.queryKey, newData);
      } else {
        // * Default: replace entire data with new values
        queryClient.setQueryData(options.queryKey, variables);
      }

      return { previousData, newValues: variables } as any;
    },
    onSuccess: async (data: TData, variables, context) => {
      // * Always show success toast
      toast.success(options.successMessage);

      // * Always broadcast query invalidation for the main query
      broadcastQueryInvalidation({
        queryKey: options.queryKey as unknown as string[],
        options: {
          correlationId: `update-${resourceName}-${Date.now()}`,
        },
        config: {
          predicate: true,
          refetchType: "all",
        },
      });

      // * Reset the form to the new values if resetForm is provided
      resetForm?.(data as DefaultValues<TFormValues>);

      // * Call custom onSuccess if provided
      if (onSuccess) {
        await onSuccess(data, variables, context as TContext);
      }
    },
    onError: async (error: APIError, variables, context) => {
      // * Rollback optimistic update on error
      if (context && typeof context === "object" && "previousData" in context) {
        queryClient.setQueryData(options.queryKey, context.previousData);
      }

      // * Standard error handling
      handleMutationError({
        error,
        setFormError,
        resourceName,
      });

      // * Custom error handling if provided
      if (onError) {
        await onError(error, variables, context as TContext);
      }
    },
    onSettled: async (data, error, variables, context) => {
      // * Invalidate the main query to ensure fresh data
      await queryClient.invalidateQueries({
        queryKey: options.queryKey,
      });

      // * Invalidate any additional queries specified
      if (invalidateQueries && invalidateQueries.length > 0) {
        await Promise.all(
          invalidateQueries.map((queryKey) =>
            queryClient.invalidateQueries({ queryKey }),
          ),
        );
      }

      // * Call custom onSettled if provided
      if (onSettled) {
        await onSettled(data, error, variables, context as TContext);
      }
    },
  });
}
