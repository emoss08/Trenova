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
  invalidateQueries?: QueryKey[];
  optimisticUpdate?: (variables: TVariables, currentData: unknown) => unknown;
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
      if (onMutate) {
        const customContext = await onMutate(variables);
        if (customContext !== undefined) {
          return customContext;
        }
      }

      await queryClient.cancelQueries({
        queryKey: options.queryKey,
      });

      const previousData = queryClient.getQueryData(options.queryKey);

      if (optimisticUpdate) {
        const newData = optimisticUpdate(variables, previousData);
        queryClient.setQueryData(options.queryKey, newData);
      } else {
        queryClient.setQueryData(options.queryKey, variables);
      }

      return { previousData, newValues: variables } as any;
    },
    onSuccess: async (data: TData, variables, context) => {
      toast.success(options.successMessage);

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

      resetForm?.(data as DefaultValues<TFormValues>);

      if (onSuccess) {
        await onSuccess(data, variables, context as TContext);
      }
    },
    onError: async (error: APIError, variables, context) => {
      if (context && typeof context === "object" && "previousData" in context) {
        queryClient.setQueryData(options.queryKey, context.previousData);
      }

      handleMutationError({
        error,
        setFormError,
        resourceName,
      });

      if (onError) {
        await onError(error, variables, context as TContext);
      }
    },
    onSettled: async (data, error, variables, context) => {
      await queryClient.invalidateQueries({
        queryKey: options.queryKey,
      });

      if (invalidateQueries && invalidateQueries.length > 0) {
        await Promise.all(
          invalidateQueries.map((queryKey) =>
            queryClient.invalidateQueries({ queryKey }),
          ),
        );
      }

      if (onSettled) {
        await onSettled(data, error, variables, context as TContext);
      }
    },
  });
}
