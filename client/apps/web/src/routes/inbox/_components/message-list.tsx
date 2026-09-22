import type { InboundMessageRow } from "@/lib/graphql/inbox";
import { Button } from "@trenova/shared/components/ui/button";
import { EmptySheet, GhostLine } from "@trenova/shared/components/ui/empty-sheet";
import { Input } from "@trenova/shared/components/ui/input";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium, formatUnixWeekday, getStartOfDay } from "@trenova/shared/lib/date";
import { SearchIcon, XIcon } from "lucide-react";
import { useEffect, useMemo, useRef, type RefObject } from "react";
import { groupByDay, type DayGroup } from "./day-groups";
import { InboxMessageRow } from "./message-row";

function userStartOfDay(seconds: number): number {
  return getStartOfDay(new Date(seconds * 1000));
}

function dayHeading(t: (value: string) => string, group: DayGroup<InboundMessageRow>): string {
  switch (group.bucket) {
    case "today":
      return t("Today");
    case "yesterday":
      return t("Yesterday");
    case "week":
      return formatUnixWeekday(group.day);
    case "earlier":
      return formatUnixDateMedium(group.day);
  }
}

export type MessageListState = {
  messages: InboundMessageRow[];
  isLoading: boolean;
  isError: boolean;
  hasNextPage: boolean;
  isFetchingNextPage: boolean;
  fetchNextPage: () => void;
  retry: () => void;
};

/**
 * The column of mail in the open folder, newest arrival first and grouped by
 * the day it came in.
 *
 * The next page loads as the bottom comes into view, with a button for the
 * same thing when it does not — a list that stops without saying why looks
 * like the end of the mail.
 */
export function MessageList({
  title,
  list,
  openId,
  now,
  search,
  searchRef,
  onSearchChange,
  onOpen,
  empty,
}: {
  title: string;
  list: MessageListState;
  openId: string | null;
  now: number;
  search: string;
  searchRef: RefObject<HTMLInputElement | null>;
  onSearchChange: (value: string) => void;
  onOpen: (id: string) => void;
  empty: { title: string; description: string };
}) {
  const t = useT();
  const groups = useMemo(
    () => groupByDay(list.messages, now, userStartOfDay),
    [list.messages, now],
  );
  const rowRefs = useRef(new Map<string, HTMLButtonElement>());
  const sentinelRef = useRef<HTMLDivElement>(null);
  const { hasNextPage, isFetchingNextPage, fetchNextPage } = list;

  useEffect(() => {
    if (openId === null) {
      return;
    }
    rowRefs.current.get(openId)?.scrollIntoView({ block: "nearest" });
  }, [openId]);

  useEffect(() => {
    const sentinel = sentinelRef.current;
    if (sentinel === null || !hasNextPage) {
      return;
    }
    const observer = new IntersectionObserver((entries) => {
      if (entries.some((entry) => entry.isIntersecting) && !isFetchingNextPage) {
        fetchNextPage();
      }
    });
    observer.observe(sentinel);
    return () => observer.disconnect();
  }, [hasNextPage, isFetchingNextPage, fetchNextPage]);

  return (
    <section aria-label={title} className="flex min-h-0 flex-1 flex-col">
      <div className="border-border flex flex-col gap-2 border-b px-4 py-3">
        <div className="flex items-baseline justify-between gap-2">
          <h2 className="truncate text-sm font-semibold">{title}</h2>
          {list.messages.length > 0 && (
            <span className="text-foreground-subtle text-xs tabular-nums">
              {list.hasNextPage
                ? t("{0, plural, one {#+ message} other {#+ messages}}", list.messages.length)
                : t("{0, plural, one {# message} other {# messages}}", list.messages.length)}
            </span>
          )}
        </div>
        <Input
          ref={searchRef}
          value={search}
          onChange={(event) => onSearchChange(event.target.value)}
          placeholder={t("Search sender or subject")}
          aria-label={t("Search sender or subject")}
          leftElement={<SearchIcon className="text-foreground-subtle size-3.5" />}
          rightElement={
            search === "" ? (
              <Kbd className="mr-1">/</Kbd>
            ) : (
              <Button
                size="icon-xs"
                variant="ghost"
                aria-label={t("Clear the search")}
                onClick={() => onSearchChange("")}
              >
                <XIcon className="size-3.5" />
              </Button>
            )
          }
        />
      </div>

      <ScrollArea className="min-h-0 flex-1">
        {list.isLoading ? (
          <div className="flex flex-col gap-3 p-4" aria-busy="true">
            {[0, 1, 2, 3, 4].map((index) => (
              <div key={index} className="flex gap-3">
                <Skeleton className="size-8 shrink-0 rounded-md" />
                <div className="flex flex-1 flex-col gap-1.5">
                  <Skeleton className="h-3.5 w-1/2" />
                  <Skeleton className="h-3.5 w-4/5" />
                  <Skeleton className="h-3 w-3/5" />
                </div>
              </div>
            ))}
          </div>
        ) : list.isError ? (
          <EmptySheet
            className="my-10"
            title={t("The mail could not be loaded")}
            description={t("Something went wrong reading this folder. Nothing was changed.")}
            sketch={<GhostLine className="w-1/2" />}
            action={
              <Button size="sm" variant="outline" onClick={list.retry}>
                {t("Try again")}
              </Button>
            }
          />
        ) : list.messages.length === 0 ? (
          <EmptySheet
            className="my-10"
            title={search.trim() === "" ? empty.title : t("Nothing matches that search")}
            description={
              search.trim() === ""
                ? empty.description
                : t("Search looks at the sender's name and address, and the subject.")
            }
            sketch={
              <div className="flex flex-col gap-3">
                <GhostLine className="w-2/3" />
                <GhostLine className="w-1/2" />
                <GhostLine className="w-3/5" />
              </div>
            }
          />
        ) : (
          <div>
            {groups.map((group) => (
              <div key={group.day}>
                <h3 className="bg-background text-foreground-subtle border-border-subtle sticky top-0 z-10 border-b px-5 py-1.5 text-xs font-medium">
                  {dayHeading(t, group)}
                </h3>
                <ul>
                  {group.items.map((message) => (
                    <InboxMessageRow
                      key={message.id}
                      ref={(node) => {
                        if (node === null) {
                          rowRefs.current.delete(message.id);
                        } else {
                          rowRefs.current.set(message.id, node);
                        }
                      }}
                      message={message}
                      active={message.id === openId}
                      now={now}
                      onOpen={onOpen}
                    />
                  ))}
                </ul>
              </div>
            ))}

            <div ref={sentinelRef} className="flex justify-center p-4">
              {list.hasNextPage ? (
                <Button
                  size="sm"
                  variant="ghost"
                  disabled={list.isFetchingNextPage}
                  onClick={list.fetchNextPage}
                >
                  {list.isFetchingNextPage ? t("Loading more…") : t("Load more")}
                </Button>
              ) : (
                <span className="text-foreground-subtle text-xs">{t("That is all of it")}</span>
              )}
            </div>
          </div>
        )}
      </ScrollArea>
    </section>
  );
}
