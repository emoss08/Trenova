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
import { formatDateInUserTimezone } from "@trenova/shared/lib/date";

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

/** A time as the reader's language and clock write it, without the narrow space newer ICU puts before AM/PM. */
function clockTime(date: Date, timezone: string): string {
  return formatDateInUserTimezone(date, { hour: "numeric", minute: "2-digit", timezone }).replace(
    /\u202f/g,
    " ",
  );
}

/**
 * When a schedule runs next, the way its card says it: "Today · 7:30 AM",
 * "Tomorrow · 8:00 AM", or "Mon, Oct 5 · 7:30 AM". Read on the reader's clock
 * and written in their language.
 */
export function nextRunLabel(
  nextRunAt: number,
  now: number,
  timezone: string,
  t: TranslateFn,
): string {
  const at = new Date(nextRunAt * 1000);
  const time = clockTime(at, timezone);

  const day = dayKey(nextRunAt, timezone);
  if (day === dayKey(now, timezone)) {
    return t("Today · {0}", time);
  }
  if (day === dayKey(now + DAY_SECONDS, timezone)) {
    return t("Tomorrow · {0}", time);
  }

  const date = formatDateInUserTimezone(at, {
    timezone,
    weekday: "short",
    month: "short",
    day: "numeric",
  });

  return `${date} · ${time}`;
}

const CRON_NUMBER = /^\d{1,2}$/;

/**
 * When a schedule runs, in the reader's language: "Every weekday · 7:30 AM".
 * It is written from the schedule's cron, the time as the cron names it on
 * the schedule's own clock; the English label the server stored is the
 * fallback for a cron this does not describe.
 */
export function cadenceLabel(
  schedule: Pick<ConversationSchedule, "cadence" | "cronExpression">,
  t: TranslateFn,
): string {
  const fields = schedule.cronExpression.trim().split(/\s+/);
  if (fields.length !== 5) {
    return schedule.cadence;
  }
  const [minuteField, hourField, dayOfMonth, month, dayOfWeek] = fields;
  if (!CRON_NUMBER.test(minuteField) || !CRON_NUMBER.test(hourField)) {
    return schedule.cadence;
  }
  const hour = Number(hourField);
  const minute = Number(minuteField);
  if (hour > 23 || minute > 59 || dayOfMonth !== "*" || month !== "*") {
    return schedule.cadence;
  }
  const time = clockTime(new Date(Date.UTC(2000, 0, 1, hour, minute)), "UTC");

  switch (dayOfWeek) {
    case "*":
      return t("Every day · {0}", time);
    case "1-5":
      return t("Every weekday · {0}", time);
    case "0":
    case "7":
      return t("Every Sunday · {0}", time);
    case "1":
      return t("Every Monday · {0}", time);
    case "2":
      return t("Every Tuesday · {0}", time);
    case "3":
      return t("Every Wednesday · {0}", time);
    case "4":
      return t("Every Thursday · {0}", time);
    case "5":
      return t("Every Friday · {0}", time);
    case "6":
      return t("Every Saturday · {0}", time);
    default:
      return schedule.cadence;
  }
}

/** The line under a schedule's request: "{when} · next {next}", or "paused". */
export function scheduleLine(
  schedule: Pick<ConversationSchedule, "cadence" | "cronExpression" | "enabled" | "nextRunAt">,
  now: number,
  timezone: string,
  t: TranslateFn,
): string {
  const cadence = cadenceLabel(schedule, t);
  if (!schedule.enabled) {
    return t("{0} · paused", cadence);
  }
  if (!schedule.nextRunAt) {
    return cadence;
  }

  return t("{0} · next {1}", cadence, nextRunLabel(schedule.nextRunAt, now, timezone, t));
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
