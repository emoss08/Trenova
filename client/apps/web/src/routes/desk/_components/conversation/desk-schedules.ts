import { isScheduleRequest } from "@/components/assistant/composer-commands";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type { ConversationSchedule, ConversationScheduleList } from "@/types/assistant";
import {
  appendToHistory,
  continuesHistory,
  type ThreadHistory,
} from "@/components/assistant/thread-history";
import { useMutation, useQueryClient, type QueryClient } from "@tanstack/react-query";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { handleMutationError } from "@/hooks/use-api-mutation";

export { isScheduleRequest };

const DAY_SECONDS = 24 * 60 * 60;

/** The calendar day an instant falls on in a timezone, as yyyy-mm-dd. */
function dayKey(unixSeconds: number, timezone: string): string {
  return new Intl.DateTimeFormat("en-CA", {
    timeZone: timezone,
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).format(new Date(unixSeconds * 1000));
}

/**
 * When a schedule runs next, the way its card says it: "Today · 7:30 AM",
 * "Tomorrow · 8:00 AM", or "Mon, Oct 5 · 7:30 AM". Read on the reader's clock.
 */
export function nextRunLabel(
  nextRunAt: number,
  now: number,
  timezone: string,
  t: TranslateFn,
): string {
  const at = new Date(nextRunAt * 1000);
  const time = new Intl.DateTimeFormat("en-US", {
    timeZone: timezone,
    hour: "numeric",
    minute: "2-digit",
  })
    .format(at)
    // Newer ICU sets "7:30 AM" with a narrow no-break space; the card's
    // cadence label, written by the server, uses a plain one.
    .replace(/\u202f/g, " ");

  const day = dayKey(nextRunAt, timezone);
  if (day === dayKey(now, timezone)) {
    return t("Today · {0}", time);
  }
  if (day === dayKey(now + DAY_SECONDS, timezone)) {
    return t("Tomorrow · {0}", time);
  }

  const date = new Intl.DateTimeFormat("en-US", {
    timeZone: timezone,
    weekday: "short",
    month: "short",
    day: "numeric",
  }).format(at);

  return `${date} · ${time}`;
}

/** The line under a schedule's request: "{when} · next {next}", or "paused". */
export function scheduleLine(
  schedule: Pick<ConversationSchedule, "cadence" | "enabled" | "nextRunAt">,
  now: number,
  timezone: string,
  t: TranslateFn,
): string {
  if (!schedule.enabled) {
    return t("{0} · paused", schedule.cadence);
  }
  if (!schedule.nextRunAt) {
    return schedule.cadence;
  }

  return t("{0} · next {1}", schedule.cadence, nextRunLabel(schedule.nextRunAt, now, timezone, t));
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

  return { items: list.items.map((item) => (item.id === id ? next : item)), total: list.total };
}

function writeSchedule(
  queryClient: QueryClient,
  threadId: string,
  id: string,
  next: ConversationSchedule | null,
) {
  queryClient.setQueryData<ConversationScheduleList>(
    queries.assistant.schedules(threadId).queryKey,
    (list) => patchScheduleList(list, id, next),
  );
}

/**
 * Pausing, resuming and deleting one of a conversation's schedules. Each
 * shows at once and is put back if the server refuses it.
 */
export function useScheduleActions(threadId: string) {
  const queryClient = useQueryClient();
  const key = queries.assistant.schedules(threadId).queryKey;

  const restore = (previous: ConversationScheduleList | undefined) => {
    queryClient.setQueryData(key, previous);
  };

  const toggle = useMutation({
    mutationFn: (schedule: ConversationSchedule) =>
      apiService.assistantService.setScheduleEnabled(schedule.id, !schedule.enabled),
    onMutate: async (schedule) => {
      await queryClient.cancelQueries({ queryKey: key });
      const previous = queryClient.getQueryData<ConversationScheduleList>(key);
      writeSchedule(queryClient, threadId, schedule.id, {
        ...schedule,
        enabled: !schedule.enabled,
      });
      return { previous };
    },
    onSuccess: (saved) => writeSchedule(queryClient, threadId, saved.id, saved),
    onError: (error, _schedule, context) => {
      restore(context?.previous);
      handleMutationError({ error, resourceName: "Schedule" });
    },
  });

  const remove = useMutation({
    mutationFn: (schedule: ConversationSchedule) =>
      apiService.assistantService.deleteSchedule(schedule.id),
    onMutate: async (schedule) => {
      await queryClient.cancelQueries({ queryKey: key });
      const previous = queryClient.getQueryData<ConversationScheduleList>(key);
      writeSchedule(queryClient, threadId, schedule.id, null);
      return { previous };
    },
    onError: (error, _schedule, context) => {
      restore(context?.previous);
      handleMutationError({ error, resourceName: "Schedule" });
    },
  });

  return {
    toggle: (schedule: ConversationSchedule) => toggle.mutate(schedule),
    remove: (schedule: ConversationSchedule) => remove.mutate(schedule),
    busy: toggle.isPending || remove.isPending,
  };
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
      const historyKey = queries.assistant.messages(threadId).queryKey;
      const cached = queryClient.getQueryData<ThreadHistory>(historyKey);
      if (continuesHistory(cached, [message])) {
        queryClient.setQueryData<ThreadHistory>(historyKey, (history) =>
          appendToHistory(history, [message]),
        );
      } else {
        await queryClient.invalidateQueries({ queryKey: historyKey });
      }

      const scheduleKey = queries.assistant.schedules(threadId).queryKey;
      if (queryClient.getQueryData(scheduleKey)) {
        writeSchedule(queryClient, threadId, schedule.id, schedule);
      } else {
        await queryClient.invalidateQueries({ queryKey: scheduleKey });
      }
      void queryClient.invalidateQueries({ queryKey: queries.assistant.threads().queryKey });
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
