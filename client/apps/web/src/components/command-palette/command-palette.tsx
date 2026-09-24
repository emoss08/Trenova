import { useAsk } from "@/components/assistant/use-ask";
import { usePageContext } from "@/components/assistant/use-page-context";
import { conversationPath } from "@/lib/conversation-path";
import { useCommandPaletteStore } from "@/stores/command-palette-store";
import { useRecentRecordsStore } from "@/stores/recent-records-store";
import type { AssistantEntityRef } from "@/types/assistant";
import { CommandDialog, CommandList } from "@trenova/shared/components/ui/command";
import { useDebounce } from "@trenova/shared/hooks/use-debounce";
import { useT } from "@trenova/shared/i18n/use-t";
import { rankByFuzzyScore } from "@trenova/shared/lib/fuzzy-score";
import { cn } from "@trenova/shared/lib/utils";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { useQueryClient } from "@tanstack/react-query";
import { SearchXIcon } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { prefetchRecordPreview } from "./_components/preview/preview-queries";
import { PalettePreview } from "./_components/preview/palette-preview";
import { AskAnswerCard } from "./ask-answer-card";
import { askQuestion } from "./ask-question";
import {
  ACTION_COPY_ID,
  ACTION_COPY_LINK,
  ACTION_NEW_TAB,
  buildItemActions,
  findAction,
} from "./palette-actions";
import { actionItemKey, PaletteActionsView } from "./palette-actions-view";
import { PaletteFooter, type FooterHint } from "./palette-footer";
import { PaletteInput } from "./palette-input";
import { PaletteList } from "./palette-list";
import {
  intentKeepsPaletteOpen,
  isRecordScope,
  type PaletteAction,
  type PaletteItem,
  type PaletteScope,
  type PaletteSection,
} from "./palette-model";
import { nextScope, PaletteScopeTabs } from "./palette-scope-tabs";
import {
  buildHomeSections,
  buildSearchSections,
  flattenSections,
  recordsFromGroups,
} from "./palette-sections";
import {
  filterMentionOptions,
  getMentionState,
  resolveMentionCommit,
  stripMentionToken,
  type SearchEntityOption,
} from "./search-entity-filter";
import {
  usePaletteCatalog,
  usePaletteHome,
  usePaletteRemoteSearch,
  usePinnedPages,
  useUnreadCount,
} from "./use-palette-data";
import { usePaletteIntentRunner } from "./use-palette-intents";

const SEARCH_DEBOUNCE_MS = 200;
const PREVIEW_SETTLE_MS = 90;

interface ActionsView {
  item: PaletteItem;
  title: string;
  /** What was typed before the actions opened, given back on the way out. */
  savedQuery: string;
}

interface MentionState {
  open: boolean;
  text: string;
  index: number;
}

const CLOSED_MENTION: MentionState = { open: false, text: "", index: 0 };

function itemTitle(item: PaletteItem): string {
  switch (item.kind) {
    case "record":
      return item.record.title;
    case "page":
      return item.page.title;
    case "command":
      return item.command.label;
    case "attention":
      return item.attention.label;
    case "notification":
      return item.notification.title;
    case "ask":
      return item.question;
  }
}

function caretAtEnd(input: HTMLInputElement): boolean {
  return input.selectionStart === input.value.length && input.selectionEnd === input.value.length;
}

/**
 * The command palette. It answers three questions from one box — where is
 * it, what can I do, and what is waiting on me — and keeps a live look at
 * whatever is selected beside the list. Nothing is fetched until it opens.
 */
export function CommandPalette() {
  const t = useT();
  const queryClient = useQueryClient();
  const open = useCommandPaletteStore((state) => state.open);
  const setOpen = useCommandPaletteStore((state) => state.setOpen);
  const organizationId = useAuthStore((state) => state.user?.currentOrganizationId);
  const recordOpen = useRecentRecordsStore((state) => state.recordOpen);

  const [query, setQuery] = useState("");
  const [scope, setScope] = useState<PaletteScope>("all");
  const [mention, setMention] = useState<MentionState>(CLOSED_MENTION);
  const [actionsView, setActionsView] = useState<ActionsView | null>(null);
  const [selected, setSelected] = useState("");
  const [askSubject, setAskSubject] = useState<AssistantEntityRef | null>(null);
  const inputRef = useRef<HTMLInputElement | null>(null);

  const unread = useUnreadCount();
  const catalog = usePaletteCatalog(unread);
  const pinned = usePinnedPages(open, catalog.pageIndex);
  const home = usePaletteHome({ open, catalog, pinned, unread });
  const debouncedQuery = useDebounce(query, SEARCH_DEBOUNCE_MS);
  const question =
    actionsView !== null
      ? null
      : askSubject !== null
        ? query.trim() || null
        : scope === "all"
          ? askQuestion(query)
          : null;
  const remote = usePaletteRemoteSearch({
    enabled: open && actionsView === null && askSubject === null && question === null,
    query: debouncedQuery,
    scope,
  });

  const getPageContext = usePageContext();
  const { turn, ask, stop: stopAsk, reset: resetAsk, keep: keepAsk } = useAsk();

  const reset = useCallback(() => {
    setQuery("");
    setScope("all");
    setMention(CLOSED_MENTION);
    setActionsView(null);
    setSelected("");
    setAskSubject(null);
    resetAsk();
  }, [resetAsk]);

  const close = useCallback(() => {
    setOpen(false);
    reset();
  }, [reset, setOpen]);

  const askAbout = useCallback((subject: AssistantEntityRef) => {
    setAskSubject(subject);
    setActionsView(null);
    setScope("all");
    setQuery("");
    setSelected("");
    inputRef.current?.focus();
  }, []);

  const runIntent = usePaletteIntentRunner({ close, askAbout });

  const actionContext = useMemo(
    () => ({ ...catalog.actionContext, pinnedUrls: pinned.pinnedUrls }),
    [catalog.actionContext, pinned.pinnedUrls],
  );

  const sections = useMemo((): PaletteSection[] => {
    if (actionsView !== null) {
      return [];
    }
    if (askSubject !== null) {
      return question === null
        ? []
        : [{ id: "ask", heading: t("Assistant"), items: [{ kind: "ask", key: "ask", question }] }];
    }
    if (query.trim() === "" && scope === "all") {
      return buildHomeSections(home, t);
    }
    return buildSearchSections(
      {
        query,
        scope,
        question,
        remote,
        pages: catalog.pages,
        commands: catalog.commands,
        recentRecords: home.recentRecords,
      },
      t,
    );
  }, [
    actionsView,
    askSubject,
    catalog.commands,
    catalog.pages,
    home,
    query,
    question,
    remote,
    scope,
    t,
  ]);

  const items = useMemo(() => flattenSections(sections), [sections]);
  const itemsByKey = useMemo(() => new Map(items.map((item) => [item.key, item])), [items]);

  const viewActions = useMemo(() => {
    if (actionsView === null) {
      return [];
    }
    const all = buildItemActions(actionsView.item, actionContext, t);
    return rankByFuzzyScore(query, all, (action) => [{ text: action.label }]);
  }, [actionContext, actionsView, query, t]);

  const visibleKeys = useMemo(
    () => (actionsView !== null ? viewActions.map(actionItemKey) : items.map((item) => item.key)),
    [actionsView, items, viewActions],
  );
  const selectedKey = visibleKeys.includes(selected) ? selected : (visibleKeys[0] ?? "");
  const selectedItem =
    actionsView !== null ? actionsView.item : (itemsByKey.get(selectedKey) ?? null);
  const selectedActions = useMemo(
    () => (selectedItem ? buildItemActions(selectedItem, actionContext, t) : []),
    [actionContext, selectedItem, t],
  );

  const settledKey = useDebounce(
    actionsView !== null ? `view:${actionsView.item.key}` : selectedKey,
    PREVIEW_SETTLE_MS,
  );
  const previewItem =
    actionsView !== null
      ? actionsView.item
      : (itemsByKey.get(settledKey) ?? (settledKey === "" ? null : selectedItem));
  const previewActions = useMemo(
    () => (previewItem ? buildItemActions(previewItem, actionContext, t) : []),
    [actionContext, previewItem, t],
  );

  const firstRecord = useMemo(() => items.find((item) => item.kind === "record"), [items]);
  useEffect(() => {
    if (open && firstRecord?.kind === "record") {
      prefetchRecordPreview(queryClient, firstRecord.record);
    }
  }, [firstRecord, open, queryClient]);

  const relatedCommands = useCallback(
    (href: string) =>
      catalog.commands.filter(
        (command) =>
          command.group === "create" &&
          command.intent.type === "navigate" &&
          command.intent.href.split("?")[0] === href,
      ),
    [catalog.commands],
  );

  const handleAsk = useCallback(() => {
    if (question === null) {
      return;
    }
    void ask(question, {
      context: getPageContext(),
      mentions: askSubject ? [askSubject] : undefined,
    });
  }, [ask, askSubject, getPageContext, question]);

  const handleOpenInDesk = useCallback(() => {
    void keepAsk().then((threadId) => {
      if (threadId !== null) {
        runIntent({ type: "navigate", href: conversationPath(threadId) });
      }
    });
  }, [keepAsk, runIntent]);

  const runAction = useCallback(
    (item: PaletteItem, action: PaletteAction) => {
      const intent = action.intent;
      if (
        item.kind === "record" &&
        (intent.type === "navigate" || intent.type === "new-tab" || intent.type === "open-document")
      ) {
        recordOpen(organizationId, item.record);
      }
      if (intentKeepsPaletteOpen(intent) && actionsView !== null) {
        setActionsView(null);
        setQuery(actionsView.savedQuery);
      }
      runIntent(intent);
    },
    [actionsView, organizationId, recordOpen, runIntent],
  );

  const handleSelect = useCallback(
    (item: PaletteItem) => {
      if (item.kind === "ask") {
        handleAsk();
        return;
      }
      const primary = buildItemActions(item, actionContext, t)[0];
      if (primary) {
        runAction(item, primary);
      }
    },
    [actionContext, handleAsk, runAction, t],
  );

  const openActions = useCallback(() => {
    if (!selectedItem || selectedActions.length === 0 || selectedItem.kind === "ask") {
      return;
    }
    setActionsView({ item: selectedItem, title: itemTitle(selectedItem), savedQuery: query });
    setQuery("");
    setSelected("");
  }, [query, selectedActions.length, selectedItem]);

  const closeActions = useCallback(() => {
    if (actionsView === null) {
      return;
    }
    setQuery(actionsView.savedQuery);
    setSelected(actionsView.item.key);
    setActionsView(null);
  }, [actionsView]);

  const mentionOptions = useMemo(() => filterMentionOptions(mention.text), [mention.text]);

  const pickMention = useCallback((option: SearchEntityOption) => {
    setScope(option.key);
    setQuery((current) => stripMentionToken(current));
    setMention(CLOSED_MENTION);
  }, []);

  const handleQueryChange = useCallback(
    (value: string) => {
      if (actionsView !== null || askSubject !== null) {
        setQuery(value);
        return;
      }
      const committed = value.match(/@([a-zA-Z]+)\s$/);
      if (committed) {
        const entity = resolveMentionCommit(committed[1] ?? "");
        if (entity) {
          setScope(entity);
          setQuery(stripMentionToken(value));
          setMention(CLOSED_MENTION);
          return;
        }
      }
      const state = getMentionState(value);
      setMention({ open: state.mentionOpen, text: state.mentionText, index: 0 });
      setQuery(value);
    },
    [actionsView, askSubject],
  );

  const runShortcut = useCallback(
    (event: React.KeyboardEvent, actionId: string) => {
      const action = findAction(selectedActions, actionId);
      if (!action || !selectedItem) {
        return false;
      }
      event.preventDefault();
      runAction(selectedItem, action);
      return true;
    },
    [runAction, selectedActions, selectedItem],
  );

  const handleKeyDown = (event: React.KeyboardEvent<HTMLInputElement>) => {
    const mod = event.metaKey || event.ctrlKey;
    const key = event.key.toLowerCase();

    if (mention.open && mentionOptions.length > 0) {
      if (event.key === "ArrowDown" || event.key === "ArrowUp") {
        event.preventDefault();
        const step = event.key === "ArrowDown" ? 1 : -1;
        setMention((current) => ({
          ...current,
          index: (current.index + step + mentionOptions.length) % mentionOptions.length,
        }));
        return;
      }
      if (event.key === "Enter" || event.key === "Tab") {
        const option = mentionOptions[mention.index];
        if (option) {
          event.preventDefault();
          pickMention(option);
        }
        return;
      }
    }

    if (event.key === "Tab") {
      event.preventDefault();
      if (actionsView === null && askSubject === null) {
        setScope((current) => nextScope(current, event.shiftKey ? -1 : 1));
        setSelected("");
      }
      return;
    }

    if (mod && key === "k") {
      event.preventDefault();
      if (actionsView !== null) {
        closeActions();
      } else {
        openActions();
      }
      return;
    }

    if (actionsView !== null) {
      if ((event.key === "Backspace" || event.key === "ArrowLeft") && query === "") {
        event.preventDefault();
        closeActions();
      }
      return;
    }

    if (
      event.key === "ArrowRight" &&
      selectedActions.length > 1 &&
      caretAtEnd(event.currentTarget)
    ) {
      event.preventDefault();
      openActions();
      return;
    }

    if (mod && event.key === "Enter" && runShortcut(event, ACTION_NEW_TAB)) {
      return;
    }
    if (mod && !event.shiftKey && key === "l" && runShortcut(event, ACTION_COPY_LINK)) {
      return;
    }
    if (event.altKey && !mod && event.code === "KeyC" && runShortcut(event, ACTION_COPY_ID)) {
      return;
    }

    if (event.key === "Backspace" && query === "") {
      if (askSubject !== null) {
        event.preventDefault();
        setAskSubject(null);
        resetAsk();
      } else if (scope !== "all") {
        event.preventDefault();
        setScope("all");
      }
    }
  };

  const recordCounts = useMemo(() => {
    const counts: Partial<Record<PaletteScope, number>> = {};
    if (!remote.ready || remote.loading) {
      return counts;
    }
    for (const [type, records] of recordsFromGroups(remote.groups)) {
      counts[type as PaletteScope] = records.length;
    }
    return counts;
  }, [remote.groups, remote.loading, remote.ready]);

  const primaryAction = selectedActions[0];
  const hints: FooterHint[] =
    actionsView !== null
      ? [
          { keys: ["↑", "↓"], label: t("Navigate") },
          { keys: ["↵"], label: t("Run") },
          { keys: ["←"], label: t("Back") },
        ]
      : [
          { keys: ["↑", "↓"], label: t("Navigate") },
          ...(selectedItem?.kind === "ask"
            ? [{ keys: ["↵"], label: t("Ask") }]
            : selectedItem?.kind === "command"
              ? [{ keys: ["↵"], label: t("Run") }]
              : primaryAction
                ? [{ keys: ["↵"], label: t("Open") }]
                : []),
          ...(findAction(selectedActions, ACTION_NEW_TAB)
            ? [{ keys: [actionContext.mac ? "⌘" : "Ctrl", "↵"], label: t("New tab") }]
            : []),
          ...(selectedActions.length > 1 ? [{ keys: ["→"], label: t("Actions") }] : []),
          ...(askSubject === null ? [{ keys: ["Tab"], label: t("Scope") }] : []),
          { keys: ["Esc"], label: t("Close") },
        ];

  const showAnswer = turn !== null && question !== null;
  const hasRows = actionsView !== null ? viewActions.length > 0 : items.length > 0;
  const stillLoading = sections.some((section) => section.loading);
  const hasErrorSection = sections.some((section) => section.error);
  const showEmpty =
    !hasRows &&
    !stillLoading &&
    !hasErrorSection &&
    !showAnswer &&
    (query.trim() !== "" || scope !== "all" || actionsView !== null);

  return (
    <CommandDialog
      open={open}
      onOpenChange={(next, eventDetails) => {
        if (!next && eventDetails.reason === "escape-key") {
          if (mention.open) {
            eventDetails.cancel();
            setMention(CLOSED_MENTION);
            return;
          }
          if (actionsView !== null) {
            eventDetails.cancel();
            closeActions();
            return;
          }
        }
        if (next) {
          setOpen(true);
        } else {
          close();
        }
      }}
      title={t("Command palette")}
      description={t("Search records, jump to pages, run commands and ask the assistant")}
      showCloseButton={false}
      className={cn(
        "top-[12vh] flex h-[min(40rem,calc(100dvh-16vh))] w-full translate-y-0 flex-col gap-0 overflow-hidden p-0",
        "rounded-surface bg-overlay ui-lift-float sm:max-w-4xl",
      )}
      commandProps={{
        shouldFilter: false,
        loop: true,
        value: selectedKey,
        onValueChange: setSelected,
        className: "flex h-full min-h-0 flex-col rounded-none bg-transparent",
      }}
    >
      <PaletteInput
        ref={inputRef}
        value={query}
        onValueChange={handleQueryChange}
        onKeyDown={handleKeyDown}
        scope={scope}
        onClearScope={() => setScope("all")}
        actionsFor={actionsView?.title ?? null}
        onBack={closeActions}
        askSubject={askSubject?.label ?? null}
        onClearAskSubject={() => {
          setAskSubject(null);
          resetAsk();
        }}
        busy={remote.loading}
        mention={{
          open: mention.open,
          options: mentionOptions,
          index: mention.index,
          onPick: pickMention,
        }}
      />
      {actionsView === null && askSubject === null ? (
        <PaletteScopeTabs
          scope={scope}
          onScopeChange={(next) => {
            setScope(next);
            setSelected("");
            inputRef.current?.focus();
          }}
          counts={recordCounts}
        />
      ) : (
        <div aria-hidden className="border-border-subtle border-b" />
      )}
      <div className="grid min-h-0 flex-1 md:grid-cols-[minmax(0,1fr)_minmax(0,22rem)]">
        <div className="flex min-h-0 flex-col">
          {showAnswer && (
            <AskAnswerCard
              question={question}
              turn={turn}
              onAsk={handleAsk}
              onStop={stopAsk}
              onOpenInDesk={handleOpenInDesk}
              className="mx-3 mt-3"
            />
          )}
          <CommandList className="max-h-none min-h-0 flex-1 scroll-py-2 pb-2">
            {actionsView !== null ? (
              <PaletteActionsView
                title={actionsView.title}
                actions={viewActions}
                query={query}
                onRun={(action) => runAction(actionsView.item, action)}
              />
            ) : (
              <PaletteList
                sections={sections}
                query={query}
                onSelect={handleSelect}
                onRetry={remote.retry}
              />
            )}
            {showEmpty && (
              <div className="flex flex-col items-center gap-2 px-6 py-14 text-center">
                <SearchXIcon className="text-foreground-subtle size-5" />
                <p className="text-foreground text-sm font-medium">
                  {actionsView !== null ? t("No matching actions") : t("Nothing matches that")}
                </p>
                <p className="text-foreground-subtle max-w-72 text-xs">
                  {isRecordScope(scope)
                    ? t("Try a different spelling, or press Backspace to search everything.")
                    : t("Try fewer words, or end with a question mark to ask the assistant.")}
                </p>
              </div>
            )}
          </CommandList>
        </div>
        <aside
          aria-label={t("Preview")}
          className="border-border-subtle bg-card hidden min-h-0 border-l md:block"
        >
          <PalettePreview
            item={previewItem}
            actions={previewActions}
            onRun={(intent) => {
              if (previewItem) {
                const action = previewActions.find((candidate) => candidate.intent === intent);
                if (action) {
                  runAction(previewItem, action);
                  return;
                }
              }
              runIntent(intent);
            }}
            relatedCommands={relatedCommands}
          />
        </aside>
      </div>
      <PaletteFooter
        hints={hints}
        showAskHint={
          actionsView === null && askSubject === null && question === null && scope === "all"
        }
      />
    </CommandDialog>
  );
}
