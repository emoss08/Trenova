import { useAsk } from "@/components/assistant/use-ask";
import { usePageContext } from "@/components/assistant/use-page-context";
import { isRecordEntityType, recordPath } from "@/config/record-links";
import { lazyComponent } from "@/lib/lazy-component";
import { conversationPath } from "@/lib/conversation-path";
import { isGlobalSearchEntityType } from "@/services/global-search";
import { useCommandPaletteStore } from "@/stores/command-palette-store";
import { useRecentRecordsStore } from "@/stores/recent-records-store";
import type { AssistantEntityRef } from "@/types/assistant";
import { CommandDialog } from "@trenova/shared/components/ui/command";
import { useDebounce } from "@trenova/shared/hooks/use-debounce";
import { useT } from "@trenova/shared/i18n/use-t";
import { getNameInitials } from "@trenova/shared/lib/utils";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { useQueryClient } from "@tanstack/react-query";
import { Suspense, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { PalettePreview, type PreviewContext } from "./_components/preview/palette-preview";
import { prefetchRecordPreview } from "./_components/preview/preview-queries";
import { answerSources, plainAnswer } from "./ask-answer";
import { askQuestion, commandModeQuery, questionSearchText } from "./ask-question";
import "./command-palette.css";
import { PALETTE_ENTITY_ICONS } from "./palette-entities";
import {
  ACTION_ASK,
  ACTION_COPY_ID,
  ACTION_COPY_LINK,
  ACTION_NEW_TAB,
  ACTION_PIN,
  buildItemActions,
  findAction,
  modKey,
} from "./palette-actions";
import { PaletteActionsMenu } from "./palette-actions-menu";
import { PaletteFooter, type FooterAside, type FooterHint } from "./palette-footer";
import { PaletteInput } from "./palette-input";
import { PaletteList } from "./palette-list";
import {
  intentKeepsPaletteOpen,
  itemTitle,
  type PaletteAction,
  type PaletteIcon,
  type PaletteIntent,
  type PaletteItem,
  type PaletteRecord,
  type PaletteScope,
  type PaletteSection,
} from "./palette-model";
import { nextScope, PaletteScopeTabs, scopePresentation } from "./palette-scope-tabs";
import {
  buildHomeSections,
  buildSearchSections,
  buildSubjectSections,
  flattenSections,
  rankLocal,
  recordsNamedIn,
  scopeCounts,
} from "./palette-sections";
import {
  filterMentionOptions,
  getMentionState,
  resolveMentionCommit,
  stripMentionToken,
  type SearchEntityOption,
} from "./search-entity-filter";
import {
  isRemoteSearchable,
  paletteSearchQuery,
  usePaletteCatalog,
  usePaletteHome,
  usePaletteRemoteSearch,
  usePinnedPages,
  useUnreadCount,
} from "./use-palette-data";
import { usePaletteIntentRunner } from "./use-palette-intents";

// The answer is drawn in the Desk's prose, with its markdown and code highlighting. The
// palette itself is read ahead of every first open, so the answer's code is read only once
// a question is asked; the model takes far longer to reply than the code takes to arrive.
const AnswerCard = lazyComponent(
  () => import("./ask-answer-card"),
  (module) => module.AskAnswerCard,
);

const SEARCH_DEBOUNCE_MS = 200;
const PREVIEW_SETTLE_MS = 90;
const CONFIRMATION_MS = 1800;

interface MentionState {
  open: boolean;
  text: string;
  index: number;
}

const CLOSED_MENTION: MentionState = { open: false, text: "", index: 0 };

/** The question Nova is answering, and the record it is about, if any. */
interface AskedQuestion {
  question: string;
  subject: AssistantEntityRef | null;
}

function caretAtEnd(input: HTMLInputElement): boolean {
  return input.selectionStart === input.value.length && input.selectionEnd === input.value.length;
}

function sameSubject(a: AssistantEntityRef | null, b: AssistantEntityRef | null): boolean {
  return a === b || (a !== null && b !== null && a.type === b.type && a.id === b.id);
}

/** A record an answer cited, as a palette record, when the palette can draw and open it. */
function recordFromRef(ref: AssistantEntityRef): PaletteRecord | null {
  if (!isGlobalSearchEntityType(ref.type)) {
    return null;
  }
  const href = isRecordEntityType(ref.type) ? recordPath(ref.type, ref.id) : "";
  return { entityType: ref.type, id: ref.id, title: ref.label, href, metadata: {} };
}

function itemTile(item: PaletteItem): { icon?: PaletteIcon; initials?: string } {
  switch (item.kind) {
    case "record":
      return item.record.entityType === "worker"
        ? { initials: getNameInitials(item.record.title) }
        : { icon: PALETTE_ENTITY_ICONS[item.record.entityType] };
    case "page":
      return { icon: item.page.icon };
    case "command":
      return { icon: item.command.icon };
    default:
      return {};
  }
}

/**
 * The command palette. It answers four questions from one box — where is it,
 * what can I do, what is waiting on me, and what does Nova make of it — and
 * keeps a live look at whatever is selected beside the list. Nothing is
 * fetched until it opens.
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
  const [actionsOpen, setActionsOpen] = useState(false);
  const [selected, setSelected] = useState("");
  const [askSubject, setAskSubject] = useState<AssistantEntityRef | null>(null);
  const [asked, setAsked] = useState<AskedQuestion | null>(null);
  const [peek, setPeek] = useState<AssistantEntityRef | null>(null);
  const [confirmation, setConfirmation] = useState<string | null>(null);
  const confirmationTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const inputRef = useRef<HTMLInputElement | null>(null);

  const unread = useUnreadCount();
  const catalog = usePaletteCatalog(unread);
  const pinned = usePinnedPages(open, catalog.pageIndex);
  const home = usePaletteHome({ catalog, pinned });

  const commandText = askSubject === null && scope === "all" ? commandModeQuery(query) : null;
  const effectiveScope: PaletteScope = commandText !== null ? "commands" : scope;
  const searchQuery = commandText ?? (mention.open ? stripMentionToken(query) : query);
  const trimmedQuery = searchQuery.trim();
  const question =
    askSubject !== null
      ? null
      : effectiveScope === "commands" || effectiveScope === "pages"
        ? null
        : askQuestion(searchQuery);

  const recordQuery = question !== null ? questionSearchText(question) : searchQuery;
  const debouncedRecordQuery = useDebounce(recordQuery, SEARCH_DEBOUNCE_MS);
  const remote = usePaletteRemoteSearch({
    enabled: open && askSubject === null,
    query: debouncedRecordQuery,
    scope: effectiveScope,
  });

  const getPageContext = usePageContext();
  const { turn, ask, stop: stopAsk, reset: resetAsk, keep: keepAsk } = useAsk();

  const confirm = useCallback((message: string) => {
    if (confirmationTimer.current) {
      clearTimeout(confirmationTimer.current);
    }
    setConfirmation(message);
    confirmationTimer.current = setTimeout(() => setConfirmation(null), CONFIRMATION_MS);
  }, []);

  useEffect(
    () => () => {
      if (confirmationTimer.current) {
        clearTimeout(confirmationTimer.current);
      }
    },
    [],
  );

  const clearAnswer = useCallback(() => {
    setAsked(null);
    setPeek(null);
    resetAsk();
  }, [resetAsk]);

  const reset = useCallback(() => {
    setQuery("");
    setScope("all");
    setMention(CLOSED_MENTION);
    setActionsOpen(false);
    setSelected("");
    setAskSubject(null);
    setAsked(null);
    setPeek(null);
    resetAsk();
  }, [resetAsk]);

  const close = useCallback(() => {
    setOpen(false);
    reset();
  }, [reset, setOpen]);

  const focusInput = useCallback(() => {
    requestAnimationFrame(() => inputRef.current?.focus());
  }, []);

  const askAbout = useCallback(
    (subject: AssistantEntityRef) => {
      setAskSubject(subject);
      setActionsOpen(false);
      setScope("all");
      setQuery("");
      setSelected("");
      setMention(CLOSED_MENTION);
      clearAnswer();
      focusInput();
    },
    [clearAnswer, focusInput],
  );

  const runIntent = usePaletteIntentRunner({ close, askAbout, confirm });

  const actionContext = useMemo(
    () => ({ ...catalog.actionContext, pinnedUrls: pinned.pinnedUrls }),
    [catalog.actionContext, pinned.pinnedUrls],
  );

  const ranking = useMemo(
    () => rankLocal(searchQuery, catalog.pages, catalog.commands, t),
    [catalog.commands, catalog.pages, searchQuery, t],
  );

  const answerVisible =
    turn !== null &&
    asked !== null &&
    sameSubject(asked.subject, askSubject) &&
    (askSubject !== null
      ? query.trim() === "" || query.trim() === asked.question
      : question === asked.question);

  const { sections, empty } = useMemo((): { sections: PaletteSection[]; empty: boolean } => {
    if (askSubject !== null) {
      return {
        sections: buildSubjectSections(
          askSubject,
          query,
          answerVisible && asked ? asked.question : null,
          t,
        ),
        empty: false,
      };
    }
    if (trimmedQuery === "" && effectiveScope === "all") {
      return { sections: buildHomeSections(home, t), empty: false };
    }
    const results = buildSearchSections(
      {
        query: searchQuery,
        recordQuery,
        scope: effectiveScope,
        question,
        remote,
        ranking,
        recentRecords: home.recentRecords,
      },
      t,
    );
    return answerVisible
      ? { sections: results.sections.filter((entry) => entry.id !== "nova"), empty: false }
      : results;
  }, [
    answerVisible,
    askSubject,
    asked,
    effectiveScope,
    home,
    query,
    question,
    ranking,
    recordQuery,
    remote,
    searchQuery,
    t,
    trimmedQuery,
  ]);

  const counts = useMemo(
    () =>
      askSubject === null && trimmedQuery !== ""
        ? scopeCounts(ranking, remote, commandText !== null ? "commands" : "search")
        : null,
    [askSubject, commandText, ranking, remote, trimmedQuery],
  );

  const items = useMemo(() => flattenSections(sections), [sections]);
  const itemsByKey = useMemo(() => new Map(items.map((item) => [item.key, item])), [items]);
  const selectedKey = itemsByKey.has(selected) ? selected : (items[0]?.key ?? "");
  const selectedItem = itemsByKey.get(selectedKey) ?? null;
  const selectedActions = useMemo(
    () => (selectedItem ? buildItemActions(selectedItem, actionContext, t) : []),
    [actionContext, selectedItem, t],
  );

  const settledKey = useDebounce(selectedKey, PREVIEW_SETTLE_MS);
  const settledItem = itemsByKey.get(settledKey) ?? (settledKey === "" ? null : selectedItem);
  const previewItem = useMemo((): PaletteItem | null => {
    const peekRecord = peek ? recordFromRef(peek) : null;
    return peekRecord
      ? {
          kind: "record",
          key: `peek:${peekRecord.entityType}:${peekRecord.id}`,
          record: peekRecord,
        }
      : settledItem;
  }, [peek, settledItem]);
  const previewActions = useMemo(
    () => (previewItem ? buildItemActions(previewItem, actionContext, t) : []),
    [actionContext, previewItem, t],
  );
  const answerDone = answerVisible && turn?.status === "done";
  const sources = useMemo(
    () =>
      answerDone && items.length === 0
        ? answerSources(turn).flatMap((ref) => {
            const record = recordFromRef(ref);
            return record ? [record] : [];
          })
        : [],
    [answerDone, items.length, turn],
  );

  const firstRecord = useMemo(() => items.find((item) => item.kind === "record"), [items]);
  useEffect(() => {
    if (open && firstRecord?.kind === "record") {
      prefetchRecordPreview(queryClient, firstRecord.record);
    }
  }, [firstRecord, open, queryClient]);

  const openRecord = useCallback(
    (record: PaletteRecord) => {
      if (record.entityType === "document") {
        recordOpen(organizationId, record);
        runIntent({ type: "open-document", documentId: record.id, disposition: "view" });
        return;
      }
      if (record.href === "") {
        return;
      }
      recordOpen(organizationId, record);
      runIntent({ type: "navigate", href: record.href });
    },
    [organizationId, recordOpen, runIntent],
  );

  const openCited = useCallback(
    (ref: AssistantEntityRef) => {
      const record = recordFromRef(ref);
      if (record) {
        openRecord(record);
      } else if (isRecordEntityType(ref.type)) {
        runIntent({ type: "navigate", href: recordPath(ref.type, ref.id) });
      }
    },
    [openRecord, runIntent],
  );

  const namedRecords = useCallback(
    async (question: string): Promise<AssistantEntityRef[]> => {
      const text = questionSearchText(question);
      if (text === "" || !isRemoteSearchable(text, effectiveScope)) {
        return [];
      }
      const found = await queryClient.fetchQuery(paletteSearchQuery(text, effectiveScope));
      return recordsNamedIn(question, found.groups);
    },
    [effectiveScope, queryClient],
  );

  const startAsk = useCallback(
    (item: Extract<PaletteItem, { kind: "ask" }>) => {
      setActionsOpen(false);
      setPeek(null);
      setAsked({ question: item.question, subject: item.subject });
      // A failed read is forgotten, and drawing the answer asks again.
      void AnswerCard.preload().catch(() => undefined);
      const context = getPageContext();
      void ask(
        item.question,
        item.subject
          ? { context, mentions: [item.subject] }
          : { context, findMentions: () => namedRecords(item.question) },
      );
    },
    [ask, getPageContext, namedRecords],
  );

  const continueInDesk = useCallback(() => {
    void keepAsk().then((threadId) => {
      if (threadId !== null) {
        runIntent({ type: "navigate", href: conversationPath(threadId) });
      }
    });
  }, [keepAsk, runIntent]);

  const followUp = useCallback(() => {
    clearAnswer();
    setQuery("");
    setSelected("");
    focusInput();
  }, [clearAnswer, focusInput]);

  const runAction = useCallback(
    (item: PaletteItem, action: PaletteAction) => {
      const intent = action.intent;
      if (
        item.kind === "record" &&
        (intent.type === "navigate" ||
          intent.type === "new-tab" ||
          intent.type === "open-document") &&
        action.id !== "open-customer" &&
        action.id !== "open-current-load"
      ) {
        recordOpen(organizationId, item.record);
      }
      if (intentKeepsPaletteOpen(intent)) {
        setActionsOpen(false);
        focusInput();
      }
      runIntent(intent);
    },
    [focusInput, organizationId, recordOpen, runIntent],
  );

  const changeScope = useCallback(
    (next: PaletteScope) => {
      setScope(next);
      setSelected("");
      setActionsOpen(false);
      focusInput();
    },
    [focusInput],
  );

  const handleSelect = useCallback(
    (item: PaletteItem) => {
      if (item.kind === "ask") {
        startAsk(item);
        return;
      }
      if (item.kind === "widen") {
        changeScope("all");
        return;
      }
      const primary = buildItemActions(item, actionContext, t)[0];
      if (primary) {
        runAction(item, primary);
      }
    },
    [actionContext, changeScope, runAction, startAsk, t],
  );

  const previewContext = useMemo(
    (): PreviewContext => ({
      onRun: (intent: PaletteIntent) => {
        if (previewItem) {
          const action = previewActions.find((candidate) => candidate.intent === intent);
          if (action) {
            runAction(previewItem, action);
            return;
          }
        }
        runIntent(intent);
      },
      onNavigate: (href: string) => runIntent({ type: "navigate", href }),
      onOpenRecord: openRecord,
      onMore: () => {
        if (selectedActions.length > 1) {
          setActionsOpen(true);
        }
      },
      recentRecords: home.recentRecords,
      relatedCommands: actionContext.relatedCommands,
      pinnedUrls: pinned.pinnedUrls,
    }),
    [
      actionContext.relatedCommands,
      home.recentRecords,
      openRecord,
      pinned.pinnedUrls,
      previewActions,
      previewItem,
      runAction,
      runIntent,
      selectedActions.length,
    ],
  );

  const mentionOptions = useMemo(() => filterMentionOptions(mention.text), [mention.text]);

  const pickMention = useCallback(
    (option: SearchEntityOption) => {
      setScope(option.key);
      setQuery((current) => stripMentionToken(current));
      setMention(CLOSED_MENTION);
      setSelected("");
      focusInput();
    },
    [focusInput],
  );

  const handleQueryChange = useCallback(
    (value: string) => {
      setActionsOpen(false);
      setPeek(null);
      if (asked !== null) {
        clearAnswer();
      }
      if (askSubject !== null) {
        setQuery(value);
        return;
      }
      const committed = value.match(/(?:^|\s)@([a-zA-Z]+)\s$/);
      if (committed) {
        const entity = resolveMentionCommit(committed[1] ?? "");
        if (entity) {
          setScope(entity);
          setQuery(stripMentionToken(value));
          setMention(CLOSED_MENTION);
          return;
        }
      }
      const state = value.startsWith(">") ? null : getMentionState(value);
      setMention(
        state && state.mentionOpen
          ? { open: true, text: state.mentionText, index: 0 }
          : CLOSED_MENTION,
      );
      setQuery(value);
    },
    [askSubject, asked, clearAnswer],
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

  const openActions = useCallback(() => {
    if (selectedItem && selectedActions.length > 1) {
      setMention(CLOSED_MENTION);
      setActionsOpen(true);
    }
  }, [selectedActions.length, selectedItem]);

  const closeActions = useCallback(() => {
    setActionsOpen(false);
    focusInput();
  }, [focusInput]);

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
        const option = mentionOptions[Math.min(mention.index, mentionOptions.length - 1)];
        if (option) {
          event.preventDefault();
          pickMention(option);
        }
        return;
      }
    }

    if (event.key === "Tab") {
      event.preventDefault();
      if (askSubject === null) {
        changeScope(nextScope(scope, event.shiftKey ? -1 : 1));
      }
      return;
    }

    if (mod && key === "k") {
      event.preventDefault();
      if (actionsOpen) {
        closeActions();
      } else {
        openActions();
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

    if (mod && event.key === "Enter") {
      if (answerDone) {
        event.preventDefault();
        continueInDesk();
        return;
      }
      if (runShortcut(event, ACTION_NEW_TAB)) {
        return;
      }
    }
    if (mod && !event.shiftKey && key === "i" && runShortcut(event, ACTION_ASK)) {
      return;
    }
    if (mod && !event.shiftKey && key === "l" && runShortcut(event, ACTION_COPY_LINK)) {
      return;
    }
    if (mod && !event.shiftKey && key === "p" && runShortcut(event, ACTION_PIN)) {
      return;
    }
    if (event.altKey && !mod && event.code === "KeyC" && runShortcut(event, ACTION_COPY_ID)) {
      return;
    }

    if (event.key === "Backspace" && query === "") {
      if (askSubject !== null) {
        event.preventDefault();
        setAskSubject(null);
        clearAnswer();
      } else if (scope !== "all") {
        event.preventDefault();
        changeScope("all");
      }
    }
  };

  /** Esc steps back one layer at a time, and closes only when there is nothing left to undo. */
  const stepBack = (): boolean => {
    if (mention.open) {
      setMention(CLOSED_MENTION);
      return true;
    }
    if (actionsOpen) {
      closeActions();
      return true;
    }
    if (answerVisible) {
      clearAnswer();
      return true;
    }
    if (askSubject !== null) {
      setAskSubject(null);
      setQuery("");
      clearAnswer();
      return true;
    }
    if (query !== "") {
      handleQueryChange("");
      return true;
    }
    if (scope !== "all") {
      changeScope("all");
      return true;
    }
    return false;
  };

  const mod = modKey(actionContext);
  const primaryLabel =
    selectedItem?.kind === "ask"
      ? t("Ask")
      : selectedItem?.kind === "command"
        ? t("Run")
        : t("Open");
  const hints: FooterHint[] = actionsOpen
    ? [
        { keys: ["↑", "↓"], label: t("Choose") },
        { keys: ["↵"], label: t("Run") },
        { keys: ["Esc"], label: t("Back") },
      ]
    : [
        { keys: ["↑", "↓"], label: t("Navigate") },
        ...(selectedItem ? [{ keys: ["↵"], label: primaryLabel }] : []),
        ...(answerDone
          ? [{ keys: [mod, "↵"], label: t("Continue in Desk") }]
          : findAction(selectedActions, ACTION_NEW_TAB)
            ? [{ keys: [mod, "↵"], label: t("New tab") }]
            : []),
        ...(selectedActions.length > 1 ? [{ keys: [mod, "K"], label: t("Actions") }] : []),
        ...(askSubject === null ? [{ keys: ["Tab"], label: t("Scope") }] : []),
      ];
  const aside: FooterAside =
    askSubject !== null
      ? { kind: "subject", title: askSubject.label }
      : question !== null
        ? { kind: "question" }
        : effectiveScope === "commands" || effectiveScope === "pages"
          ? null
          : { kind: "ask" };

  const scopeLabel = scope !== "all" ? scopePresentation(scope, t).label : null;
  const showEmpty = empty && !answerVisible;
  const actionTile = selectedItem ? itemTile(selectedItem) : {};

  const head = (
    <>
      {answerVisible && turn && asked && (
        <Suspense fallback={null}>
          <AnswerCard.Component
            question={asked.question}
            turn={turn}
            subject={asked.subject?.label ?? null}
            onAsk={() => startAsk({ kind: "ask", key: "retry", ...asked })}
            onStop={stopAsk}
            onContinue={continueInDesk}
            onCopy={() => runIntent({ type: "copy", text: plainAnswer(turn), what: "answer" })}
            onFollowUp={followUp}
            onPeek={setPeek}
            onOpenRecord={openCited}
            mac={actionContext.mac}
          />
        </Suspense>
      )}
      {showEmpty && (
        <div className="px-2.5 pt-6 pb-3">
          <p className="text-foreground text-base font-medium">
            {scopeLabel
              ? t("Nothing called “{0}” in {1}", trimmedQuery, scopeLabel.toLowerCase())
              : t("Nothing called “{0}”", trimmedQuery)}
          </p>
          <p className="text-foreground-subtle mt-1 max-w-md text-sm">
            {scope !== "all"
              ? t(
                  "Try everything instead, or ask Nova — it reads records the search index doesn't, like stop notes and email.",
                )
              : t("Check the spelling, try a PRO or BOL number, or ask Nova.")}
          </p>
        </div>
      )}
    </>
  );

  return (
    <CommandDialog
      open={open}
      onOpenChange={(next, eventDetails) => {
        if (!next && eventDetails.reason === "escape-key" && stepBack()) {
          eventDetails.cancel();
          return;
        }
        if (next) {
          setOpen(true);
        } else {
          close();
        }
      }}
      title={t("Command palette")}
      description={t("Search records, jump to pages, run commands and ask Nova")}
      showCloseButton={false}
      className="cp-dialog bg-card rounded-surface ui-lift-float top-[8vh] flex h-[min(680px,calc(100dvh-12vh))] w-full max-w-[min(1080px,calc(100%-2rem))] translate-y-0 flex-col gap-0 overflow-hidden p-0 sm:max-w-[min(1080px,calc(100%-2rem))]"
      commandProps={{
        shouldFilter: false,
        loop: true,
        value: selectedKey,
        onValueChange: (value) => {
          setSelected(value);
          setPeek(null);
        },
        className: "flex h-full min-h-0 flex-col rounded-none bg-transparent",
      }}
    >
      <PaletteInput
        ref={inputRef}
        value={query}
        onValueChange={handleQueryChange}
        onKeyDown={handleKeyDown}
        onClear={() => {
          handleQueryChange("");
          focusInput();
        }}
        scope={scope}
        onClearScope={() => changeScope("all")}
        commandMode={commandText !== null}
        askSubject={askSubject?.label ?? null}
        onClearAskSubject={() => {
          setAskSubject(null);
          clearAnswer();
          focusInput();
        }}
        mention={{
          open: mention.open,
          options: mentionOptions,
          index: Math.min(mention.index, Math.max(0, mentionOptions.length - 1)),
          onPick: pickMention,
        }}
      />
      {askSubject === null ? (
        <PaletteScopeTabs scope={scope} onScopeChange={changeScope} counts={counts} />
      ) : (
        <div aria-hidden className="border-border-subtle border-b" />
      )}
      <div className="relative grid min-h-0 flex-1 grid-cols-1 min-[780px]:grid-cols-[minmax(0,1fr)_340px] min-[900px]:grid-cols-[minmax(0,1fr)_420px]">
        <div className="flex min-h-0 flex-col">
          <PaletteList
            sections={sections}
            selectedKey={selectedKey}
            onSelect={handleSelect}
            onRetry={remote.retry}
            onSeeAll={changeScope}
            head={head}
          />
        </div>
        <aside
          aria-label={t("Preview")}
          className="border-border-subtle bg-card hidden min-h-0 border-l min-[780px]:block"
        >
          <PalettePreview
            item={previewItem}
            sources={sources}
            actions={previewActions}
            context={previewContext}
            mac={actionContext.mac}
          />
        </aside>
        {actionsOpen && selectedItem && (
          <PaletteActionsMenu
            title={itemTitle(selectedItem)}
            icon={actionTile.icon}
            initials={actionTile.initials}
            actions={selectedActions}
            onRun={(action) => runAction(selectedItem, action)}
            onClose={closeActions}
          />
        )}
      </div>
      <PaletteFooter hints={hints} aside={aside} confirmation={confirmation} />
    </CommandDialog>
  );
}
