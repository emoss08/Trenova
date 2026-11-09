import { useT } from "@trenova/shared/i18n/use-t";
import { Badge } from "@trenova/shared/components/ui/badge";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@trenova/shared/components/ui/collapsible";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { TextShimmer } from "@trenova/shared/components/ui/text-shimmer";
import { usePermission } from "@/hooks/use-permission";
import { formatToUserTimezone } from "@trenova/shared/lib/date";
import { fetchRecordActivity, type ActivityEntry } from "@/lib/graphql/record-activity";
import { audit } from "@/lib/queries/audit";
import { cn } from "@trenova/shared/lib/utils";
import {
  formatAuditFieldValue,
  formatFieldLabel,
  isRecordValue,
  normalizeAuditChanges,
  operationLabel,
  operationVariant,
  userInitials,
  type NormalizedAuditChange,
} from "@/routes/admin/audit-logs/_components/audit-log-formatters";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useInfiniteQuery } from "@tanstack/react-query";
import { formatDistanceToNow, fromUnixTime } from "date-fns";
import { ChevronRightIcon, LinkExternal01Icon } from "@trenova/shared/components/icons";
import { useEffect, useMemo, useRef, useState } from "react";
import { Link, useNavigate } from "react-router";
import { AuditEmpty } from "./audit-empty";

const PAGE_SIZE = 20;

function auditEntryUrl(entryId: string) {
  return `/admin/audit-logs?entry=${encodeURIComponent(entryId)}`;
}

export default function AuditTab({ resourceId }: { resourceId: string }) {
  const t = useT();

  const query = useInfiniteQuery({
    queryKey: [...audit.history(resourceId).queryKey],
    queryFn: async ({ pageParam, signal }) =>
      fetchRecordActivity(resourceId, { first: PAGE_SIZE, after: pageParam ?? null }, { signal }),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) =>
      lastPage.hasNextPage ? (lastPage.endCursor ?? undefined) : undefined,
  });

  const { allowed: canViewAuditLogs } = usePermission(Resource.AuditLog, Operation.Read);
  const { hasNextPage, isFetchingNextPage, fetchNextPage } = query;
  const observerTarget = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && hasNextPage && !isFetchingNextPage) {
          void fetchNextPage();
        }
      },
      { threshold: 0.1 },
    );

    const currentTarget = observerTarget.current;
    if (currentTarget) {
      observer.observe(currentTarget);
    }

    return () => {
      if (currentTarget) {
        observer.unobserve(currentTarget);
      }
    };
  }, [hasNextPage, isFetchingNextPage, fetchNextPage]);

  const allEntries = useMemo(
    () => query.data?.pages.flatMap((page) => page.rows) ?? [],
    [query.data?.pages],
  );

  if (query.isLoading) {
    return <AuditCardsSkeleton />;
  }

  if (query.isError) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
        <p className="text-sm">{t("Failed to load audit history.")}</p>
        <button
          type="button"
          onClick={() => void query.refetch()}
          className="mt-2 text-xs text-foreground underline"
        >
          {t("Retry")}
        </button>
      </div>
    );
  }

  if (allEntries.length === 0) {
    return (
      <AuditEmpty
        title={t("No activity yet")}
        description={t(
          "Every change to this record is written here as it happens: who made it, when, and what it changed.",
        )}
      />
    );
  }

  return (
    <div className="flex flex-col gap-2">
      {allEntries.map((entry) => (
        <AuditCard key={entry.id} entry={entry} canNavigate={canViewAuditLogs} />
      ))}
      {isFetchingNextPage && (
        <div className="flex items-center justify-center py-4">
          <TextShimmer className="font-mono text-sm" duration={1}>
            {t("Loading more...")}
          </TextShimmer>
        </div>
      )}
      <div ref={observerTarget} className="h-px" />
    </div>
  );
}

function AuditCard({ entry, canNavigate }: { entry: ActivityEntry; canNavigate: boolean }) {
  const t = useT();

  const [open, setOpen] = useState(false);
  const navigate = useNavigate();
  const date = fromUnixTime(entry.timestamp);
  const relativeTime = formatDistanceToNow(date, { addSuffix: true });
  const fullDate = formatToUserTimezone(entry.timestamp);
  const userName = entry.actorName || entry.user?.name || t("Unknown user");
  const operation = entry.operation?.toLowerCase() || "";

  const changes =
    operation === "update" && isRecordValue(entry.changes) && Object.keys(entry.changes).length > 0
      ? normalizeAuditChanges(entry.changes)
      : [];

  const hasDetails = changes.length > 0;

  const handleCardClick = () => {
    if (!hasDetails && canNavigate) {
      void navigate(auditEntryUrl(entry.id));
    }
  };

  return (
    <Collapsible open={open} onOpenChange={setOpen}>
      <div
        className={cn(
          "rounded-lg border border-border bg-card",
          !hasDetails && canNavigate && "cursor-pointer transition-colors hover:bg-muted/50",
        )}
        onClick={handleCardClick}
      >
        <CollapsibleTrigger
          className={cn(
            "flex w-full items-center gap-3 p-3 text-left",
            hasDetails && "cursor-pointer transition-colors hover:bg-muted/50",
            open && "border-b border-border",
          )}
          disabled={!hasDetails}
        >
          <span className="flex size-7 shrink-0 items-center justify-center rounded-full bg-muted text-xs font-medium text-muted-foreground">
            {userInitials(entry.user?.name)}
          </span>

          <div className="flex min-w-0 flex-1 flex-col gap-0.5">
            <div className="flex items-center gap-2">
              <span className="truncate text-sm font-medium">{userName}</span>
              <Badge variant={operationVariant(entry.operation)} className="shrink-0">
                {operationLabel(entry.operation)}
              </Badge>
            </div>
            <OperationSummary operation={operation} changeCount={changes.length} />
          </div>

          <div className="flex shrink-0 items-center gap-2">
            <span className="text-xs text-muted-foreground" title={fullDate}>
              {relativeTime}
            </span>
            {hasDetails && (
              <ChevronRightIcon
                className={cn(
                  "size-4 text-muted-foreground transition-transform duration-200",
                  open && "rotate-90",
                )}
              />
            )}
          </div>
        </CollapsibleTrigger>

        <CollapsibleContent>
          <div className="flex flex-col gap-px rounded-b-lg border-b border-border bg-muted p-2">
            {changes.map((change) => (
              <ChangeItem key={change.path} change={change} />
            ))}
          </div>
          {canNavigate && (
            <div className="justify-left flex px-2 py-1">
              <Link
                to={auditEntryUrl(entry.id)}
                className="inline-flex items-center gap-1 text-xs text-muted-foreground transition-colors hover:text-foreground"
              >
                {t("View full record")}
                <LinkExternal01Icon className="size-3" />
              </Link>
            </div>
          )}
        </CollapsibleContent>
      </div>
    </Collapsible>
  );
}

function OperationSummary({ operation, changeCount }: { operation: string; changeCount: number }) {
  const t = useT();

  if (operation === "create") {
    return <p className="text-xs text-muted-foreground">{t("Created this resource")}</p>;
  }

  if (operation === "delete") {
    return <p className="text-xs text-muted-foreground">{t("Deleted this resource")}</p>;
  }

  if (operation === "update" && changeCount > 0) {
    return (
      <p className="text-xs text-muted-foreground">
        {t("{0, plural, one {# field} other {# fields}} changed", changeCount)}
      </p>
    );
  }

  return (
    <p className="text-xs text-muted-foreground">
      {t("{0} this resource", operationLabel(operation))}
    </p>
  );
}

function ChangeItem({ change }: { change: NormalizedAuditChange }) {
  const fromFormatted = formatAuditFieldValue(change.from, change.path);
  const toFormatted = formatAuditFieldValue(change.to, change.path);
  const fromDisplay = fromFormatted.value;
  const toDisplay = toFormatted.value;

  return (
    <div className="flex items-start gap-3 rounded-md px-3 py-2">
      <span className="shrink-0 pt-px text-xs font-medium text-foreground">
        {formatFieldLabel(change.path)}
      </span>
      <div className="flex min-w-0 flex-1 items-center justify-end gap-1.5 text-xs">
        {change.type !== "added" && (
          <span
            className="max-w-[140px] truncate rounded-md bg-danger-subtle px-1.5 py-0.5 text-danger-foreground"
            title={fromDisplay}
          >
            {fromDisplay}
          </span>
        )}
        {change.type === "changed" && <span className="text-muted-foreground">&rarr;</span>}
        {change.type !== "removed" && (
          <span
            className="max-w-[140px] truncate rounded-md bg-success-subtle px-1.5 py-0.5 text-success-foreground"
            title={toDisplay}
          >
            {toDisplay}
          </span>
        )}
      </div>
    </div>
  );
}

function AuditCardsSkeleton() {
  return (
    <div className="flex flex-col gap-2">
      {Array.from({ length: 4 }).map((_, i) => (
        <div key={i} className="rounded-lg border border-border bg-card p-3">
          <div className="flex items-center gap-3">
            <Skeleton className="size-7 rounded-full" />
            <div className="flex flex-1 flex-col gap-1.5">
              <div className="flex items-center gap-2">
                <Skeleton className="h-4 w-24" />
                <Skeleton className="h-5 w-14 rounded-badge" />
              </div>
              <Skeleton className="h-3 w-32" />
            </div>
            <Skeleton className="h-3 w-16" />
          </div>
        </div>
      ))}
    </div>
  );
}
