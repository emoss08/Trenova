import { queries } from "@/lib/queries";
import { updateCachedThread } from "@/lib/thread-list";
import { apiService } from "@/services/api";
import type { AssistantStreamEvent, AssistantThread, ContextUsage } from "@/types/assistant";
import { useQueryClient } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { useCallback, useEffect, useReducer, useRef } from "react";
import { toast } from "sonner";
import { compactionReducer, IDLE, meterView } from "./compaction";
import { followTurn, stopTurnQuietly, turnFailureDetail } from "./follow-turn";
import { registerTurnReader } from "./turn-readers";

/**
 * A conversation's context and its compaction: how full it is, whether it
 * compacts itself, and a compaction under way, which the composer shows and
 * can cancel.
 *
 * The measure lives on the thread and moves with each turn: the turn's stream
 * says how full the conversation is as the turn is saved, and the thread's
 * cached copy is updated from it. A compaction is a turn of its own, followed
 * on its stream from the moment the person asks, the moment a turn sets one
 * off, or the moment the conversation is reopened while one runs, which the
 * reply's own rejoin finds and hands over.
 */
export function useCompaction(thread: AssistantThread, busy: boolean) {
  const t = useT();
  const queryClient = useQueryClient();
  const [state, dispatch] = useReducer(compactionReducer, IDLE);
  const usage = thread.contextUsage ?? null;
  const following = useRef<{ turnId: string; controller: AbortController } | null>(null);
  const usageRef = useRef(usage);
  useEffect(() => {
    usageRef.current = usage;
  }, [usage]);
  // A Cancel pressed before the server named the turn; the turn is stopped
  // as soon as it is known.
  const cancelledEarly = useRef(false);

  // The conversation is read from the list and, when it is not listed, on
  // its own; both copies take the new measure.
  const keepUsage = useCallback(
    (next: ContextUsage | null | undefined, autoCompactOff?: boolean) => {
      const merge = (cached: AssistantThread): AssistantThread => ({
        ...cached,
        contextUsage: next ?? cached.contextUsage,
        autoCompactOff: autoCompactOff ?? cached.autoCompactOff,
      });
      queryClient.setQueryData<AssistantThread>(
        queries.assistant.thread(thread.id).queryKey,
        (cached) => (cached ? merge(cached) : cached),
      );
      updateCachedThread(queryClient, thread.id, merge);
    },
    [queryClient, thread.id],
  );

  const settle = useCallback(async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: queries.assistant.messages(thread.id).queryKey }),
      queryClient.invalidateQueries({ queryKey: queries.assistant.thread(thread.id).queryKey }),
      queryClient.invalidateQueries({ queryKey: queries.assistant.threads().queryKey }),
      queryClient.invalidateQueries({ queryKey: queries.assistant.activeTurns().queryKey }),
    ]);
  }, [queryClient, thread.id]);

  const onEvent = useCallback(
    (event: AssistantStreamEvent) => {
      switch (event.event) {
        case "context":
          if (event.data.threadId === thread.id) {
            keepUsage(event.data.usage, event.data.autoCompactOff);
          }
          return;
        case "compaction_started": {
          // A compaction found under way when the conversation was reopened
          // comes without figures; the meter's own stand in for them.
          if (event.data.before > 0) {
            dispatch({ type: "started", event: event.data });
            return;
          }
          const view = meterView(usageRef.current);
          dispatch({
            type: "started",
            event: { ...event.data, before: view.used, after: view.used - view.frees },
          });
          return;
        }
        case "compaction_finished":
          keepUsage(event.data.usage);
          dispatch({ type: "finished", turnId: event.data.turnId });
          void settle();
          return;
        case "compaction_cancelled":
          keepUsage(null, event.data.autoCompactOff || undefined);
          dispatch({ type: "cancelled", turnId: event.data.turnId });
          void settle();
          return;
        case "error":
          dispatch({ type: "failed", message: event.data.message });
          void settle();
          return;
        case "done":
          // An ending rebuilt from the turn's record: the conversation says
          // how it came out.
          if (following.current) {
            dispatch({ type: "finished", turnId: following.current.turnId });
          }
          void settle();
          return;
        default:
      }
    },
    [keepUsage, settle, thread.id],
  );

  /** Follows a compaction's turn to its end, unless it is already being followed. */
  const follow = useCallback(
    async (turnId: string) => {
      if (following.current?.turnId === turnId) {
        return;
      }
      following.current?.controller.abort();
      const controller = new AbortController();
      following.current = { turnId, controller };
      const unregister = registerTurnReader(controller);
      try {
        await followTurn(turnId, { signal: controller.signal, onEvent });
      } catch (error) {
        if (!controller.signal.aborted) {
          dispatch({
            type: "failed",
            message: turnFailureDetail(error, t("The connection to the assistant was lost.")),
          });
          void settle();
        }
      } finally {
        unregister();
        if (following.current?.controller === controller) {
          following.current = null;
        }
      }
    },
    [onEvent, settle, t],
  );

  useEffect(() => () => following.current?.controller.abort(), []);

  // A compaction a turn set off, named on that turn's stream, is followed on
  // its own.
  useEffect(() => {
    if (state.phase === "compacting" && state.turnId !== null) {
      void follow(state.turnId);
    }
  }, [follow, state]);

  // A failure is said once, and the composer is free again.
  useEffect(() => {
    if (state.phase === "idle" && state.error) {
      toast.error(t("The conversation could not be compacted"), { description: state.error });
    }
  }, [state, t]);

  const compact = useCallback(async () => {
    if (busy || state.phase === "compacting") {
      return;
    }
    const view = meterView(usage);
    cancelledEarly.current = false;
    dispatch({ type: "request", before: view.used, after: view.used - view.frees });
    try {
      const started = await apiService.assistantService.compactThread(thread.id);
      if (cancelledEarly.current) {
        stopTurnQuietly(started.turnId);
        return;
      }
      void queryClient.invalidateQueries({ queryKey: queries.assistant.activeTurns().queryKey });
      dispatch({
        type: "started",
        event: {
          turnId: started.turnId,
          threadId: thread.id,
          auto: false,
          before: 0,
          after: 0,
          autoCompactOff: false,
        },
      });
    } catch (error) {
      dispatch({
        type: "failed",
        message: turnFailureDetail(error, t("The conversation could not be compacted.")),
      });
    }
  }, [busy, queryClient, state.phase, t, thread.id, usage]);

  const cancel = useCallback(() => {
    if (state.phase !== "compacting") {
      return;
    }
    if (state.turnId === null) {
      cancelledEarly.current = true;
    } else {
      stopTurnQuietly(state.turnId);
    }
    following.current?.controller.abort();
    following.current = null;
    // Cancelling a compaction that started on its own stops the
    // conversation compacting itself; the server does the same.
    if (state.auto) {
      keepUsage(null, true);
    }
    dispatch({ type: "cancel" });
    void settle();
  }, [keepUsage, settle, state]);

  const setAuto = useCallback(
    async (on: boolean) => {
      keepUsage(null, !on);
      try {
        await apiService.assistantService.updateThread(thread.id, { autoCompact: on });
      } catch {
        keepUsage(null, on);
        toast.error(t("The setting could not be saved"));
      }
    },
    [keepUsage, t, thread.id],
  );

  return {
    usage,
    auto: !thread.autoCompactOff,
    setAuto,
    compacting: state.phase === "compacting" ? state : null,
    compact,
    cancel,
    /** Hands the compaction what a reply's stream says about the conversation. */
    onConversationEvent: onEvent,
  };
}

export type Compaction = ReturnType<typeof useCompaction>;
