import { useClearOfLauncher } from "@/components/assistant/launcher-clearance";
import { useNowSeconds } from "@/hooks/use-now-seconds";
import { useWidthBelow } from "@/hooks/use-width-below";
import { describeApiError } from "@/lib/api-error-message";
import type { InboundMessageDetail, TriageAction } from "@/lib/graphql/inbox";
import { inbox } from "@/lib/queries/inbox";
import { useTriageStore } from "@/stores/triage-store";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Drawer,
  DrawerContent,
  DrawerDescription,
  DrawerTitle,
} from "@trenova/shared/components/ui/drawer";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useLatestCallback } from "@trenova/shared/hooks/use-latest-callback";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useCallback, useEffect, useMemo, useReducer, useRef, useState } from "react";
import { useAnnouncePresence } from "../use-triage-presence";
import { useTriageActions } from "../use-triage-actions";
import { ActionBar } from "./action-bar";
import { actionBarMode } from "./action-bar-mode";
import { ActionsPanel } from "./actions-panel";
import { ActivityTrace } from "./activity-trace";
import { FileChips } from "./file-chips";
import { MessageBody } from "./message-body";
import { MessageChips, SettledReceipt } from "./message-chips";
import { MessageHeader } from "./message-header";
import { INITIAL_PANEL, panelReducer, stepStart, type DraftKind } from "./panel-state";
import { ReaderPanelProvider, type ReaderPanel } from "./reader-panel";
import { ReaderToolbar } from "./reader-toolbar";
import { SentReplies } from "./sent-replies";

/**
 * The reader's contract with the page.
 *
 * The page owns which message is open and the folder's order; the reader owns
 * everything inside its column. It performs every action through
 * `useTriageActions()` and registers what ↵ and R do with
 * `useRegisterReaderCommands`, so the keys reach whatever the reader draws.
 */
export type ReaderProps = {
  messageId: string;
  /**
   * Where the open message sits in the folder: 1-based, how many rows are
   * loaded, and whether the folder holds more than that.
   */
  position: { index: number; total: number; more: boolean };
  /** Whether the person has stopped on this message; one passed over is shown only if cached. */
  settled: boolean;
  onClose: () => void;
  /** J/K from the toolbar's arrows: the next (1) or previous (-1) message. */
  onStep: (direction: 1 | -1) => void;
};

/**
 * Registers what the open reader does for ↵ (do the next step) and R (reply)
 * while the calling component is mounted.
 */
export function useRegisterReaderCommands(commands: {
  doNextStep?: () => void;
  reply?: () => void;
}): void {
  const { doNextStep, reply } = commands;
  useEffect(() => {
    useTriageStore.getState().setReaderCommands({ doNextStep, reply });
    return () => useTriageStore.getState().setReaderCommands({});
  }, [doNextStep, reply]);
}

/** Beside the message from this width; above its body below it. */
const RAIL_WIDTH = 1041;
/** Below this width the panel is a sheet opened from the action bar. */
const COMPACT_WIDTH = 560;
/** What the action bar shifts left by when the rail is beside the message, to sit under the words. */
const RAIL_OFFSET = "pr-89";

function ReaderSkeleton() {
  return (
    <div aria-busy="true" className="mx-auto flex w-full max-w-170 flex-col gap-4 px-8 pt-18">
      <Skeleton className="h-7 w-2/3" />
      <div className="flex items-center gap-3">
        <Skeleton className="size-6 rounded-full" />
        <Skeleton className="h-3.5 w-48" />
      </div>
      <Skeleton className="h-6 w-40 rounded-full" />
      <Skeleton className="h-28 w-full" />
    </div>
  );
}

function ReaderError({
  error,
  onRetry,
  onClose,
}: {
  error: unknown;
  onRetry: () => void;
  onClose: () => void;
}) {
  const t = useT();
  return (
    <div className="flex flex-1 flex-col items-center justify-center gap-2 p-8 text-center">
      <p className="text-base font-semibold">{t("This message could not be opened")}</p>
      <p className="text-foreground-subtle max-w-sm text-sm">{describeApiError(error)}</p>
      <div className="mt-2 flex gap-2">
        <Button size="sm" variant="outline" onClick={onClose}>
          {t("Close")}
        </Button>
        <Button size="sm" onClick={onRetry}>
          {t("Try again")}
        </Button>
      </div>
    </div>
  );
}

function stepsKey(actions: readonly TriageAction[]): string {
  return actions.map((action) => `${action.id}:${action.done ? 1 : 0}`).join("|");
}

function ReaderView({
  message,
  position,
  onClose,
  onStep,
}: Omit<ReaderProps, "messageId" | "settled"> & { message: InboundMessageDetail }) {
  const t = useT();
  const actions = useTriageActions();
  const now = useNowSeconds();
  const rootRef = useRef<HTMLElement>(null);
  const narrow = useWidthBelow(rootRef, RAIL_WIDTH);
  const compact = useWidthBelow(rootRef, COMPACT_WIDTH);
  const panelOpen = useTriageStore((store) => store.panelOpen);
  const [state, dispatch] = useReducer(panelReducer, INITIAL_PANEL);
  const [splitRequested, setSplitRequested] = useState<string | null>(null);
  const [adhocReply, setAdhocReply] = useState(false);
  const submits = useRef(new Map<string, () => void>());
  const primary = useRef<(() => void) | null>(null);
  const clearOfLauncher = useClearOfLauncher();

  const key = stepsKey(message.actions);
  const [seenSteps, setSeenSteps] = useState<string | null>(null);
  if (key !== seenSteps) {
    setSeenSteps(key);
    dispatch({ type: "stepsChanged", steps: message.actions });
  }

  const showPanel = useLatestCallback(() => {
    if (compact) {
      dispatch({ type: "sheet", open: true });
    } else if (!useTriageStore.getState().panelOpen) {
      useTriageStore.getState().setPanelOpen(true);
    }
  });

  const draft = useLatestCallback((kind: DraftKind) => {
    const step = message.actions.find((action) =>
      kind === "load" ? action.kind === "BuildLoad" : action.kind === "Reply",
    );
    if (kind === "reply" && step === undefined) {
      setAdhocReply(true);
    }
    showPanel();
    dispatch({ type: "draftStarted", draft: kind, stepId: step?.id ?? kind });
    const run = kind === "load" ? actions.draftLoad(message.id) : actions.draftReply(message.id);
    run.then(
      () => dispatch({ type: "draftSettled", draft: kind }),
      (failure: unknown) =>
        dispatch({ type: "draftSettled", draft: kind, error: describeApiError(failure) }),
    );
  });

  const start = useLatestCallback((action: TriageAction) => {
    const how = stepStart(action, {
      hasLoadDraft: message.loadDraft !== null,
      hasReplyDraft: message.replyDraft !== null,
      replyInFlight: action.reply?.status === "Failed" || action.reply?.status === "Sending",
    });
    if (how === "link") {
      useTriageStore.getState().openOverlay({ kind: "link", messageId: message.id });
      return;
    }
    showPanel();
    switch (how) {
      case "draftLoad":
        draft("load");
        return;
      case "draftReply":
        draft("reply");
        return;
      case "compose":
        dispatch({ type: "open", id: action.id });
        return;
      case "toggle":
        dispatch({ type: "toggle", id: action.id });
    }
  });

  const openFile = useLatestCallback((attachmentId: string, split: boolean) => {
    const stepId = `file:${attachmentId}`;
    if (!message.actions.some((action) => action.id === stepId)) {
      useTriageStore.getState().openOverlay({ kind: "doc", messageId: message.id, attachmentId });
      return;
    }
    setSplitRequested(split ? attachmentId : null);
    showPanel();
    dispatch({ type: "open", id: stepId });
  });

  const openReply = useLatestCallback(() => {
    const step = message.actions.find((action) => action.kind === "Reply");
    if (step === undefined) {
      setAdhocReply(true);
    }
    showPanel();
    dispatch({ type: "open", id: step?.id ?? "reply" });
  });

  const registerSubmit = useCallback((stepId: string, submit: (() => void) | null) => {
    if (submit === null) {
      submits.current.delete(stepId);
    } else {
      submits.current.set(stepId, submit);
    }
  }, []);

  const doNextStep = useLatestCallback(() => {
    const open = state.open === null ? undefined : submits.current.get(state.open);
    if (open !== undefined) {
      open();
      return;
    }
    primary.current?.();
  });

  const runNext = useLatestCallback(() => {
    const mode = actionBarMode(message, state.drafting);
    if (mode.kind !== "next" || mode.busy) {
      return;
    }
    const open = state.open === mode.action.id ? submits.current.get(mode.action.id) : undefined;
    if (open !== undefined) {
      open();
      return;
    }
    start(mode.action);
  });

  useRegisterReaderCommands({ doNextStep, reply: openReply });

  const panel = useMemo<ReaderPanel>(
    () => ({
      state,
      dispatch,
      start,
      draftReply: () => draft("reply"),
      draftLoad: () => draft("load"),
      registerSubmit,
      openFile,
      splitRequested,
      adhocReply,
      openReply,
      compact,
    }),
    [state, start, draft, registerSubmit, openFile, splitRequested, adhocReply, openReply, compact],
  );

  const openSteps = message.actions.filter((action) => !action.done).length;
  const rail = !narrow && panelOpen;
  const inline = narrow && !compact && panelOpen;

  return (
    <ReaderPanelProvider value={panel}>
      <article
        ref={rootRef}
        aria-label={message.subject === "" ? t("(no subject)") : message.subject}
        className="@container/reader relative flex min-h-0 flex-1 flex-col"
      >
        <ReaderToolbar
          message={message}
          position={position}
          onClose={onClose}
          onStep={onStep}
          openSteps={openSteps}
          panelShown={compact ? state.sheet : panelOpen}
          onTogglePanel={() =>
            compact
              ? dispatch({ type: "sheet", open: !state.sheet })
              : useTriageStore.getState().togglePanel()
          }
          onReply={openReply}
        />
        <div className="scrollbar-overlay min-h-0 flex-1 overflow-y-auto">
          <div
            className={cn(
              "animate-rise px-5 pt-6 pb-36 sm:px-8",
              rail
                ? "grid grid-cols-[minmax(0,640px)_300px] justify-center gap-x-14"
                : cn("mx-auto flex flex-col", inline ? "max-w-186" : "max-w-170"),
            )}
          >
            <div className="flex min-w-0 flex-col gap-5">
              <MessageHeader message={message} now={now} />
              <MessageChips message={message} />
              {inline && <ActionsPanel message={message} placement="inline" />}
              <SettledReceipt message={message} />
              <MessageBody message={message} />
              <FileChips message={message} />
              <SentReplies message={message} />
              <ActivityTrace activity={message.activity} now={now} />
            </div>
            {rail && <ActionsPanel message={message} placement="rail" />}
          </div>
        </div>
        <div
          ref={clearOfLauncher}
          className={cn(
            "pointer-events-none absolute inset-x-0 bottom-4.5 z-20 flex justify-center px-4 transition-[padding] duration-300 ease-settle",
            rail && RAIL_OFFSET,
          )}
        >
          <ActionBar message={message} onPrimary={runNext} primaryRef={primary} />
        </div>
        {compact && (
          <Drawer open={state.sheet} onOpenChange={(open) => dispatch({ type: "sheet", open })}>
            <DrawerContent className="bg-raised max-h-[78%] rounded-t-2xl">
              <DrawerTitle className="sr-only">{t("Actions and information")}</DrawerTitle>
              <DrawerDescription className="sr-only">
                {t("The steps this message asks for, and what it tells you")}
              </DrawerDescription>
              <div className="overflow-y-auto px-4 pt-3 pb-6">
                <ActionsPanel message={message} placement="sheet" />
              </div>
            </DrawerContent>
          </Drawer>
        )}
      </article>
    </ReaderPanelProvider>
  );
}

/**
 * One message, open: its subject and thread, who sent it and what Nova read
 * it as, the words with what Nova read underlined, its files, the replies
 * that went out and what happened to it — beside the steps it asks for, each
 * finished in place, and under a bar that does the next one.
 */
export function Reader({ messageId, position, settled, onClose, onStep }: ReaderProps) {
  const query = useQuery({ ...inbox.message(messageId), enabled: settled });
  useAnnouncePresence(messageId, "viewing");
  const message = query.data;

  if (message === undefined) {
    return query.isError ? (
      <ReaderError error={query.error} onRetry={() => void query.refetch()} onClose={onClose} />
    ) : (
      <ReaderSkeleton />
    );
  }

  return <ReaderView message={message} position={position} onClose={onClose} onStep={onStep} />;
}
