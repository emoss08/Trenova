import { useQuery } from "@tanstack/react-query";
import type { PanelMode } from "@trenova/shared/types/data-table";
import { useEffect } from "react";
import type { FieldValues, UseFormReturn } from "react-hook-form";

type EditRecordSource<T> = {
  open: boolean;
  mode: PanelMode;
  /** Cache namespace for the record, e.g. "rate-agreement". */
  queryKey: string;
  id?: string;
  version?: number;
  fetch: (id: string) => Promise<T>;
};

export type EditRecordState = {
  /**
   * Whether the form holds the full record rather than the bare table row.
   *
   * A panel must not submit until this is true. The row it opened from carries
   * no children, so a save sent before the detail lands tells the server the
   * record has none — and the server reads a child it was not sent as one the
   * user deleted.
   */
  isSeated: boolean;
  isLoading: boolean;
  isError: boolean;
};

/**
 * Re-seats an edit panel's form from the full record rather than the table row
 * it was opened from.
 *
 * A table row carries only the columns the list shows; the record's children —
 * lanes, schedules, axes — live behind the detail endpoint. A form reset from
 * the bare row opens every child editor empty, and saving that emptiness would
 * erase the children the record actually holds. The row's version participates
 * in the cache key so a save that lands refetches rather than re-seating the
 * form from a stale record.
 */
export function useEditRecordReset<T extends FieldValues>(
  form: UseFormReturn<T>,
  { open, mode, queryKey, id, version, fetch }: EditRecordSource<T>,
): EditRecordState {
  const enabled = open && mode === "edit" && Boolean(id);

  const { data, isFetching, isError } = useQuery({
    queryKey: [queryKey, id, version],
    queryFn: () => fetch(id as string),
    enabled,
  });

  const { reset } = form;
  useEffect(() => {
    if (enabled && data) {
      reset(data);
    }
  }, [enabled, data, reset]);

  return {
    // A create panel has no record to wait for, and neither does a panel that
    // is closed. Only an edit waits, and only for its own record: a fetch that
    // failed is never seated, because the form is still holding the bare row.
    isSeated: !enabled || Boolean(data),
    isLoading: enabled && isFetching && !data,
    isError: enabled && isError,
  };
}
