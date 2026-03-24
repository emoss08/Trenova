import { Badge } from "@/components/ui/badge";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { Skeleton } from "@/components/ui/skeleton";
import { TextShimmer } from "@/components/ui/text-shimmer";
import { usePermission } from "@/hooks/use-permission";
import { formatToUserTimezone } from "@/lib/date";
import { queries } from "@/lib/queries";
import { cn } from "@/lib/utils";
import {
  formatAuditValueWithDates,
  formatFieldLabel,
  normalizeAuditChanges,
  operationLabel,
  operationVariant,
  userInitials,
  type NormalizedAuditChange,
} from "@/routes/admin/audit-logs/_components/audit-log-formatters";
import { apiService } from "@/services/api";
import type { AuditEntry } from "@/types/audit-entry";
import { Operation, Resource } from "@/types/permission";
import { useInfiniteQuery } from "@tanstack/react-query";
import { formatDistanceToNow, fromUnixTime } from "date-fns";
import { ChevronRight, ExternalLinkIcon } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { Link, useNavigate } from "react-router";
import { AuditAlert } from "./audit-alert";

const PAGE_SIZE = 20;

function auditEntryUrl(entryId: string) {
  return `/admin/audit-logs?panelType=edit&panelEntityId=${entryId}`;
}

export default function AuditTab({ resourceId }: { resourceId: string }) {
  const query = useInfiniteQuery({
    queryKey: [...queries.audit.history(resourceId).queryKey],
    queryFn: async ({ pageParam }) => {
      return await apiService.auditService.listByResourceId(resourceId, {
        limit: PAGE_SIZE,
        offset: pageParam,
      });
    },
    initialPageParam: 0,
    getNextPageParam: (lastPage, _, lastPageParam) => {
      if (lastPage.next || lastPage.results.length === PAGE_SIZE) {
        return lastPageParam + PAGE_SIZE;
      }
      return undefined;
    },
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
    () => query.data?.pages.flatMap((page) => page.results) ?? [],
    [query.data?.pages],
  );

  if (query.isLoading) {
    return <AuditCardsSkeleton />;
  }

  if (query.isError) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
        <p className="text-sm">Failed to load audit history.</p>
        <button
          type="button"
          onClick={() => void query.refetch()}
          className="mt-2 text-xs text-foreground underline"
        >
          Retry
        </button>
      </div>
    );
  }

  if (allEntries.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
        <p className="text-sm">No audit history for this resource.</p>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-2">
      <AuditAlert />
      {allEntries.map((entry) => (
        <AuditCard key={entry.id} entry={entry} canNavigate={canViewAuditLogs} />
      ))}
      {isFetchingNextPage && (
        <div className="flex items-center justify-center py-4">
          <TextShimmer className="font-mono text-sm" duration={1}>
            Loading more...
          </TextShimmer>
        </div>
      )}
      <div ref={observerTarget} className="h-px" />
    </div>
  );
}

function AuditCard({ entry, canNavigate }: { entry: AuditEntry; canNavigate: boolean }) {
  const [open, setOpen] = useState(false);
  const navigate = useNavigate();
  const date = fromUnixTime(entry.timestamp);
  const relativeTime = formatDistanceToNow(date, { addSuffix: true });
  const fullDate = formatToUserTimezone(entry.timestamp);
  const userName = entry.user?.name || entry.user?.emailAddress || "Unknown user";
  const operation = entry.operation?.toLowerCase() || "";

  const changes =
    operation === "update" && entry.changes && Object.keys(entry.changes).length > 0
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
              <ChevronRight
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
                View full record
                <ExternalLinkIcon className="size-3" />
              </Link>
            </div>
          )}
        </CollapsibleContent>
      </div>
    </Collapsible>
  );
}

function OperationSummary({ operation, changeCount }: { operation: string; changeCount: number }) {
  if (operation === "create") {
    return <p className="text-xs text-muted-foreground">Created this resource</p>;
  }

  if (operation === "delete") {
    return <p className="text-xs text-muted-foreground">Deleted this resource</p>;
  }

  if (operation === "update" && changeCount > 0) {
    return (
      <p className="text-xs text-muted-foreground">
        {changeCount} field{changeCount !== 1 ? "s" : ""} changed
      </p>
    );
  }

  return <p className="text-xs text-muted-foreground">{operationLabel(operation)} this resource</p>;
}

function ChangeItem({ change }: { change: NormalizedAuditChange }) {
  const fromFormatted = formatAuditValueWithDates(change.from, change.path);
  const toFormatted = formatAuditValueWithDates(change.to, change.path);
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
            className="max-w-[140px] truncate rounded bg-red-500/10 px-1.5 py-0.5 text-red-600 dark:text-red-400"
            title={fromDisplay}
          >
            {fromDisplay}
          </span>
        )}
        {change.type === "changed" && <span className="text-muted-foreground">&rarr;</span>}
        {change.type !== "removed" && (
          <span
            className="max-w-[140px] truncate rounded bg-green-500/10 px-1.5 py-0.5 text-green-600 dark:text-green-400"
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
                <Skeleton className="h-5 w-14 rounded-full" />
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
