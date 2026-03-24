import type { ApiRequestError } from "@/lib/api";
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

type OptimisticContext<TQueryData = unknown, TVariables = unknown> = {
  previousData: TQueryData | undefined;
  newValues: TVariables;
};

export type OptimisticMutationOptions<
  TData,
  TVariables,
  TQueryData = unknown,
  TFormValues extends FieldValues = FieldValues,
  TContext = OptimisticContext<TQueryData, TVariables>,
> = {
  queryKey: QueryKey;
  mutationFn: (data: TVariables) => Promise<TData>;
  successMessage?: string;
  resourceName: string;
  setFormError?: UseFormSetError<TFormValues>;
  resetForm?: UseFormReset<TFormValues>;
  invalidateQueries?: QueryKey[];
  optimisticUpdate?: (
    variables: TVariables,
    currentData: TQueryData | undefined,
  ) => TQueryData;
  onMutate?: (
    variables: TVariables,
  ) => Promise<TContext | undefined> | TContext | undefined;
  onSuccess?: (
    data: TData,
    variables: TVariables,
    context: TContext | undefined,
  ) => unknown;
  onError?: (
    error: ApiRequestError,
    variables: TVariables,
    context: TContext | undefined,
  ) => unknown;
  onSettled?: (
    data: TData | undefined,
    error: ApiRequestError | null,
    variables: TVariables,
    context: TContext | undefined,
  ) => unknown;
};

export function useOptimisticMutation<
  TData,
  TVariables,
  TQueryData = unknown,
  TFormValues extends FieldValues = FieldValues,
  TContext = OptimisticContext<TQueryData, TVariables>,
>(
  options: OptimisticMutationOptions<
    TData,
    TVariables,
    TQueryData,
    TFormValues,
    TContext
  >,
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

  return useMutation<
    TData,
    ApiRequestError,
    TVariables,
    OptimisticContext<TQueryData, TVariables>
  >({
    mutationFn: options.mutationFn,
    onMutate: async (variables) => {
      if (onMutate) {
        const customContext = await onMutate(variables);
        if (customContext !== undefined) {
          return customContext as OptimisticContext<TQueryData, TVariables>;
        }
      }

      await queryClient.cancelQueries({
        queryKey: options.queryKey,
      });

      const previousData = queryClient.getQueryData<TQueryData>(
        options.queryKey,
      );

      if (optimisticUpdate) {
        const newData = optimisticUpdate(variables, previousData);
        queryClient.setQueryData(options.queryKey, newData);
      }

      return { previousData, newValues: variables };
    },
    onSuccess: async (data: TData, variables, context) => {
      if (options.successMessage) {
        toast.success(options.successMessage);
      }

      resetForm?.(data as DefaultValues<TFormValues>);

      if (onSuccess) {
        await onSuccess(data, variables, context as TContext);
      }
    },
    onError: async (error: ApiRequestError, variables, context) => {
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
