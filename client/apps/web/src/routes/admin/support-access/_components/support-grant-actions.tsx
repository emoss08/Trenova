import { Link } from "@trenova/shared/components/link";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { usePermission } from "@/hooks/use-permission";
import { AUDIT_EXPORT_LIST_KEY, requestAuditExport } from "@/lib/graphql/audit-log-admin";
import { fetchSupportAccessActions, type SupportAccessGrant } from "@/lib/graphql/support-access";
import { supportAccess } from "@/lib/queries/support-access";
import { withoutRecordIds } from "@/lib/record-ids";
import { useInfiniteQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { toast } from "@trenova/shared/components/ui/toast";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTime } from "@trenova/shared/lib/date";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { SupportActionsSkeleton } from "./support-grant-skeleton";

const EXPORT_MARGIN_SECONDS = 60;

/** Writes every entry made under the grant to a file the reader downloads from the audit log. */
function ExportGrantActivity({ grant }: { grant: SupportAccessGrant }) {
  const t = useT();
  const queryClient = useQueryClient();
  const { allowed: canExport } = usePermission(Resource.AuditLog, Operation.Export);
  const { allowed: canIncludeSecurity } = usePermission(Resource.SecurityAuditLog, Operation.Read);
  const request = useApiMutation({
    scope: { id: `support-activity-export-${grant.id}` },
    resourceName: "Audit export",
    mutationFn: () => {
      const ended = grant.revokedAt ?? Math.min(grant.expiresAt, Math.floor(Date.now() / 1000));
      return requestAuditExport({
        format: "CSV",
        rangeFrom: grant.grantedAt - EXPORT_MARGIN_SECONDS,
        rangeTo: ended + EXPORT_MARGIN_SECONDS,
        includeSecurity: canIncludeSecurity,
        filters: { principalId: grant.id },
      });
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: [AUDIT_EXPORT_LIST_KEY] });
      toast.success(t("The export is being written"), {
        description: t("Download it from Exports on the audit log when it is ready."),
      });
    },
  });

  if (!canExport) {
    return null;
  }

  return (
    <Button
      type="button"
      size="sm"
      variant="secondary"
      className="self-end"
      isLoading={request.isPending}
      onClick={() => request.mutateAsync(undefined).catch(() => undefined)}
    >
      {t("Export what support did")}
    </Button>
  );
}

/**
 * Everything done under one grant, newest first, read from the audit log: every entry a support
 * session wrote is credited to its grant. Entries about kinds of record the reader cannot read
 * are left out, as they are on the audit log itself.
 */
export function SupportGrantActions({ grant }: { grant: SupportAccessGrant }) {
  const t = useT();
  const grantId = grant.id;
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
      <p className="py-2 text-sm text-foreground-subtle">
        {t("What support did could not be loaded. You may not be able to read the audit log.")}
      </p>
    );
  }

  const actions = query.data?.pages.flatMap((page) => page.items) ?? [];
  if (actions.length === 0) {
    return (
      <p className="py-2 text-sm text-foreground-subtle">
        {t("Nothing has been done under this access.")}
      </p>
    );
  }

  return (
    <div className="flex flex-col gap-1 py-2">
      <ExportGrantActivity grant={grant} />
      <ul className="flex flex-col divide-y divide-border-subtle">
        {actions.map((action) => (
          <li key={action.id} className="flex items-baseline gap-3 py-1.5 text-sm">
            <span className="w-40 shrink-0 text-foreground-subtle tabular-nums">
              {formatUnixDateTime(action.timestamp)}
            </span>
            <Link
              to="/admin/audit-logs"
              search={{ entry: action.id }}
              className="min-w-0 flex-1 truncate text-foreground hover:underline"
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
