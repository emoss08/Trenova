import type { VirtualRowsHandle } from "@/components/virtual-rows";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { handleMutationError } from "@/hooks/use-api-mutation";
import { useMediaQuery } from "@/hooks/use-media-query";
import { useNowSeconds } from "@/hooks/use-now-seconds";
import { useUserTimezone } from "@/hooks/use-user-timezone";
import type { InboundMessageFilter, TriageView } from "@/lib/graphql/inbox";
import type { SelectOption } from "@/lib/graphql/select-options";
import type { GenericLimitOffsetResponse } from "@trenova/shared/types/server";
import { queries } from "@/lib/queries";
import { inboxMessagesQuery } from "@/lib/queries/inbox";
import type { RoutePrefetch, RoutePrefetchEntry } from "@/lib/route-prefetch";
import { knownRow } from "@/lib/triage/cache";
import { useTriageStore } from "@/stores/triage-store";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  ContextMenuHost,
  isContextMenuOpen,
} from "@trenova/shared/components/context-menu/context-menu-host";
import { CheckIcon, LayoutLeftIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import { getStartOfDay } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import {
  lazy,
  Suspense,
  useCallback,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  type ReactNode,
} from "react";
import { useNavigate, useSearchParams } from "react-router";
import { USER_FORMATTERS } from "./_components/action-copy";
import { folderTitle } from "./_components/folder-labels";
import {
  BRIEFING,
  WAITING,
  folderFilter,
  folderHasWaiting,
  folderKey,
  folderParams,
  isQueueFolder,
  messageParams,
  parseTriageLocation,
  resolveSort,
  searchParamsWith,
  sortParams,
  type FolderSort,
  type QueueFolder,
  type TriageFolder,
} from "./_components/folders";
import { stepFrom, useTriageKeys, type TriageCommand } from "./_components/keys";
import { LinkMessageDialog } from "./_components/link-dialog";
import { buildListItems } from "./_components/list/list-items";
import {
  ConnectFirstSource,
  FirstMessage,
  ListEmpty,
  NovaOffBanner,
  SourceHealthBanner,
} from "./_components/list/list-states";
import { TriageList } from "./_components/list/triage-list";
import type { RowHandlers } from "./_components/list/triage-row";
import { rowMenu } from "./_components/menus";
import { ShortcutsSheet } from "./_components/shortcuts-sheet";
import { LaneStrip } from "./_components/sidebar/lane-strip";
import { TriageSidebar } from "./_components/sidebar/triage-sidebar";
import { TriageNavigationContext, type TriageNavigation } from "./_components/triage-navigation";
import {
  TriageActionsProvider,
  useTriageActions,
  useTriageActionsController,
} from "./_components/use-triage-actions";
import {
  useFolderPages,
  usePrefetchMessage,
  useTriageAssignees,
  useTriageBriefing,
  useTriageCounts,
  useTriageDocumentTypes,
  useTriageMailboxes,
  useTriageViews,
} from "./_components/use-triage-data";
import { useTriagePresenceScope } from "./_components/use-triage-presence";

const BriefingView = lazy(() =>
  import("./_components/briefing/briefing-view").then((module) => ({
    default: module.BriefingView,
  })),
);
const RulesView = lazy(() =>
  import("./_components/rules/rules-view").then((module) => ({ default: module.RulesView })),
);
const Reader = lazy(() =>
  import("./_components/reader/reader").then((module) => ({ default: module.Reader })),
);

/** Where the reader sits beside the list rather than in its place. */
const WIDE_QUERY = "(min-width: 1024px)";
const AI_CONTROL_PATH = "/admin/agent-control";
const SOURCES_PATH = "/admin/inbound-mailboxes";

/**
 * Warms what the page reads on mount: the counts, the sources and views the
 * rail draws, the open folder's first page (or the Briefing or the rules),
 * and an open message. A view's folder needs the view to build its search,
 * so it is left to the page.
 */
export const prefetch: RoutePrefetch = ({ request }) => {
  const location = parseTriageLocation(new URL(request.url).searchParams);
  const entries: RoutePrefetchEntry[] = [
    queries.inbox.counts(),
    queries.inbox.mailboxes(),
    queries.inbox.views(),
  ];
  const folder = location.folder;
  if (folder.kind === "briefing") {
    entries.push(queries.inbox.briefing());
  } else if (folder.kind === "rules") {
    entries.push(queries.inbox.rules());
  } else if (folder.kind !== "view") {
    const settled =
      folder.kind === "lane" && (folder.lane === "handled" || folder.lane === "ignored");
    const filter = folderFilter(folder, {
      search: location.search,
      sort: resolveSort(location.sort, !settled),
      views: [],
    });
    if (filter !== null) {
      entries.push(inboxMessagesQuery(filter));
    }
  }
  if (location.messageId !== null) {
    entries.push(queries.inbox.message(location.messageId));
  }

  return entries;
};

function startOfUserDay(seconds: number): number {
  return getStartOfDay(new Date(seconds * 1000));
}

function Lazy({ children }: { children: ReactNode }) {
  return <Suspense fallback={<div className="min-h-0 flex-1" />}>{children}</Suspense>;
}

/**
 * Triage: messages that arrived from connected sources, what Nova did with
 * them, and what is left for a person, each with its next step worked out.
 *
 * The address holds where a person is — the Briefing, Rules, or a folder of
 * the queue with its search and order, and the open message — so a link and
 * the back button land where they look. The page owns that state and hands
 * the rest of Triage two stable things: navigation (which message is open,
 * the folder's order) and the one set of actions.
 */
export function TriagePage() {
  const t = useT();
  const [searchParams, setSearchParams] = useSearchParams();
  const location = useMemo(() => parseTriageLocation(searchParams), [searchParams]);
  const wide = useMediaQuery(WIDE_QUERY);
  const state = useRef({
    openId: location.messageId,
    orderedIds: [] as readonly string[],
    filter: null as InboundMessageFilter | null,
    wide,
  });

  useEffect(() => useTriageStore.getState().mount(), []);
  useTriagePresenceScope();

  const navigation = useMemo<TriageNavigation>(
    () => ({
      openId: () => state.current.openId,
      orderedIds: () => state.current.orderedIds,
      filter: () => state.current.filter,
      isWide: () => state.current.wide,
      open: (id) => setSearchParams((current) => messageParams(current, id)),
      openInQueue: (id) =>
        setSearchParams((current) => messageParams(folderParams(current, WAITING), id)),
    }),
    [setSearchParams],
  );

  const publish = useCallback((next: WorkspaceState) => {
    state.current = next;
  }, []);

  const railCollapsed = useTriageStore((store) => store.railCollapsed);
  const toggleRail = useTriageStore((store) => store.toggleRail);

  return (
    <PageLayout
      fill
      className="p-0"
      pageHeaderProps={{
        title: t("Triage"),
        description: t(
          "Messages from connected sources, what Nova did with them on its own, and what is left for a person, each with its next step",
        ),
        actions: (
          <Tooltip>
            <TooltipTrigger
              render={
                <Button
                  size="icon-sm"
                  variant="quiet"
                  className="hidden md:inline-flex"
                  aria-label={railCollapsed ? t("Show folders") : t("Hide folders")}
                  aria-pressed={!railCollapsed}
                  onClick={toggleRail}
                />
              }
            >
              <LayoutLeftIcon aria-hidden className="size-4" />
            </TooltipTrigger>
            <TooltipContent>{railCollapsed ? t("Show folders") : t("Hide folders")}</TooltipContent>
          </Tooltip>
        ),
      }}
    >
      <TriageNavigationContext.Provider value={navigation}>
        <ActionsScope>
          <TriageWorkspace
            location={location}
            wide={wide}
            publish={publish}
            setSearchParams={setSearchParams}
          />
        </ActionsScope>
      </TriageNavigationContext.Provider>
      <ContextMenuHost />
    </PageLayout>
  );
}

function ActionsScope({ children }: { children: ReactNode }) {
  const actions = useTriageActionsController();

  return <TriageActionsProvider value={actions}>{children}</TriageActionsProvider>;
}

type WorkspaceState = {
  openId: string | null;
  orderedIds: readonly string[];
  filter: InboundMessageFilter | null;
  wide: boolean;
};

const NO_VIEWS: readonly TriageView[] = [];

function TriageWorkspace({
  location,
  wide,
  publish,
  setSearchParams,
}: {
  location: ReturnType<typeof parseTriageLocation>;
  wide: boolean;
  /** Hands the page what navigation reads: the open message, the order, the filter. */
  publish: (state: WorkspaceState) => void;
  setSearchParams: ReturnType<typeof useSearchParams>[1];
}) {
  const t = useT();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const actions = useTriageActions();
  const timeZone = useUserTimezone();
  const now = useNowSeconds();
  const meId = useAuthStore((store) => store.user?.id ?? null);
  const prefetchMessage = usePrefetchMessage();
  const searchRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<VirtualRowsHandle>(null);
  const autoOpened = useRef<string | null>(null);

  const overlay = useTriageStore((store) => store.overlay);
  const closeOverlay = useTriageStore((store) => store.closeOverlay);
  const railCollapsed = useTriageStore((store) => store.railCollapsed);

  const { folder, messageId: openId, search } = location;
  const counts = useTriageCounts().data;
  const mailboxesQuery = useTriageMailboxes();
  const mailboxes = mailboxesQuery.data ?? [];
  const views = useTriageViews().data ?? NO_VIEWS;
  const briefing = useTriageBriefing().data;
  useTriageAssignees();
  useTriageDocumentTypes();

  const queueFolder: QueueFolder | null = isQueueFolder(folder) ? folder : null;
  const hasWaiting = queueFolder === null ? false : folderHasWaiting(queueFolder, counts, views);
  const sort = resolveSort(location.sort, hasWaiting);
  const filter = useMemo(
    () => (queueFolder === null ? null : folderFilter(queueFolder, { search, sort, views })),
    [queueFolder, search, sort, views],
  );
  const pages = useFolderPages(filter);
  const items = useMemo(
    () => buildListItems(t, pages.messages, sort, now, startOfUserDay),
    [t, pages.messages, sort, now],
  );
  const orderedIds = useMemo(
    () => items.flatMap((item) => (item.type === "message" ? [item.message.id] : [])),
    [items],
  );

  const state = useRef<WorkspaceState>({ openId, orderedIds, filter, wide });
  useLayoutEffect(() => {
    const snapshot = { openId, orderedIds, filter, wide };
    state.current = snapshot;
    publish(snapshot);
  });

  const update = useCallback(
    (change: (current: URLSearchParams) => URLSearchParams, replace = false) =>
      setSearchParams((current) => change(current), { replace }),
    [setSearchParams],
  );
  const openMessage = useCallback(
    (id: string | null, replace = false) =>
      update((current) => messageParams(current, id), replace),
    [update],
  );
  const selectFolder = useCallback(
    (next: TriageFolder) => {
      useTriageStore.getState().clearSelection();
      update((current) => folderParams(current, next));
    },
    [update],
  );

  const commitSearch = useCallback(
    (value: string) => update((current) => searchParamsWith(current, value), true),
    [update],
  );
  const changeSort = useCallback(
    (next: FolderSort) => update((current) => sortParams(current, next), true),
    [update],
  );

  const folderKeyValue = folderKey(folder);
  useEffect(() => {
    if (
      !wide ||
      queueFolder === null ||
      openId !== null ||
      pages.isLoading ||
      orderedIds.length === 0 ||
      autoOpened.current === folderKeyValue
    ) {
      return;
    }
    autoOpened.current = folderKeyValue;
    openMessage(orderedIds[0], true);
  }, [wide, queueFolder, openId, pages.isLoading, orderedIds, folderKeyValue, openMessage]);

  useEffect(() => {
    if (openId === null) {
      return;
    }
    listRef.current?.scrollToKey(openId);
    const index = orderedIds.indexOf(openId);
    for (const neighbour of [orderedIds[index + 1], orderedIds[index - 1]]) {
      if (neighbour !== undefined) {
        prefetchMessage(neighbour);
      }
    }
  }, [openId, orderedIds, prefetchMessage]);

  const handlers = useMemo<RowHandlers>(
    () => ({
      open: (id) => openMessage(id),
      toggleSelected: (id, extend) => {
        const store = useTriageStore.getState();
        if (extend) {
          store.selectRange(state.current.orderedIds, id);
        } else {
          store.toggleSelected(id);
        }
      },
      swipe: (id, outcome) =>
        outcome === "handle" ? actions.handle([id]) : actions.snoozeUntilTomorrow([id]),
      prefetch: prefetchMessage,
      menu: (id) => {
        const message = knownRow(queryClient, id);
        if (message === undefined) {
          return null;
        }
        const selected = useTriageStore.getState().selected;
        const selection =
          selected.has(id) && selected.size > 1
            ? [...selected].flatMap((selectedId) => {
                const row = knownRow(queryClient, selectedId);
                return row === undefined ? [] : [row];
              })
            : [];
        return rowMenu({
          t,
          actions,
          message,
          selection,
          meId,
          assignees: queryClient.getQueryData(queries.inbox.assignees().queryKey) ?? [],
          documentTypes:
            queryClient.getQueryData<GenericLimitOffsetResponse<SelectOption>>(
              queries.inbox.documentTypes().queryKey,
            )?.results ?? [],
          time: { now: Math.floor(Date.now() / 1000), timeZone, format: USER_FORMATTERS },
          open: (target) => openMessage(target),
          link: (target) => {
            openMessage(target);
            useTriageStore.getState().openOverlay({ kind: "link", messageId: target });
          },
        });
      },
    }),
    [openMessage, actions, prefetchMessage, queryClient, t, meId, timeZone, state],
  );

  const dispatch = useCallback(
    (command: TriageCommand) => {
      const store = useTriageStore.getState();
      const current = state.current.openId;
      const targets =
        store.selected.size > 0 ? [...store.selected] : current === null ? [] : [current];
      const thenClear = () => store.clearSelection();
      switch (command) {
        case "briefing":
          selectFolder(BRIEFING);
          return;
        case "next":
        case "previous": {
          const next = stepFrom(state.current.orderedIds, current, command === "next" ? 1 : -1);
          if (next !== null && next !== current) {
            openMessage(next);
          }
          return;
        }
        case "search":
          searchRef.current?.focus();
          return;
        case "togglePanel":
          store.togglePanel();
          return;
        case "doNextStep":
          store.readerCommands.doNextStep?.();
          return;
        case "reply":
          store.readerCommands.reply?.();
          return;
        case "handle":
          actions.handle(targets);
          thenClear();
          return;
        case "ignore":
          actions.ignore(targets);
          thenClear();
          return;
        case "snooze":
          actions.snoozeUntilTomorrow(targets);
          thenClear();
          return;
        case "assignToMe":
          actions.assignToMe(targets);
          thenClear();
          return;
        case "toggleSelect":
          if (current !== null) {
            store.toggleSelected(current);
          }
          return;
        case "link":
          if (current !== null) {
            store.openOverlay({ kind: "link", messageId: current });
          }
          return;
        case "askDesk": {
          const message = current === null ? undefined : knownRow(queryClient, current);
          if (message !== undefined) {
            actions.askInDesk(message);
          }
          return;
        }
        case "shortcuts":
          store.openOverlay({ kind: "shortcuts" });
          return;
        case "clearSelection":
          store.clearSelection();
          return;
        case "closePanel":
          store.setPanelOpen(false);
          return;
        case "closeMessage":
          openMessage(null);
          return;
      }
    },
    [actions, openMessage, queryClient, selectFolder, state],
  );

  useTriageKeys({
    enabled: true,
    dispatch,
    context: () => {
      const store = useTriageStore.getState();
      return {
        inQueue: state.current.filter !== null,
        hasOpenMessage: state.current.openId !== null,
        selectionSize: store.selected.size,
        panelOpen: store.panelOpen,
        overlayOpen: store.overlay !== null || isContextMenuOpen(),
      };
    },
  });

  const title = folderTitle(t, folder, mailboxes, views);
  const position = openId === null ? 0 : orderedIds.indexOf(openId) + 1;
  const mailbox =
    folder.kind === "mailbox" ? mailboxes.find((row) => row.id === folder.mailboxId) : undefined;

  const saveView = (query: string) => {
    if (query === "") {
      return;
    }
    const name = query.charAt(0).toLocaleUpperCase() + query.slice(1);
    void actions.createView(name, query).then(
      (view) => {
        update((current) =>
          searchParamsWith(folderParams(current, { kind: "view", viewId: view.id }), ""),
        );
      },
      (error: unknown) => handleMutationError({ error, resourceName: "View" }),
    );
  };

  const empty = (): ReactNode => {
    if (search.trim() !== "") {
      return (
        <ListEmpty
          title={t("Nothing matches “{0}”", search.trim())}
          description={t(
            "Search covers the sender, subject, customer and PRO across every source.",
          )}
        />
      );
    }
    if (mailboxesQuery.isSuccess && mailboxes.length === 0) {
      return <ConnectFirstSource onAdd={() => navigate(SOURCES_PATH)} />;
    }
    if (counts !== undefined && counts.total === 0 && mailboxes.length > 0) {
      return <FirstMessage mailbox={mailbox ?? mailboxes[0]} />;
    }
    const back = (
      <Button size="sm" variant="outline" onClick={() => selectFolder(BRIEFING)}>
        {t("Back to the briefing")}
      </Button>
    );
    if (folder.kind === "lane" && folder.lane === "waiting") {
      return (
        <ListEmpty
          title={t("All caught up")}
          description={t(
            "{0, plural, one {Nova handled # on its own.} other {Nova handled # on its own.}}",
            briefing?.handledByNovaToday ?? counts?.handled ?? 0,
          )}
        >
          {back}
        </ListEmpty>
      );
    }
    if (folder.kind === "lane" && folder.lane === "them") {
      return (
        <ListEmpty
          title={t("Not waiting on anyone")}
          description={t(
            "After you reply, choose “Wait for reply” and it parks here until they answer.",
          )}
        >
          {back}
        </ListEmpty>
      );
    }
    return (
      <ListEmpty
        title={t("Nothing here")}
        description={t("Messages appear here as they arrive from a connected source.")}
      >
        {back}
      </ListEmpty>
    );
  };

  const banner = (
    <>
      {mailbox !== undefined && <SourceHealthBanner mailbox={mailbox} now={now} />}
      {briefing !== undefined && !briefing.novaEnabled && (
        <NovaOffBanner
          onTurnOn={actions.readUnread}
          onOpenControl={() => navigate(AI_CONTROL_PATH)}
        />
      )}
    </>
  );

  return (
    <div className="flex min-h-0 flex-1">
      <aside
        aria-hidden={railCollapsed || undefined}
        inert={railCollapsed || undefined}
        className={cn(
          "border-border bg-canvas hidden shrink-0 flex-col overflow-hidden border-r transition-[width] duration-300 ease-settle md:flex",
          railCollapsed ? "w-0 border-r-0" : "w-58",
        )}
      >
        <div className="flex min-h-0 w-58 flex-1 flex-col">
          <TriageSidebar
            folder={folder}
            onSelect={selectFolder}
            onShortcuts={() => useTriageStore.getState().openOverlay({ kind: "shortcuts" })}
          />
        </div>
      </aside>

      <div className="flex min-w-0 flex-1 flex-col">
        <div className="md:hidden">
          <LaneStrip
            folder={folder}
            onSelect={selectFolder}
            onShortcuts={() => useTriageStore.getState().openOverlay({ kind: "shortcuts" })}
          />
        </div>

        {folder.kind === "briefing" && (
          <Lazy>
            <BriefingView
              onOpenMessage={(id) => navigationOpenInQueue(update, id)}
              onOpenFolder={selectFolder}
            />
          </Lazy>
        )}
        {folder.kind === "rules" && (
          <Lazy>
            <RulesView />
          </Lazy>
        )}
        {queueFolder !== null && (
          <div className="flex min-h-0 flex-1">
            <div
              className={cn(
                "border-border-subtle min-h-0 flex-col lg:flex lg:w-96 lg:shrink-0 lg:border-r",
                openId !== null && !wide ? "hidden" : "flex flex-1 lg:flex-none",
              )}
            >
              <TriageList
                searchPlaceholder={t("Search {0}", title.toLocaleLowerCase())}
                search={search}
                onSearch={commitSearch}
                onSaveView={saveView}
                searchRef={searchRef}
                sort={sort}
                canSort={hasWaiting}
                onSort={changeSort}
                items={items}
                messages={pages.messages}
                isLoading={pages.isLoading}
                openId={openId}
                now={now}
                handlers={handlers}
                listRef={listRef}
                onEndReached={pages.fetchNextPage}
                isFetchingNextPage={pages.isFetchingNextPage}
                banner={banner}
                empty={empty()}
              />
            </div>
            <section
              aria-label={t("Message")}
              className={cn(
                "min-w-0 flex-1 flex-col",
                openId === null && !wide ? "hidden" : "flex",
              )}
            >
              {openId === null ? (
                <NothingOpen
                  empty={!pages.isLoading && pages.messages.length === 0}
                  waitingLane={folder.kind === "lane" && folder.lane === "waiting"}
                  title={title}
                  onBack={() => selectFolder(BRIEFING)}
                />
              ) : (
                <Lazy>
                  <Reader
                    key={openId}
                    messageId={openId}
                    position={{ index: position, total: orderedIds.length }}
                    onClose={() => openMessage(null)}
                    onStep={(direction) => {
                      const next = stepFrom(orderedIds, openId, direction);
                      if (next !== null) {
                        openMessage(next);
                      }
                    }}
                  />
                </Lazy>
              )}
            </section>
          </div>
        )}
      </div>

      {overlay?.kind === "link" && (
        <LinkOverlay messageId={overlay.messageId} onClose={closeOverlay} />
      )}
      <ShortcutsSheet open={overlay?.kind === "shortcuts"} onClose={closeOverlay} />
    </div>
  );
}

function navigationOpenInQueue(
  update: (change: (current: URLSearchParams) => URLSearchParams) => void,
  id: string,
) {
  update((current) => messageParams(folderParams(current, WAITING), id));
}

function LinkOverlay({ messageId, onClose }: { messageId: string; onClose: () => void }) {
  const message = useQuery(queries.inbox.message(messageId)).data;
  if (message === undefined) {
    return null;
  }

  return (
    <LinkMessageDialog
      message={message}
      open
      onOpenChange={(open) => {
        if (!open) {
          onClose();
        }
      }}
      onLinked={onClose}
    />
  );
}

function NothingOpen({
  empty,
  waitingLane,
  title,
  onBack,
}: {
  empty: boolean;
  waitingLane: boolean;
  title: string;
  onBack: () => void;
}) {
  const t = useT();
  const rt = useRichT();

  if (!empty) {
    return (
      <div className="text-foreground-subtle flex flex-1 flex-col items-center justify-center gap-2 p-8 text-center text-sm">
        <p>{t("Choose a message to read")}</p>
        <p className="flex items-center gap-1 text-xs">
          {rt("Press <k>J</k> to open the next one.", { k: (chunks) => <Kbd>{chunks}</Kbd> })}
        </p>
      </div>
    );
  }

  return (
    <div className="animate-rise flex flex-1 flex-col items-center justify-center gap-2 p-8 text-center">
      <span className="bg-success-subtle text-success mb-1 flex size-10 items-center justify-center rounded-full">
        <CheckIcon aria-hidden className="size-4.5" strokeWidth={2.2} />
      </span>
      <p className="text-base font-semibold">
        {waitingLane ? t("You're all caught up") : t("Nothing in {0}", title.toLocaleLowerCase())}
      </p>
      <p className="text-foreground-subtle max-w-xs text-sm">
        {waitingLane
          ? t("Nova brings the next one here the moment it lands.")
          : t("Messages show up here as they match.")}
      </p>
      <Button size="sm" variant="outline" className="mt-2" onClick={onBack}>
        {t("Back to the briefing")}
      </Button>
    </div>
  );
}
