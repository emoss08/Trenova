import { isScheduleRequest } from "@/components/assistant/composer-commands";
import { assistant } from "@/lib/queries/assistant";
import { apiService } from "@/services/api";
import type { ConversationSchedule, ConversationScheduleList } from "@/types/assistant";
import {
  appendToHistory,
  continuesHistory,
  type ThreadHistory,
} from "@/components/assistant/thread-history";
import {
  useMutation,
  useMutationState,
  useQueryClient,
  type InfiniteData,
  type QueryClient,
} from "@tanstack/react-query";
import type { StartedTurn } from "@/services/assistant";
import { useEffect, useState } from "react";
import { handleMutationError } from "@/hooks/use-api-mutation";

export { isScheduleRequest };

/**
 * A schedule as the server answered a change to it, over the one already
 * shown. A pause, resume or run is answered without the conversation's
 * title, which has not changed, so the title already read is kept.
 */
export function mergeSchedule(
  current: ConversationSchedule,
  next: ConversationSchedule,
): ConversationSchedule {
  return next.threadTitle === "" && current.threadTitle !== ""
    ? { ...next, threadTitle: current.threadTitle }
    : next;
}

/** Writes one schedule into a conversation's cached list, or removes it. */
export function patchScheduleList(
  list: ConversationScheduleList | undefined,
  id: string,
  next: ConversationSchedule | null,
): ConversationScheduleList | undefined {
  if (!list) {
    return list;
  }
  const exists = list.items.some((item) => item.id === id);
  if (next === null) {
    return exists
      ? { items: list.items.filter((item) => item.id !== id), total: Math.max(0, list.total - 1) }
      : list;
  }
  if (!exists) {
    return { items: [next, ...list.items], total: list.total + 1 };
  }

  return {
    items: list.items.map((item) => (item.id === id ? mergeSchedule(item, next) : item)),
    total: list.total,
  };
}

/** The pages of every schedule the person keeps, as the Scheduled page reads them. */
export type SchedulePages = InfiniteData<ConversationScheduleList, number>;

/**
 * Writes one schedule into the pages the Scheduled page has read, or removes
 * it. A schedule not among them is left for the next read to bring in, since
 * where it belongs depends on pages not yet read; a removal lowers every
 * page's total, so the count of what is left to read stays right.
 */
export function patchSchedulePages(
  data: SchedulePages | undefined,
  id: string,
  next: ConversationSchedule | null,
): SchedulePages | undefined {
  if (!data || !data.pages.some((page) => page.items.some((item) => item.id === id))) {
    return data;
  }

  return {
    ...data,
    pages: data.pages.map((page) =>
      next === null
        ? {
            items: page.items.filter((item) => item.id !== id),
            total: Math.max(0, page.total - 1),
          }
        : {
            items: page.items.map((item) => (item.id === id ? mergeSchedule(item, next) : item)),
            total: page.total,
          },
    ),
  };
}

/**
 * Schedules in the order the Scheduled page lists them, the order the server
 * reads them in: the ones that will fire first, soonest first and one with
 * no next slot last among them, then the paused ones the same way. Sorting
 * what is loaded again moves a schedule paused or resumed here at once,
 * rather than when the list is next read.
 */
export function orderSchedules(
  schedules: readonly ConversationSchedule[],
): ConversationSchedule[] {
  return [...schedules].sort((a, b) => {
    if (a.enabled !== b.enabled) {
      return a.enabled ? -1 : 1;
    }
    const aNext = a.nextRunAt ?? null;
    const bNext = b.nextRunAt ?? null;
    if (aNext !== bNext) {
      if (aNext === null) {
        return 1;
      }
      if (bNext === null) {
        return -1;
      }
      return aNext - bNext;
    }
    if (a.createdAt !== b.createdAt) {
      return b.createdAt - a.createdAt;
    }

    return a.id < b.id ? 1 : a.id > b.id ? -1 : 0;
  });
}

/** The caches one schedule is drawn from: its conversation's cards and the Scheduled page. */
type ScheduleCaches = {
  thread: ConversationScheduleList | undefined;
  all: SchedulePages | undefined;
};

function scheduleKeys(threadId: string) {
  return {
    thread: assistant.schedules(threadId).queryKey,
    all: assistant.allSchedules().queryKey,
  };
}

function readSchedules(queryClient: QueryClient, threadId: string): ScheduleCaches {
  const keys = scheduleKeys(threadId);

  return {
    thread: queryClient.getQueryData<ConversationScheduleList>(keys.thread),
    all: queryClient.getQueryData<SchedulePages>(keys.all),
  };
}

function restoreSchedules(queryClient: QueryClient, threadId: string, caches: ScheduleCaches) {
  const keys = scheduleKeys(threadId);
  queryClient.setQueryData(keys.thread, caches.thread);
  queryClient.setQueryData(keys.all, caches.all);
}

function writeSchedule(
  queryClient: QueryClient,
  threadId: string,
  id: string,
  next: ConversationSchedule | null,
) {
  const keys = scheduleKeys(threadId);
  queryClient.setQueryData<ConversationScheduleList>(keys.thread, (list) =>
    patchScheduleList(list, id, next),
  );
  queryClient.setQueryData<SchedulePages>(keys.all, (pages) =>
    patchSchedulePages(pages, id, next),
  );
}

/**
 * Writes a schedule as the server saved it over the one shown, in its
 * conversation's cards and on the Scheduled page.
 */
export function storeSchedule(
  queryClient: QueryClient,
  shown: ConversationSchedule,
  saved: ConversationSchedule,
) {
  writeSchedule(queryClient, saved.threadId, saved.id, mergeSchedule(shown, saved));
}

/**
 * Settles one schedule's caches before it is changed in them, so a read
 * already under way cannot land over the change, and keeps what they held to
 * put back if the server refuses it.
 */
async function holdSchedules(
  queryClient: QueryClient,
  schedule: ConversationSchedule,
): Promise<ScheduleCaches> {
  const keys = scheduleKeys(schedule.threadId);
  await Promise.all([
    queryClient.cancelQueries({ queryKey: keys.thread }),
    queryClient.cancelQueries({ queryKey: keys.all }),
  ]);

  return readSchedules(queryClient, schedule.threadId);
}

/** Marks the changes made to a schedule, so a row can say which of them is under way. */
const SCHEDULE_CHANGE_KEY = ["assistant-schedule-change"] as const;

/**
 * Pausing, resuming, deleting and running one of the person's schedules,
 * from its card in a conversation or its row on the Scheduled page. A pause,
 * resume or delete shows at once in both places and is put back if the
 * server refuses it. `onRan` hears of a run that started, with the
 * conversation it is answering in.
 */
export function useScheduleActions(options?: {
  onRan?: (schedule: ConversationSchedule, turn: StartedTurn) => void;
}) {
  const queryClient = useQueryClient();
  const onRan = options?.onRan;

  const toggle = useMutation({
    mutationKey: SCHEDULE_CHANGE_KEY,
    mutationFn: (schedule: ConversationSchedule) =>
      apiService.assistantService.setScheduleEnabled(schedule.id, !schedule.enabled),
    onMutate: async (schedule) => {
      const previous = await holdSchedules(queryClient, schedule);
      writeSchedule(queryClient, schedule.threadId, schedule.id, {
        ...schedule,
        enabled: !schedule.enabled,
      });
      return { previous };
    },
    onSuccess: (saved, schedule) =>
      writeSchedule(queryClient, schedule.threadId, saved.id, mergeSchedule(schedule, saved)),
    onError: (error, schedule, context) => {
      if (context) {
        restoreSchedules(queryClient, schedule.threadId, context.previous);
      }
      handleMutationError({ error, resourceName: "Schedule" });
    },
  });

  const remove = useMutation({
    mutationKey: SCHEDULE_CHANGE_KEY,
    mutationFn: (schedule: ConversationSchedule) =>
      apiService.assistantService.deleteSchedule(schedule.id),
    onMutate: async (schedule) => {
      const previous = await holdSchedules(queryClient, schedule);
      writeSchedule(queryClient, schedule.threadId, schedule.id, null);
      return { previous };
    },
    onError: (error, schedule, context) => {
      if (context) {
        restoreSchedules(queryClient, schedule.threadId, context.previous);
      }
      handleMutationError({ error, resourceName: "Schedule" });
    },
  });

  const run = useMutation({
    mutationKey: SCHEDULE_CHANGE_KEY,
    mutationFn: (schedule: ConversationSchedule) =>
      apiService.assistantService.runSchedule(schedule.id),
    onSuccess: (turn, schedule) => {
      void queryClient.invalidateQueries({ queryKey: assistant.activeTurns().queryKey });
      onRan?.(schedule, turn);
    },
    onError: (error) => handleMutationError({ error, resourceName: "Schedule" }),
  });

  const changing = useMutationState({
    filters: { mutationKey: SCHEDULE_CHANGE_KEY, status: "pending" },
    select: (mutation) => (mutation.state.variables as ConversationSchedule | undefined)?.id,
  });

  return {
    toggle: (schedule: ConversationSchedule) => toggle.mutate(schedule),
    remove: (schedule: ConversationSchedule) => remove.mutate(schedule),
    run: (schedule: ConversationSchedule) => run.mutate(schedule),
    /** Whether a change to any schedule is under way. */
    busy: changing.length > 0,
    /** Whether a change to this schedule is under way. */
    isChanging: (id: string) => changing.includes(id),
  };
}

/**
 * The time a schedule's labels are read against, in Unix seconds. "Today"
 * and "Tomorrow" on a next run move at midnight; a minute is close enough
 * for a label.
 */
export function useScheduleClock(): number {
  const [clock, setClock] = useState(() => Math.floor(Date.now() / 1000));
  useEffect(() => {
    const timer = window.setInterval(() => setClock(Math.floor(Date.now() / 1000)), 60_000);
    return () => window.clearInterval(timer);
  }, []);

  return clock;
}

/**
 * Scheduling a message in a conversation. The server reads the cadence and
 * answers with the schedule and the message that draws its card; both go
 * into the caches the conversation already reads, so the card appears
 * without the thread being read again. `pending` is the text while the
 * request is out, shown as the person's message in the meantime.
 */
export function useCreateSchedule(threadId: string) {
  const queryClient = useQueryClient();

  const mutation = useMutation({
    mutationFn: (content: string) => apiService.assistantService.createSchedule(threadId, content),
    onSuccess: async ({ schedule, message }) => {
      const historyKey = assistant.messages(threadId).queryKey;
      const cached = queryClient.getQueryData<ThreadHistory>(historyKey);
      if (continuesHistory(cached, [message])) {
        queryClient.setQueryData<ThreadHistory>(historyKey, (history) =>
          appendToHistory(history, [message]),
        );
      } else {
        await queryClient.invalidateQueries({ queryKey: historyKey });
      }

      const scheduleKey = assistant.schedules(threadId).queryKey;
      if (queryClient.getQueryData(scheduleKey)) {
        queryClient.setQueryData<ConversationScheduleList>(scheduleKey, (list) =>
          patchScheduleList(list, schedule.id, schedule),
        );
      } else {
        await queryClient.invalidateQueries({ queryKey: scheduleKey });
      }
      // Where a new schedule falls among the rest depends on pages the
      // Scheduled page may not have read, so it is read again.
      void queryClient.invalidateQueries({ queryKey: assistant.allSchedules().queryKey });
      // A conversation opened with a schedule is named for its request.
      void queryClient.invalidateQueries({ queryKey: assistant.thread(threadId).queryKey });
      void queryClient.invalidateQueries({ queryKey: assistant.threads().queryKey });
    },
    onError: (error) => handleMutationError({ error, resourceName: "Schedule" }),
  });

  return {
    /** Schedules the text; `onFailed` gets it back to put in the composer again. */
    create: (content: string, onFailed?: (content: string) => void) =>
      mutation.mutate(content, { onError: () => onFailed?.(content) }),
    pending: mutation.isPending ? (mutation.variables ?? null) : null,
  };
}
