import { useT } from "@trenova/shared/i18n/use-t";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { getTodayDate } from "@trenova/shared/lib/date";
import {
  fetchMyAvailability,
  fetchMySchedule,
  fetchMyShiftSwaps,
  respondToMyShiftSwap,
  setMyAvailability,
  type MyScheduleDay,
} from "@trenova/shared/lib/graphql/driver-portal";
import {
  DAY_LABELS,
  DAY_LABELS_LONG,
  formatShiftDate,
  formatShiftWindow,
  rotaStateTone,
  SWAP_STATUS_TONES,
  swapActionsFor,
} from "@trenova/shared/lib/scheduling";
import { cn } from "@trenova/shared/lib/utils";
import type { AvailabilityPreferenceValue } from "@trenova/shared/types/scheduling";
import { CalendarClockIcon, RepeatIcon } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";

type PreferenceChoice = AvailabilityPreferenceValue | "None";

const PREFERENCE_ITEMS = [
  { value: "Preferred", label: "Prefer" },
  { value: "Available", label: "Can" },
  { value: "Unavailable", label: "Can't" },
] satisfies { value: PreferenceChoice; label: string }[];

const SECONDS_IN_DAY = 86400;

/**
 * The driver's own week, what they have told the office they would rather
 * work, and the swaps they are part of.
 *
 * Availability is a statement and never a constraint — the card says so, so a
 * driver who marks Sunday "can't" and then gets a Sunday load is not surprised
 * by a promise the app never made.
 */
export function ScheduleCard() {
  const t = useT();

  const queryClient = useQueryClient();
  const [selected, setSelected] = useState<MyScheduleDay | null>(null);

  const schedule = useQuery({
    queryKey: ["dash-schedule"],
    queryFn: ({ signal }) => fetchMySchedule({ weeks: 2 }, { signal }),
    staleTime: 60 * 1000,
  });
  const availability = useQuery({
    queryKey: ["dash-availability"],
    queryFn: ({ signal }) => fetchMyAvailability({ signal }),
    staleTime: 5 * 60 * 1000,
  });
  const swaps = useQuery({
    queryKey: ["dash-shift-swaps"],
    queryFn: ({ signal }) => fetchMyShiftSwaps({ signal }),
    staleTime: 60 * 1000,
  });

  const { mutate: saveAvailability, isPending: savingAvailability } = useMutation({
    mutationFn: setMyAvailability,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ["dash-availability"] });
      void queryClient.invalidateQueries({ queryKey: ["dash-schedule"] });
    },
    onError: (error: Error) => toast.error(error.message),
  });

  const { mutate: answerSwap, isPending: answeringSwap } = useMutation({
    mutationFn: respondToMyShiftSwap,
    onSuccess: () => {
      toast.success(t("Sent to your carrier"));
      void queryClient.invalidateQueries({ queryKey: ["dash-shift-swaps"] });
      void queryClient.invalidateQueries({ queryKey: ["dash-schedule"] });
    },
    onError: (error: Error) => toast.error(error.message),
  });

  if (schedule.isPending) {
    return <Skeleton className="h-56 w-full rounded-2xl" />;
  }
  if (!schedule.data) return null;

  const week = schedule.data;
  const stated = availability.data ?? [];
  const openSwaps = (swaps.data ?? []).filter(
    (swap) => swap.status === "Proposed" || swap.status === "Accepted",
  );
  const today = getTodayDate();
  const weeks = [week.days.slice(0, 7), week.days.slice(7, 14)].filter((days) => days.length > 0);
  const focused =
    selected ??
    week.days.find((day) => day.date <= today && today < day.date + SECONDS_IN_DAY) ??
    null;

  return (
    <div className="border-border bg-card rounded-2xl border p-4" data-testid="schedule-card">
      <div className="flex items-start justify-between gap-3">
        <div className="flex items-start gap-2">
          <CalendarClockIcon className="text-muted-foreground mt-0.5 size-4 shrink-0" />
          <div className="min-w-0">
            <h2 className="text-sm font-semibold">{t("Your schedule")}</h2>
            <p className="text-muted-foreground flex items-center gap-1.5 text-xs">
              {week.shiftColor ? (
                <span
                  className="size-2 shrink-0 rounded-full"
                  style={{ backgroundColor: week.shiftColor }}
                  aria-hidden
                />
              ) : null}
              {week.shiftName ?? t("No shift")} · {week.scheduledDays} day
              {week.scheduledDays === 1 ? "" : "s"} {t("over two weeks")}
            </p>
          </div>
        </div>
        {week.shiftCode ? <Badge variant="secondary">{week.shiftCode}</Badge> : null}
      </div>

      <div className="mt-4 flex flex-col gap-2">
        {weeks.map((days, index) => (
          <div key={days[0]?.date ?? index} className="grid grid-cols-7 gap-1">
            {days.map((day) => {
              const tone = rotaStateTone(day.state);
              const isToday = day.date <= today && today < day.date + SECONDS_IN_DAY;
              const isFocused = focused?.date === day.date;
              const date = new Date(day.date * 1000);
              return (
                <button
                  key={day.date}
                  type="button"
                  onClick={() => setSelected(day)}
                  aria-pressed={isFocused}
                  aria-label={`${DAY_LABELS_LONG[date.getUTCDay()]} ${date.getUTCDate()}: ${tone.label}`}
                  className={cn(
                    "flex flex-col items-center gap-1 rounded-lg border px-1 py-2 text-[11px] transition-colors active:brightness-95",
                    tone.cell,
                    isFocused && "ring-primary/60 ring-2",
                  )}
                >
                  <span className={cn("leading-none", isToday && "text-primary font-semibold")}>
                    {DAY_LABELS[date.getUTCDay()]}
                  </span>
                  <span
                    className={cn(
                      "grid size-6 place-items-center rounded-full leading-none tabular-nums",
                      isToday && "bg-primary text-primary-foreground",
                    )}
                  >
                    {date.getUTCDate()}
                  </span>
                  <span className={cn("size-1.5 rounded-full", tone.dot)} aria-hidden />
                </button>
              );
            })}
          </div>
        ))}
      </div>

      {focused ? (
        <div className="bg-muted/30 border-border mt-3 flex items-center justify-between gap-3 rounded-lg border px-3 py-2 text-xs">
          <div>
            <p className="font-medium">{formatShiftDate(focused.date)}</p>
            <p className="text-muted-foreground">
              {focused.scheduled
                ? formatShiftWindow(focused.startMinute, focused.durationMinutes)
                : rotaStateTone(focused.state).label}
              {focused.assignmentCount > 0
                ? t("· {0, plural, one {# load} other {# loads}} assigned", focused.assignmentCount)
                : ""}
            </p>
          </div>
          <Badge variant={rotaStateTone(focused.state).variant}>
            {t(rotaStateTone(focused.state).label)}
          </Badge>
        </div>
      ) : null}

      <div className="border-border mt-4 border-t pt-3">
        <p className="text-sm font-semibold">{t("What you would rather work")}</p>
        <p className="text-muted-foreground text-xs">
          {t(
            "Your carrier sees this when they build the rota. It is not a promise — dispatch can still put you on a day you marked, and they will see that they did.",
          )}
        </p>
        <ul className="mt-2 flex flex-col gap-1.5">
          {DAY_LABELS_LONG.map((label, dayOfWeek) => {
            const current = stated.find((row) => row.dayOfWeek === dayOfWeek);
            const value = (current?.preference ?? "None") as PreferenceChoice;
            return (
              <li key={label} className="flex items-center justify-between gap-2 text-xs">
                <span className="font-medium">{label}</span>
                <SegmentedControl<PreferenceChoice>
                  items={PREFERENCE_ITEMS.map((item) => ({
                    ...item,
                    disabled: savingAvailability,
                  }))}
                  value={value}
                  onValueChange={(next) => {
                    if (next === "None") return;
                    saveAvailability({ dayOfWeek, preference: next });
                  }}
                  aria-label={`${label} availability`}
                />
              </li>
            );
          })}
        </ul>
      </div>

      {openSwaps.length > 0 ? (
        <div className="border-border mt-4 border-t pt-3">
          <div className="flex items-center gap-2">
            <RepeatIcon className="text-muted-foreground size-4" />
            <p className="text-sm font-semibold">{t("Swaps")}</p>
          </div>
          <ul className="mt-2 flex flex-col gap-2">
            {openSwaps.map((swap) => {
              const tone = SWAP_STATUS_TONES[swap.status] ?? SWAP_STATUS_TONES.Withdrawn;
              const actions = swapActionsFor(swap);
              return (
                <li
                  key={swap.id}
                  className="bg-muted/30 border-border rounded-lg border p-3 text-xs"
                >
                  <div className="flex flex-wrap items-center gap-1.5">
                    <span className="font-medium tabular-nums">
                      {formatShiftDate(swap.shiftDate)}
                    </span>
                    <Badge variant={tone.variant}>{t(tone.label)}</Badge>
                    {swap.counterpartyName ? (
                      <span className="text-muted-foreground">
                        {swap.outgoing ? "to" : "from"} {swap.counterpartyName}
                      </span>
                    ) : null}
                  </div>
                  {swap.reason ? (
                    <p className="text-muted-foreground mt-0.5">“{swap.reason}”</p>
                  ) : null}
                  {actions.length > 0 ? (
                    <div className="mt-2 flex flex-wrap gap-1.5">
                      {actions.includes("accept") ? (
                        <Button
                          size="sm"
                          className="h-9"
                          disabled={answeringSwap}
                          onClick={() => answerSwap({ id: swap.id, response: "Accepted" })}
                        >
                          {t("Take it")}
                        </Button>
                      ) : null}
                      {actions.includes("decline") ? (
                        <Button
                          size="sm"
                          variant="outline"
                          className="h-9"
                          disabled={answeringSwap}
                          onClick={() => answerSwap({ id: swap.id, response: "Declined" })}
                        >
                          {t("Decline")}
                        </Button>
                      ) : null}
                      {actions.includes("withdraw") ? (
                        <Button
                          size="sm"
                          variant="outline"
                          className="h-9"
                          disabled={answeringSwap}
                          onClick={() => answerSwap({ id: swap.id, response: "Withdrawn" })}
                        >
                          {t("Withdraw")}
                        </Button>
                      ) : null}
                    </div>
                  ) : null}
                </li>
              );
            })}
          </ul>
        </div>
      ) : null}
    </div>
  );
}
