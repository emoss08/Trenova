import { toast } from "sonner";

export async function runBulkAction<T>(
  rows: T[],
  run: (row: T) => Promise<unknown>,
  labels: { noun: string; verb: string },
): Promise<void> {
  const results = await Promise.allSettled(rows.map((row) => run(row)));
  const failed = results.filter(
    (result): result is PromiseRejectedResult => result.status === "rejected",
  );
  const succeeded = results.length - failed.length;

  if (failed.length === 0) {
    toast.success(`${succeeded} ${labels.noun}${succeeded === 1 ? "" : "s"} ${labels.verb}`);
    return;
  }

  const reason = failed[0].reason as Error | undefined;
  toast.warning(
    `${succeeded} ${labels.verb}, ${failed.length} failed${reason?.message ? ` — ${reason.message}` : ""}`,
  );
}
