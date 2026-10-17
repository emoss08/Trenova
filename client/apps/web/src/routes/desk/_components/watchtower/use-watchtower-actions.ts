import { handleMutationError } from "@/hooks/use-api-mutation";
import { conversationPath } from "@/lib/conversation-path";
import {
  fetchWatchtowerItem,
  handOffWatchtowerItem,
  markWatchtowerItemSeen,
  resolveWatchtowerItem,
  snoozeWatchtowerItem,
  type WatchtowerItem,
} from "@/lib/graphql/watchtower";
import { isEditableTarget, keymapFromConfig, normalizedKey } from "@/lib/keymap";
import { attention } from "@/lib/queries/attention";
import { assistant } from "@/lib/queries/assistant";
import { watchtower } from "@/lib/queries/watchtower";
import { tomorrowMorning } from "@/lib/snooze-presets";
import {
  patchQueueItem,
  placeInQueue,
  removeFromQueue,
  shiftCleared,
  shiftQueueCounts,
} from "@/lib/watchtower-cache";
import { apiService } from "@/services/api";
import { useAssistantStore } from "@/stores/assistant-store";
import { useDeskAttachedPageStore } from "@/stores/desk-attached-page-store";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useLatestCallback } from "@trenova/shared/hooks/use-latest-callback";
import { useT } from "@trenova/shared/i18n/use-t";
import { GraphQLRequestError } from "@trenova/shared/lib/graphql";
import { newIdempotencyKey } from "@trenova/shared/lib/idempotency";
import { useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router";
import { useQueueCursor, useQueueLeave } from "../queue/use-queue-cursor";
import { watchtowerPageContext } from "./page-context";

/** What a key or a button asks the queue to do. */
export type WatchtowerCommand = "next" | "previous" | "handOff" | "done" | "snooze";

const KEYMAP = keymapFromConfig<WatchtowerCommand>(
  "watchtower",
  {
    "watchtower-next": "next",
    "watchtower-previous": "previous",
    "watchtower-hand-off": "handOff",
    "watchtower-done": "done",
    "watchtower-snooze": "snooze",
  },
  { ArrowDown: "next", ArrowUp: "previous" },
);

/** What a key press does, or null when the queue leaves it alone. */
export function resolveWatchtowerKey(press: {
  key: string;
  metaKey: boolean;
  ctrlKey: boolean;
  altKey: boolean;
  editable: boolean;
}): WatchtowerCommand | null {
  if (press.metaKey || press.ctrlKey || press.altKey || press.editable) {
    return null;
  }
  return KEYMAP.get(normalizedKey(press.key)) ?? null;
}

type Clear = "done" | "snooze";

export type WatchtowerActionsOptions = {
  items: readonly WatchtowerItem[];
  now: number;
  timezone: string;
  /** The start of the reader's day, which "cleared today" counts from. */
  since: number;
  /** Whether the keys are live: off while a dialog or menu owns the keyboard. */
  keysEnabled: boolean;
};

/**
 * Everything the Watchtower's buttons and keys do, in one place, so a key and
 * the button it mirrors can never do two different things.
 *
 * Done and Snooze take the card out the way the design does — it leaves, the
 * next one takes its place — and only then send the write, so the queue never
 * moves under an animation. The row leaves the cached queue and the counts
 * move at once; the person watched it happen, so a write that goes through
 * says nothing more. One that fails puts both back. Hand off assigns the agent
 * at once, and the row redraws from what the server answers.
 */
export function useWatchtowerActions({
  items,
  now,
  timezone,
  since,
  keysEnabled,
}: WatchtowerActionsOptions) {
  const t = useT();
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const cursor = useQueueCursor(items);
  const { current } = cursor;
  const { leaving, leave } = useQueueLeave();
  const [menuOpen, setMenuOpen] = useState(false);
  const [handingOff, setHandingOff] = useState(false);
  const [asking, setAsking] = useState(false);
  const pending = useRef(new Set<string>());

  const agentsQuery = useQuery(assistant.myAgents());
  const lastAgentId = useAssistantStore((state) => state.lastAgentId);
  const askAgent =
    agentsQuery.data?.find((agent) => agent.id === lastAgentId) ?? agentsQuery.data?.[0] ?? null;
  const attachPage = useDeskAttachedPageStore((state) => state.attach);

  const settle = () => {
    void queryClient.invalidateQueries({
      queryKey: watchtower.counts().queryKey,
      refetchType: "active",
    });
    void queryClient.invalidateQueries({
      queryKey: watchtower.cleared(since).queryKey,
      refetchType: "active",
    });
    void queryClient.invalidateQueries({ queryKey: attention._def, refetchType: "active" });
  };

  // Opening an item marks it seen for this reader. The dot goes at once; a
  // mark that does not reach the server puts it back.
  const currentId = current?.id ?? null;
  const currentSeen = current?.seen ?? true;
  useEffect(() => {
    if (currentId === null || currentSeen) {
      return;
    }
    patchQueueItem(queryClient, currentId, (item) => ({ ...item, seen: true }));
    markWatchtowerItemSeen(currentId).catch(() => {
      patchQueueItem(queryClient, currentId, (item) => ({ ...item, seen: false }));
    });
  }, [currentId, currentSeen, queryClient]);

  // A refused write puts the row back as the server now has it, so the card
  // the person sees again is the item as it stands and a second press is made
  // against it rather than against the copy the first one was refused for.
  // An item that changed underneath (a source refreshing it) is simply shown
  // again; anything else is an error the person is told about.
  const putBack = (item: WatchtowerItem) => {
    shiftQueueCounts(queryClient, item, 1);
    shiftCleared(queryClient, since, -1);
    cursor.select(item.id);
    return fetchWatchtowerItem(item.id).then(
      (fresh) => placeInQueue(queryClient, { ...fresh, seen: true }),
      () => placeInQueue(queryClient, item),
    );
  };

  const clear = (how: Clear) => {
    const item = current;
    if (item === null || pending.current.has(item.id)) {
      return;
    }
    pending.current.add(item.id);
    setMenuOpen(false);
    const nextId = cursor.after([item.id]);
    const until = tomorrowMorning(now, timezone);

    leave(how === "done" ? "ok" : "no", () => {
      removeFromQueue(queryClient, item.id);
      shiftQueueCounts(queryClient, item, -1);
      shiftCleared(queryClient, since, 1);
      cursor.select(nextId);

      const key = newIdempotencyKey();
      const request =
        how === "done"
          ? resolveWatchtowerItem(
              { id: item.id, version: item.version, code: "Fixed" },
              { idempotencyKey: key },
            )
          : snoozeWatchtowerItem(
              { id: item.id, version: item.version, until },
              { idempotencyKey: key },
            );
      request
        .catch((error: unknown) => {
          void putBack(item);
          if (!(error instanceof GraphQLRequestError && error.isVersionMismatchError())) {
            handleMutationError({ error, resourceName: "Watchtower item" });
          }
        })
        .finally(() => {
          pending.current.delete(item.id);
          settle();
        });
    });
  };

  const handOff = (agentId?: string) => {
    const item = current;
    if (item === null || item.agentWork !== null || item.subjectId === null || handingOff) {
      return;
    }
    const chosen = agentId ?? item.suggestedAgentId ?? null;
    if (chosen === null) {
      setMenuOpen(true);
      return;
    }
    setMenuOpen(false);
    setHandingOff(true);
    handOffWatchtowerItem(item.id, chosen, { idempotencyKey: newIdempotencyKey() })
      .then(
        (result) => {
          placeInQueue(queryClient, { ...result.item, seen: true });
        },
        (error: unknown) => handleMutationError({ error, resourceName: "Watchtower item" }),
      )
      .finally(() => setHandingOff(false));
  };

  const ask = () => {
    const item = current;
    if (item === null || askAgent === null || asking) {
      return;
    }
    setAsking(true);
    apiService.assistantService
      .startThread(askAgent.id, {
        origin: "Watchtower",
        title: item.title,
        subjectType: item.subjectType ?? undefined,
        subjectId: item.subjectId ?? undefined,
      })
      .then(
        (thread) => {
          attachPage(thread.id, watchtowerPageContext(t, item, now, timezone));
          useAssistantStore.getState().setLastAgentId(askAgent.id);
          void queryClient.invalidateQueries({ queryKey: assistant.threads().queryKey });
          void navigate(conversationPath(thread.id));
        },
        (error: unknown) => handleMutationError({ error, resourceName: "Conversation" }),
      )
      .finally(() => setAsking(false));
  };

  const run = (command: WatchtowerCommand) => {
    switch (command) {
      case "next":
        cursor.go(1);
        return;
      case "previous":
        cursor.go(-1);
        return;
      case "handOff":
        handOff();
        return;
      case "done":
        clear("done");
        return;
      case "snooze":
        clear("snooze");
        return;
    }
  };

  const dispatch = useLatestCallback(run);
  const listening = keysEnabled && !menuOpen;
  useEffect(() => {
    if (!listening) {
      return;
    }
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.defaultPrevented) {
        return;
      }
      const command = resolveWatchtowerKey({
        key: event.key,
        metaKey: event.metaKey,
        ctrlKey: event.ctrlKey,
        altKey: event.altKey,
        editable: isEditableTarget(event.target),
      });
      if (command === null) {
        return;
      }
      event.preventDefault();
      dispatch(command);
    };
    document.addEventListener("keydown", onKeyDown);
    return () => document.removeEventListener("keydown", onKeyDown);
  }, [dispatch, listening]);

  return {
    current,
    index: cursor.index,
    select: cursor.select,
    leaving,
    menuOpen,
    setMenuOpen,
    handingOff,
    canAsk: askAgent !== null && !asking,
    next: () => run("next"),
    previous: () => run("previous"),
    done: () => run("done"),
    snooze: () => run("snooze"),
    handOff,
    ask,
    reviewAndApprove: () => {
      const runId = current?.agentWork?.runId;
      if (runId) {
        void navigate(`/desk/decisions?run=${encodeURIComponent(runId)}`);
      }
    },
  };
}
