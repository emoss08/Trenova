import {
  keepPreviousData,
  skipToken,
  useQuery,
  type QueryFunctionContext,
  type QueryKey,
  type UseQueryResult,
} from "@tanstack/react-query";
import { useDebounce } from "@trenova/shared/hooks/use-debounce";
import { useWatch, type Control, type FieldValues } from "react-hook-form";

/** How long a form rests before the server is asked what it would book. */
export const PLANNED_PREVIEW_DEBOUNCE_MS = 400;

const IDLE_KEY = ["planned-preview", "idle"] as const;

// queryFn is a method so the tuple-keyed functions a key factory builds stay assignable.
interface PlanQueryOptions<TData> {
  readonly queryKey: QueryKey;
  queryFn(context: QueryFunctionContext): TData | Promise<TData>;
}

export type PlannedPreviewParams<TValues extends FieldValues, TInput, TData> = {
  control: Control<TValues>;
  /** The form's schema; a form that does not pass it asks for no plan. */
  schema: { safeParse: (values: unknown) => { success: boolean } };
  /** The request the form's values make, or null when they make none yet. */
  toInput: (values: TValues) => TInput | null;
  /** The query that plans one request, from the factory the page reads. */
  query: (input: TInput) => PlanQueryOptions<TData>;
};

export type PlannedPreview<TData> = {
  plan: UseQueryResult<TData>;
  /** The form's values make a request the server can plan. */
  hasInput: boolean;
  /** The plan shown is for the values in the form now, not ones typed before. */
  previewing: boolean;
};

/**
 * Asks the server what a form would book while the person edits it: once the
 * values pass the schema and have rested for a moment, the plan for them is
 * read, and the previous plan stays on screen while the next one loads.
 */
export function usePlannedPreview<TValues extends FieldValues, TInput, TData>({
  control,
  schema,
  toInput,
  query,
}: PlannedPreviewParams<TValues, TInput, TData>): PlannedPreview<TData> {
  const values = useWatch({ control }) as TValues;
  const input = schema.safeParse(values).success ? toInput(values) : null;
  const inputKey = input === null ? null : JSON.stringify(input);
  const debouncedKey = useDebounce(inputKey, PLANNED_PREVIEW_DEBOUNCE_MS);
  const options = debouncedKey === null ? null : query(JSON.parse(debouncedKey) as TInput);
  const plan = useQuery<TData, Error, TData, QueryKey>({
    queryKey: options?.queryKey ?? IDLE_KEY,
    queryFn: options ? (context) => options.queryFn(context) : skipToken,
    placeholderData: keepPreviousData,
    retry: false,
  });

  return {
    plan,
    hasInput: inputKey !== null,
    previewing: inputKey !== null && inputKey === debouncedKey,
  };
}
