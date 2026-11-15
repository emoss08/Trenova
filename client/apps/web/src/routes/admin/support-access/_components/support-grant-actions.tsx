import { Link } from "@trenova/shared/components/link";
import { fetchSupportAccessActions } from "@/lib/graphql/support-access";
import { supportAccess } from "@/lib/queries/support-access";
import { withoutRecordIds } from "@/lib/record-ids";
import { useInfiniteQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTime } from "@trenova/shared/lib/date";
import { SupportActionsSkeleton } from "./support-grant-skeleton";

/**
 * Everything done under one grant, newest first, read from the audit log: every entry a support
 * session wrote is credited to its grant. Entries about kinds of record the reader cannot read
 * are left out, as they are on the audit log itself.
 */
export function SupportGrantActions({ grantId }: { grantId: string }) {
  const t = useT();
  const query = useInfiniteQuery({
    queryKey: supportAccess.actions(grantId).queryKey,
    queryFn: ({ pageParam, signal }) => fetchSupportAccessActions(grantId, pageParam, { signal }),
    initialPageParam: null as string | null,
    getNextPageParam: (page) => page.nextCursor,
  });

  if (query.isLoading) {
    return <SupportActionsSkeleton />;
  }
  if (query.isError) {
    return (
      <p className="text-foreground-subtle py-2 text-sm">
        {t("What support did could not be loaded. You may not be able to read the audit log.")}
      </p>
    );
  }

  const actions = query.data?.pages.flatMap((page) => page.items) ?? [];
  if (actions.length === 0) {
    return (
      <p className="text-foreground-subtle py-2 text-sm">
        {t("Nothing has been done under this access.")}
      </p>
    );
  }

  return (
    <div className="flex flex-col gap-1 py-2">
      <ul className="divide-border-subtle flex flex-col divide-y">
        {actions.map((action) => (
          <li key={action.id} className="flex items-baseline gap-3 py-1.5 text-sm">
            <span className="text-foreground-subtle w-40 shrink-0 tabular-nums">
              {formatUnixDateTime(action.timestamp)}
            </span>
            <Link
              to="/admin/audit-logs"
              search={{ entry: action.id }}
              className="text-foreground min-w-0 flex-1 truncate hover:underline"
            >
              {withoutRecordIds(action.comment ?? action.entityRef ?? "") ||
                t("Open the audit entry")}
            </Link>
          </li>
        ))}
      </ul>
      {query.hasNextPage ? (
        <Button
          type="button"
          size="sm"
          variant="ghost"
          className="self-start"
          isLoading={query.isFetchingNextPage}
          onClick={() => query.fetchNextPage()}
        >
          {t("Show more")}
        </Button>
      ) : null}
    </div>
  );
}
