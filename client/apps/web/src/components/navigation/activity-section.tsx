import { ResolvedUserAvatar } from "@/components/resolved-user-avatar";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@trenova/shared/components/ui/collapsible";
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
import { format, formatDistanceToNowStrict, fromUnixTime } from "date-fns";
import { ChevronRightIcon, Loader2 } from "lucide-react";
import { useEffect, useRef, useState } from "react";

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
          <div className="flex items-start gap-2 rounded-md px-2 py-1 transition-colors hover:bg-muted/50" />
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
          <span className="truncate text-xs text-muted-foreground">
            <span className="font-medium text-foreground">{firstName}</span> {headline}
          </span>
          <span className="truncate text-2xs text-muted-foreground/70">
            {entry.entityRef ? (
              <>
                <span className="font-medium text-muted-foreground">{entry.entityRef}</span>
                {" · "}
                {resource}
              </>
            ) : (
              resource
            )}
          </span>
        </span>
        <span className="shrink-0 pt-0.5 text-2xs text-muted-foreground/60 tabular-nums">
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
            {format(fromUnixTime(entry.timestamp), "MMM d, yyyy · h:mm a")}
          </span>
        </div>
      </TooltipContent>
    </Tooltip>
  );
}

function OnlineIndicator() {
  const { onlineUserIDs } = useOnlineUsers();

  if (onlineUserIDs.size === 0) {
    return null;
  }

  return (
    <span className="flex items-center gap-1.5 text-2xs text-muted-foreground normal-case">
      <span className="size-1.5 rounded-full bg-success" />
      {onlineUserIDs.size} online
    </span>
  );
}

export function ActivitySection() {
  const { data: preferences } = useSidebarPreferences();
  const [openOverride, setOpenOverride] = useState<boolean | null>(null);
  const open = openOverride ?? preferences?.activity.defaultOpen ?? true;
  const {
    data: entries,
    isLoading,
    isSuccess,
    hasNextPage,
    isFetchingNextPage,
    fetchNextPage,
  } = useRecentActivityInfinite(preferences?.activity.pageSize);
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
  }, [hasNextPage, isFetchingNextPage, fetchNextPage, open, isLoading]);

  if (isSuccess && entries.length === 0) {
    return null;
  }
  if (!isLoading && !isSuccess) {
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
                className="group flex h-6 items-center gap-1 rounded-md px-2 text-2xs font-semibold tracking-wider text-muted-foreground uppercase transition-colors select-none hover:text-foreground"
              >
                <span>Recent Activity</span>
                <ChevronRightIcon
                  className={cn("size-3 shrink-0 transition-transform", open && "rotate-90")}
                />
              </button>
            )}
          />
          <OnlineIndicator />
        </div>
        <CollapsibleContent>
          {isLoading ? (
            <div className="flex flex-col gap-0.5">
              {Array.from({ length: 3 }, (_, index) => (
                <Skeleton key={index} className="h-8 w-full rounded-md" />
              ))}
            </div>
          ) : (
            <ScrollArea viewportClassName="max-h-56" maskHeight={16} maskVariant="sidebar">
              <div className="relative flex w-full flex-col gap-0.5 pr-2.5 pl-2">
                {entries?.map((entry) => (
                  <ActivityRow key={entry.id} entry={entry} />
                ))}
                {isFetchingNextPage && (
                  <div className="flex items-center justify-center py-1.5">
                    <Loader2 className="size-3.5 animate-spin text-muted-foreground" />
                  </div>
                )}
                <div ref={observerTarget} aria-hidden className="h-px w-full" />
              </div>
            </ScrollArea>
          )}
        </CollapsibleContent>
      </div>
    </Collapsible>
  );
}
