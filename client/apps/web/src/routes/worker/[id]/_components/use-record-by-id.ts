import { useQuery } from "@tanstack/react-query";

/** A record a pane is open on, as the pane needs it. */
export type RecordById<Row> = {
  /** The record, from the rows already read or from its own read; null while unknown or gone. */
  record: Row | null;
  /** Still finding it: the rows are loading, or it was not among them and its own read is out. */
  isPending: boolean;
  /** Its own read failed (a deleted record, or a network fault). */
  isError: boolean;
  error: unknown;
  refetch: () => void;
};

/**
 * The record a pane is open on: the row the list already read when it is there, and otherwise
 * (a deep link to a record past the loaded pages or the server's cap) one read of that record
 * by id. The read waits until the list has settled, so a record that arrives with the list is
 * never read twice; it is keyed under `[root, workerId]`, so the area's `useWorkerInvalidation`
 * refreshes it with the list.
 *
 *   const event = useRecordById({
 *     root: WORKER_SAFETY_EVENTS_KEY,
 *     workerId,
 *     id: open.safetyEvent,
 *     rows: events,
 *     rowsSettled: !eventsQuery.isPending,
 *     fetchRecord: fetchWorkerSafetyEvent,
 *   });
 */
export function useRecordById<Row extends { id?: string | null }>({
  root,
  workerId,
  id,
  rows,
  rowsSettled,
  fetchRecord,
  enabled = true,
}: {
  /** The area's key root, the one its `WORKER_WRITE_REACH` row names. */
  root: string;
  workerId: string;
  /** The record the address names, or null when none is. */
  id: string | null | undefined;
  /** The rows already read (every loaded page, or the capped list). */
  rows: readonly Row[];
  /** The rows have finished their first read (loaded or failed). */
  rowsSettled: boolean;
  fetchRecord: (id: string, options: { signal: AbortSignal }) => Promise<Row>;
  /** False when the viewer may not read the area. */
  enabled?: boolean;
}): RecordById<Row> {
  const wanted = id ?? null;
  const loaded = wanted === null ? null : (rows.find((row) => row.id === wanted) ?? null);
  const own = useQuery({
    queryKey: [root, workerId, "by-id", wanted],
    queryFn: ({ signal }) => fetchRecord(wanted ?? "", { signal }),
    enabled: enabled && wanted !== null && loaded === null && rowsSettled,
  });

  if (wanted === null || !enabled) {
    return { record: null, isPending: false, isError: false, error: null, refetch: () => {} };
  }
  if (loaded) {
    return { record: loaded, isPending: false, isError: false, error: null, refetch: () => {} };
  }
  return {
    record: own.data ?? null,
    isPending: !rowsSettled || own.isPending,
    isError: own.isError,
    error: own.error,
    refetch: () => void own.refetch(),
  };
}
