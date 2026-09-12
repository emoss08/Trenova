import { useT } from "@trenova/shared/i18n/use-t";
import { ResolvedUserAvatar } from "@/components/resolved-user-avatar";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@trenova/shared/components/ui/collapsible";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useRecentActivityInfinite, type RecentActivityEntry } from "@/hooks/use-attention";
import { useOnlineUsers } from "@/hooks/use-online-users";
import { useSidebarPreferences } from "@/hooks/use-sidebar-preferences";
import { cn } from "@trenova/shared/lib/utils";
import {
  operationLabel,
  resourceLabel,
} from "@/routes/admin/audit-logs/_components/audit-log-formatters";
import { formatDistanceToNowStrict, fromUnixTime } from "date-fns";
import { ChevronRightIcon, Loader2 } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { formatUnixDateMedium, formatUnixTime } from "@trenova/shared/lib/date";

const PAST_TENSE_OPERATIONS: Record<string, string> = {
  create: "created",
  update: "updated",
  delete: "deleted",
  approve: "approved",
  reject: "rejected",
  assign: "assigned",
  unassign: "unassigned",
  archive: "archived",
  restore: "restored",
  submit: "submitted",
  cancel: "canceled",
  duplicate: "duplicated",
  close: "closed",
  lock: "locked",
  unlock: "unlocked",
  activate: "activated",
  reopen: "reopened",
  export: "exported",
  import: "imported",
};

function operationVerb(operation: string): string {
  const normalized = operation.toLowerCase();
  return PAST_TENSE_OPERATIONS[normalized] ?? operationLabel(operation).toLowerCase();
}

function activityHeadline(entry: RecentActivityEntry): string {
  if (entry.comment) {
    return entry.comment;
  }
  return `${operationVerb(entry.operation)} ${resourceLabel(entry.resource)}`;
}

function ActivityRow({ entry }: { entry: RecentActivityEntry }) {
  const actorName = entry.user?.name ?? entry.user?.username ?? "System";
  const firstName = actorName.split(" ")[0];
  const resource = resourceLabel(entry.resource);
  const headline = activityHeadline(entry);

  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <div className="hover:bg-muted/50 flex items-start gap-2 rounded-md px-2 py-1 transition-colors" />
        }
      >
        <ResolvedUserAvatar
          userId={entry.user?.id}
          name={actorName}
          profilePicUrl={entry.user?.profilePicUrl ?? undefined}
          thumbnailUrl={entry.user?.thumbnailUrl ?? undefined}
          className="mt-0.5 size-4"
          fallbackClassName="bg-muted text-[7px] font-medium text-muted-foreground"
        />
        <span className="grid min-w-0 flex-1 leading-snug">
          <span className="text-muted-foreground truncate text-xs">
            <span className="text-foreground font-medium">{firstName}</span> {headline}
          </span>
          <span className="text-2xs text-muted-foreground/70 truncate">
            {entry.entityRef ? (
              <>
                <span className="text-muted-foreground font-medium">{entry.entityRef}</span>
                {" · "}
                {resource}
              </>
            ) : (
              resource
            )}
          </span>
        </span>
        <span className="text-2xs text-muted-foreground/60 shrink-0 pt-0.5 tabular-nums">
          {formatDistanceToNowStrict(fromUnixTime(entry.timestamp), { addSuffix: false })}
        </span>
      </TooltipTrigger>
      <TooltipContent side="right" sideOffset={14} className="max-w-64">
        <div className="flex flex-col gap-1 py-0.5">
          <span className="text-xs leading-snug">
            <span className="font-semibold">{actorName}</span> {headline}
          </span>
          <span className="text-2xs text-background/70">
            {entry.entityRef ? `${entry.entityRef} · ` : ""}
            {resource} · {operationLabel(entry.operation)}
          </span>
          <span className="text-2xs text-background/70 tabular-nums">
            {`${formatUnixDateMedium(entry.timestamp)} · ${formatUnixTime(entry.timestamp)}`}
          </span>
        </div>
      </TooltipContent>
    </Tooltip>
  );
}

export function ActivityOnlineIndicator() {
  const t = useT();

  const { onlineUserIDs } = useOnlineUsers();

  if (onlineUserIDs.size === 0) {
    return null;
  }

  return (
    <span className="text-2xs text-muted-foreground flex items-center gap-1.5 normal-case">
      <span className="bg-success size-1.5 rounded-full" />
      {t("{0} online", onlineUserIDs.size)}
    </span>
  );
}

function ActivityFeedList({
  entries,
  isLoading,
  isFetchingNextPage,
  observerTarget,
  maxHeightClassName,
}: {
  entries: RecentActivityEntry[] | undefined;
  isLoading: boolean;
  isFetchingNextPage: boolean;
  observerTarget: React.RefObject<HTMLDivElement | null>;
  maxHeightClassName: string;
}) {
  if (isLoading) {
    return (
      <div className="flex flex-col gap-0.5">
        {Array.from({ length: 3 }, (_, index) => (
          <Skeleton key={index} className="h-8 w-full rounded-md" />
        ))}
      </div>
    );
  }

  return (
    <ScrollArea viewportClassName={maxHeightClassName} maskHeight={16} maskVariant="sidebar">
      <div className="relative flex w-full flex-col gap-0.5 pr-2.5 pl-2">
        {entries?.map((entry) => (
          <ActivityRow key={entry.id} entry={entry} />
        ))}
        {isFetchingNextPage && (
          <div className="flex items-center justify-center py-1.5">
            <Loader2 className="text-muted-foreground size-3.5 animate-spin" />
          </div>
        )}
        <div ref={observerTarget} aria-hidden className="h-px w-full" />
      </div>
    </ScrollArea>
  );
}

function useActivityFeed(pageSize: number | undefined, active: boolean) {
  const query = useRecentActivityInfinite(pageSize);
  const { hasNextPage, isFetchingNextPage, fetchNextPage, isLoading } = query;
  const observerTarget = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    const currentTarget = observerTarget.current;
    if (!currentTarget) return;

    const observer = new IntersectionObserver(
      (observedEntries) => {
        if (observedEntries[0].isIntersecting && hasNextPage && !isFetchingNextPage) {
          void fetchNextPage();
        }
      },
      { threshold: 0.1 },
    );
    observer.observe(currentTarget);

    return () => {
      observer.unobserve(currentTarget);
    };
  }, [hasNextPage, isFetchingNextPage, fetchNextPage, active, isLoading]);

  const isEmpty = query.isSuccess && query.data.length === 0;
  const isUnavailable = !query.isLoading && !query.isSuccess;

  return { ...query, observerTarget, isEmpty, isUnavailable };
}

/**
 * The feed on its own, for surfaces that draw their own heading. With
 * `emptyState="message"` it says so when there is nothing to show; with
 * `"hidden"` it disappears, heading included, so a sidebar does not carry a
 * label over nothing.
 */
export function ActivityFeed({
  maxHeightClassName = "max-h-80",
  emptyState = "message",
  heading,
}: {
  maxHeightClassName?: string;
  emptyState?: "message" | "hidden";
  heading?: React.ReactNode;
}) {
  const t = useT();

  const { data: preferences } = useSidebarPreferences();
  const feed = useActivityFeed(preferences?.activity.pageSize, true);

  if (feed.isEmpty || feed.isUnavailable) {
    if (emptyState === "hidden") {
      return null;
    }
    return (
      <p className="text-muted-foreground px-2 py-6 text-center text-xs">
        {feed.isUnavailable
          ? t("Activity is not available right now.")
          : t("Nothing has happened yet.")}
      </p>
    );
  }

  return (
    <>
      {heading}
      <ActivityFeedList
        entries={feed.data}
        isLoading={feed.isLoading}
        isFetchingNextPage={feed.isFetchingNextPage}
        observerTarget={feed.observerTarget}
        maxHeightClassName={maxHeightClassName}
      />
    </>
  );
}

export function ActivitySection() {
  const t = useT();

  const { data: preferences } = useSidebarPreferences();
  const [openOverride, setOpenOverride] = useState<boolean | null>(null);
  const open = openOverride ?? preferences?.activity.defaultOpen ?? true;
  const feed = useActivityFeed(preferences?.activity.pageSize, open);

  if (feed.isEmpty || feed.isUnavailable) {
    return null;
  }

  return (
    <Collapsible open={open} onOpenChange={setOpenOverride}>
      <div className="flex flex-col gap-0.5">
        <div className="flex h-6 items-center justify-between pr-2">
          <CollapsibleTrigger
            render={(props) => (
              <button
                {...props}
                className="group text-2xs text-muted-foreground hover:text-foreground flex h-6 items-center gap-1 rounded-md px-2 font-semibold tracking-wider uppercase transition-colors select-none"
              >
                <span>{t("Recent Activity")}</span>
                <ChevronRightIcon
                  className={cn("size-3 shrink-0 transition-transform", open && "rotate-90")}
                />
              </button>
            )}
          />
          <ActivityOnlineIndicator />
        </div>
        <CollapsibleContent>
          <ActivityFeedList
            entries={feed.data}
            isLoading={feed.isLoading}
            isFetchingNextPage={feed.isFetchingNextPage}
            observerTarget={feed.observerTarget}
            maxHeightClassName="max-h-56"
          />
        </CollapsibleContent>
      </div>
    </Collapsible>
  );
}
