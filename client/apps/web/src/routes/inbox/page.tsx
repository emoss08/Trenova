import { PageLayout } from "@/components/navigation/sidebar-layout";
import { queries } from "@/lib/queries";
import { inboxMessagesQuery } from "@/lib/queries/inbox";
import type { RoutePrefetch } from "@/lib/route-prefetch";
import { useInfiniteQuery, useQuery } from "@tanstack/react-query";
import { useDebounce } from "@trenova/shared/hooks/use-debounce";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { MailOpenIcon } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useSearchParams } from "react-router";
import { classificationLabel } from "./_components/classification";
import { FolderRail, laneLabel } from "./_components/folder-rail";
import {
  folderFilter,
  folderParams,
  isSameFolder,
  parseFolder,
  type InboxFolder,
} from "./_components/folders";
import { LANE_ORDER } from "./_components/lanes";
import { MessageList } from "./_components/message-list";
import { ReadingPane } from "./_components/reading-pane";
import { nextAfterLeaving, reduceTriageKey, useTriageKeys } from "./_components/triage-keys";
import { useInboxActions } from "./_components/use-inbox-actions";

const nowInSeconds = () => Math.floor(Date.now() / 1000);
const SEARCH_DEBOUNCE_MS = 250;
const CLOCK_TICK_MS = 60_000;

export const prefetch: RoutePrefetch = ({ request }) => {
  const params = new URL(request.url).searchParams;
  const folder = parseFolder(params);

  return [queries.inbox.counts(), inboxMessagesQuery(folderFilter(folder, params.get("q") ?? ""))];
};

function useNow(): number {
  const [now, setNow] = useState(nowInSeconds);
  useEffect(() => {
    const timer = window.setInterval(() => setNow(nowInSeconds()), CLOCK_TICK_MS);
    return () => window.clearInterval(timer);
  }, []);

  return now;
}

/**
 * The inbox: a mail client for what arrived on a monitored address, with the
 * desk's reading of every message beside it.
 *
 * Three panes, as mail has always been read — folders, the list, the open
 * message — and the address holds all of it: the folder, the search and the
 * open message. A watchtower item links straight to a message, the back
 * button walks back through what was opened, and a link pasted into a chat
 * opens what the sender was looking at.
 */
export function InboxPage() {
  const t = useT();
  const now = useNow();
  const [searchParams, setSearchParams] = useSearchParams();
  const searchRef = useRef<HTMLInputElement>(null);
  const [linkOpen, setLinkOpen] = useState(false);

  const folder = useMemo(() => parseFolder(searchParams), [searchParams]);
  const openId = searchParams.get("message");
  const [search, setSearch] = useState(() => searchParams.get("q") ?? "");
  const debouncedSearch = useDebounce(search, SEARCH_DEBOUNCE_MS);

  const updateParams = useCallback(
    (update: (params: URLSearchParams) => void, replace = false) => {
      setSearchParams(
        (current) => {
          const next = new URLSearchParams(current);
          update(next);
          return next;
        },
        { replace },
      );
    },
    [setSearchParams],
  );

  useEffect(() => {
    updateParams((params) => {
      if (debouncedSearch.trim() === "") {
        params.delete("q");
      } else {
        params.set("q", debouncedSearch);
      }
    }, true);
  }, [debouncedSearch, updateParams]);

  const filter = useMemo(() => folderFilter(folder, debouncedSearch), [folder, debouncedSearch]);
  const messagesQuery = useInfiniteQuery(inboxMessagesQuery(filter));
  const countsQuery = useQuery(queries.inbox.counts());
  const mailboxesQuery = useQuery({ ...queries.inbox.mailboxes(), retry: false });

  const messages = useMemo(
    () => messagesQuery.data?.pages.flatMap((page) => page.messages) ?? [],
    [messagesQuery.data],
  );
  const ids = useMemo(() => messages.map((message) => message.id), [messages]);

  const openMessage = useCallback(
    (id: string | null) => {
      setLinkOpen(false);
      updateParams((params) => {
        if (id === null) {
          params.delete("message");
        } else {
          params.set("message", id);
        }
      });
    },
    [updateParams],
  );

  const selectFolder = useCallback(
    (next: InboxFolder) => {
      if (isSameFolder(next, folder)) {
        return;
      }
      setSearchParams((current) => folderParams(current, next));
    },
    [folder, setSearchParams],
  );

  // A decision takes a waiting message out of the waiting lane. The reader
  // goes on to the next one, as a triage run does, instead of being left on a
  // message the list beside it no longer shows.
  const onReviewed = useCallback(
    (id: string) => {
      if (folder.kind === "lane" && folder.lane === "waiting") {
        openMessage(nextAfterLeaving(ids, id));
      }
    },
    [folder, ids, openMessage],
  );

  const actions = useInboxActions({ onReviewed });
  const { review, ask } = actions;
  const detail = useQuery({ ...queries.inbox.message(openId ?? ""), enabled: openId !== null });

  const onKey = useCallback(
    (key: string) => {
      const next = reduceTriageKey({ openId, ask: null }, key, ids);
      if (next.openId !== openId) {
        openMessage(next.openId);
      }
      if (next.ask === null) {
        return;
      }
      const message = detail.data;
      switch (next.ask.kind) {
        case "handle":
        case "ignore":
          if (message?.needsReview) {
            review({
              id: next.ask.id,
              status: next.ask.kind === "handle" ? "Actioned" : "Ignored",
              note: "",
            });
          }
          break;
        case "link":
          setLinkOpen(true);
          break;
        case "ask":
          if (message !== undefined) {
            ask(message);
          }
          break;
      }
    },
    [openId, ids, openMessage, detail.data, review, ask],
  );

  useTriageKeys({
    enabled: !linkOpen,
    onKey,
    onSearch: () => searchRef.current?.focus(),
  });

  const counts = countsQuery.data;
  const mailboxes = mailboxesQuery.data ?? [];
  const title = folderTitle(t, folder, mailboxes);

  return (
    <PageLayout
      fill
      className="p-0"
      pageHeaderProps={{
        title: t("Inbox"),
        description: t(
          "Mail that arrived on a monitored address, what the desk made of each message, and what is still waiting on a person",
        ),
      }}
    >
      <div className="flex min-h-0 flex-1">
        <aside className="border-border bg-sunken hidden w-60 shrink-0 flex-col border-r md:flex">
          <FolderRail
            folder={folder}
            counts={counts}
            mailboxes={mailboxes}
            onSelect={selectFolder}
          />
        </aside>

        <div
          className={cn(
            "border-border min-w-0 flex-col lg:flex lg:w-96 lg:shrink-0 lg:border-r xl:w-[28rem]",
            openId === null ? "flex flex-1 lg:flex-none" : "hidden",
          )}
        >
          <div className="border-border flex gap-1 overflow-x-auto border-b px-3 py-2 md:hidden">
            {LANE_ORDER.map((lane) => {
              const target: InboxFolder = { kind: "lane", lane };
              const active = isSameFolder(folder, target);
              return (
                <button
                  key={lane}
                  type="button"
                  aria-pressed={active}
                  onClick={() => selectFolder(target)}
                  className={cn(
                    "ui-focus-ring shrink-0 rounded-full px-2.5 py-1 text-xs transition-colors",
                    active
                      ? "bg-nav-active text-nav-active-foreground"
                      : "text-foreground-muted ring-foreground/10 ring-1",
                  )}
                >
                  {laneLabel(t, lane)}
                </button>
              );
            })}
          </div>
          <MessageList
            title={title}
            list={{
              messages,
              isLoading: messagesQuery.isLoading,
              isError: messagesQuery.isError,
              hasNextPage: messagesQuery.hasNextPage,
              isFetchingNextPage: messagesQuery.isFetchingNextPage,
              fetchNextPage: () => void messagesQuery.fetchNextPage(),
              retry: () => void messagesQuery.refetch(),
            }}
            openId={openId}
            now={now}
            search={search}
            searchRef={searchRef}
            onSearchChange={setSearch}
            onOpen={openMessage}
            empty={emptyFor(t, folder, counts?.handled ?? 0)}
          />
        </div>

        <main
          className={cn("min-w-0 flex-1 flex-col", openId === null ? "hidden lg:flex" : "flex")}
        >
          {openId === null ? (
            <NothingOpen waiting={counts?.waiting ?? 0} />
          ) : (
            <ReadingPane
              key={openId}
              messageId={openId}
              now={now}
              actions={actions}
              linkOpen={linkOpen}
              onLinkOpenChange={setLinkOpen}
              onClose={() => openMessage(null)}
            />
          )}
        </main>
      </div>
    </PageLayout>
  );
}

function folderTitle(
  t: (value: string, ...args: unknown[]) => string,
  folder: InboxFolder,
  mailboxes: { id: string; name: string; address: string }[],
): string {
  switch (folder.kind) {
    case "lane":
      return laneLabel(t, folder.lane);
    case "classification":
      return classificationLabel(t, folder.classification);
    case "mailbox": {
      const mailbox = mailboxes.find((row) => row.id === folder.mailboxId);
      if (mailbox === undefined) {
        return t("Mailbox");
      }
      return mailbox.name === "" ? mailbox.address : mailbox.name;
    }
  }
}

function emptyFor(
  t: (value: string, ...args: unknown[]) => string,
  folder: InboxFolder,
  handled: number,
): { title: string; description: string } {
  if (folder.kind === "lane" && folder.lane === "waiting") {
    return {
      title: t("Nothing is waiting on you"),
      description:
        handled > 0
          ? t(
              "{0, plural, one {The desk has handled # message. It is under Handled.} other {The desk has handled # messages. They are under Handled.}}",
              handled,
            )
          : t(
              "Tenders, rate confirmations, proofs of delivery and status requests land here when the desk needs a person.",
            ),
    };
  }
  if (folder.kind === "classification") {
    return {
      title: t("No {0} mail yet", classificationLabel(t, folder.classification).toLowerCase()),
      description: t("Messages the desk reads as this kind will collect here."),
    };
  }

  return {
    title: t("Nothing here"),
    description: t("Mail appears here as it arrives on a monitored address."),
  };
}

function NothingOpen({ waiting }: { waiting: number }) {
  const t = useT();

  return (
    <div className="flex flex-1 flex-col items-center justify-center gap-3 p-8 text-center">
      <MailOpenIcon className="text-foreground-subtle size-8" aria-hidden />
      <p className="text-sm font-medium">
        {waiting > 0
          ? t(
              "{0, plural, one {# message is waiting on you} other {# messages are waiting on you}}",
              waiting,
            )
          : t("Choose a message to read")}
      </p>
      <p className="text-foreground-subtle max-w-xs text-xs leading-relaxed">
        {t("Press j to open the first one, and keep pressing it to work down the list.")}
      </p>
    </div>
  );
}
