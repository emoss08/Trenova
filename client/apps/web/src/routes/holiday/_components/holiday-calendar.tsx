import { useT } from "@trenova/shared/i18n/use-t";
import { InfoPopover } from "@/components/info-popover";
import { usePermission } from "@/hooks/use-permission";
import {
  deleteOrgHoliday,
  fetchOrgHolidays,
  ORG_HOLIDAYS_KEY,
  type OrgHolidayRow,
} from "@/lib/graphql/org-holiday";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@trenova/shared/components/ui/alert-dialog";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import {
  daysInMonth,
  formatUtcDate,
  groupOccurrencesByMonth,
  isoDateKey,
  projectHolidaysOntoYear,
  utcMidnight,
  type HolidayOccurrence,
} from "@trenova/shared/lib/holiday";
import { cn } from "@trenova/shared/lib/utils";
import { ORG_HOLIDAY_KIND_LABELS } from "@trenova/shared/types/org-holiday";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  CalendarDaysIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
  PencilIcon,
  PlusIcon,
  RepeatIcon,
  Trash2Icon,
} from "lucide-react";
import { useCallback, useMemo, useState } from "react";
import { toast } from "sonner";
import { HolidayDialog } from "./holiday-dialog";

const MONTH_NAMES = Array.from({ length: 12 }, (_, month) =>
  new Intl.DateTimeFormat("en-US", { month: "long", timeZone: "UTC" }).format(
    new Date(Date.UTC(2024, month, 1)),
  ),
);
const WEEKDAY_LETTERS = ["S", "M", "T", "W", "T", "F", "S"];

type Occurrence = HolidayOccurrence<OrgHolidayRow>;

type DialogState = {
  open: boolean;
  entry: OrgHolidayRow | null;
  presetDate?: number;
};

export function HolidayCalendar() {
  const t = useT();

  const thisYear = new Date().getUTCFullYear();
  const [year, setYear] = useState(thisYear);
  const [dialog, setDialog] = useState<DialogState>({ open: false, entry: null });
  const [pendingDelete, setPendingDelete] = useState<OrgHolidayRow | null>(null);
  const queryClient = useQueryClient();

  const { allowed: canCreate } = usePermission(Resource.OrgHoliday, Operation.Create);
  const { allowed: canUpdate } = usePermission(Resource.OrgHoliday, Operation.Update);
  const { allowed: canDelete } = usePermission(Resource.OrgHoliday, Operation.Delete);

  const { data, isLoading, isError } = useQuery({
    queryKey: [ORG_HOLIDAYS_KEY, year],
    queryFn: ({ signal }) => fetchOrgHolidays(year, { signal }),
    staleTime: 60 * 1000,
  });

  const occurrences = useMemo(() => projectHolidaysOntoYear(data ?? [], year), [data, year]);
  const byMonth = useMemo(() => groupOccurrencesByMonth(occurrences), [occurrences]);
  const byDate = useMemo(() => {
    const map = new Map<string, Occurrence[]>();
    for (const occurrence of occurrences) {
      const bucket = map.get(occurrence.isoDate);
      if (bucket) bucket.push(occurrence);
      else map.set(occurrence.isoDate, [occurrence]);
    }
    return map;
  }, [occurrences]);
  const holidayCount = occurrences.filter((o) => o.entry.kind === "Holiday").length;
  const blackoutCount = occurrences.length - holidayCount;

  const invalidate = useCallback(
    () => queryClient.invalidateQueries({ queryKey: [ORG_HOLIDAYS_KEY], refetchType: "all" }),
    [queryClient],
  );

  const removeMutation = useMutation({
    mutationFn: (entry: OrgHolidayRow) => deleteOrgHoliday(entry.id),
    onSuccess: async (_, entry) => {
      toast.success(`${entry.name} removed`, {
        description: entry.recursAnnually
          ? "It no longer applies to any year."
          : `It no longer applies to ${formatUtcDate(entry.holidayDate)}.`,
      });
      setPendingDelete(null);
      await invalidate();
    },
    onError: (error: Error) => {
      toast.error(t("Could not remove the date"), { description: error.message });
    },
  });

  const openCreate = useCallback(
    (presetDate?: number) => setDialog({ open: true, entry: null, presetDate }),
    [],
  );
  const openEdit = useCallback((entry: OrgHolidayRow) => setDialog({ open: true, entry }), []);
  const onDayClick = useCallback(
    (isoDate: string, month: number, day: number) => {
      const marked = byDate.get(isoDate);
      if (marked && marked.length > 0) {
        if (canUpdate) openEdit(marked[0].entry);
        return;
      }
      if (canCreate) openCreate(utcMidnight(year, month, day));
    },
    [byDate, canCreate, canUpdate, openCreate, openEdit, year],
  );

  return (
    <div className="flex flex-col gap-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-2">
          <div className="bg-muted/40 flex items-center rounded-lg border p-0.5">
            <Button
              type="button"
              variant="ghost"
              size="icon"
              className="size-7"
              aria-label={t("Previous year")}
              onClick={() => setYear((current) => current - 1)}
            >
              <ChevronLeftIcon className="size-4" />
            </Button>
            <span className="min-w-14 text-center text-sm font-semibold tabular-nums">{year}</span>
            <Button
              type="button"
              variant="ghost"
              size="icon"
              className="size-7"
              aria-label={t("Next year")}
              onClick={() => setYear((current) => current + 1)}
            >
              <ChevronRightIcon className="size-4" />
            </Button>
          </div>
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={year === thisYear}
            onClick={() => setYear(thisYear)}
          >
            {t("This year")}
          </Button>
          <p className="text-muted-foreground hidden text-xs sm:block">
            {holidayCount} {holidayCount === 1 ? "holiday" : "holidays"} · {blackoutCount}{" "}
            {blackoutCount === 1 ? "blackout" : "blackouts"}
          </p>
          <InfoPopover title={t("How the calendar is used")}>
            {t("A time-off request that crosses a blackout is refused outright. A holiday inside a request is not charged against the balance, but only under a policy that does not count weekends; a policy that charges every calendar day charges holidays too.")}
          </InfoPopover>
        </div>
        <div className="flex items-center gap-3">
          <Legend />
          {canCreate ? (
            <Button type="button" size="sm" onClick={() => openCreate()}>
              <PlusIcon className="size-3.5" />
              {t("Add date")}
            </Button>
          ) : null}
        </div>
      </div>

      {isLoading ? (
        <div className="grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-4">
          {MONTH_NAMES.map((name) => (
            <Skeleton key={name} className="h-44 w-full" />
          ))}
        </div>
      ) : isError ? (
        <div className="text-destructive rounded-lg border border-dashed p-6 text-center text-sm">
          {t("The calendar could not be loaded. Try again in a moment.")}
        </div>
      ) : (
        <div className="grid grid-cols-2 gap-3 md:grid-cols-3 xl:grid-cols-4">
          {MONTH_NAMES.map((name, month) => (
            <MonthCard
              key={name}
              name={name}
              year={year}
              month={month}
              byDate={byDate}
              isCurrentYear={year === thisYear}
              onDayClick={onDayClick}
            />
          ))}
        </div>
      )}

      {!isLoading && !isError ? (
        occurrences.length === 0 ? (
          <div className="text-muted-foreground flex flex-col items-center gap-2 rounded-lg border border-dashed p-8 text-center text-sm">
            <CalendarDaysIcon className="size-6" />
            <p>{t("Nothing on the calendar for {0}.", year)}</p>
            {canCreate ? (
              <Button type="button" variant="outline" size="sm" onClick={() => openCreate()}>
                {t("Add the first date")}
              </Button>
            ) : null}
          </div>
        ) : (
          <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
            {byMonth.map((entries, month) =>
              entries.length > 0 ? (
                <section key={MONTH_NAMES[month]} className="flex flex-col gap-1.5">
                  <p className="text-muted-foreground px-1 text-[11px] font-medium uppercase">
                    {MONTH_NAMES[month]}
                  </p>
                  {entries.map((occurrence) => (
                    <EntryRow
                      key={`${occurrence.entry.id}-${occurrence.isoDate}`}
                      occurrence={occurrence}
                      canUpdate={canUpdate}
                      canDelete={canDelete}
                      onEdit={() => openEdit(occurrence.entry)}
                      onRemove={() => setPendingDelete(occurrence.entry)}
                    />
                  ))}
                </section>
              ) : null,
            )}
          </div>
        )
      ) : null}

      <HolidayDialog
        open={dialog.open}
        onOpenChange={(open) => setDialog((current) => ({ ...current, open }))}
        entry={dialog.entry}
        presetDate={dialog.presetDate}
        onSaved={() => void invalidate()}
      />

      <AlertDialog
        open={pendingDelete !== null}
        onOpenChange={(open) => !open && setPendingDelete(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t("Remove {0}?", pendingDelete?.name)}</AlertDialogTitle>
            <AlertDialogDescription>
              {pendingDelete?.kind === "Blackout"
                ? t("Workers will be able to request this date off again.")
                : t("Policies that skip weekends will start counting this date against requests.")}
              {pendingDelete?.recursAnnually ? ` ${t("This applies to every year.")}` : ""}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t("Keep")}</AlertDialogCancel>
            <AlertDialogAction
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              disabled={removeMutation.isPending}
              onClick={(event) => {
                event.preventDefault();
                if (pendingDelete) removeMutation.mutate(pendingDelete);
              }}
            >
              {t("Remove")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}

function Legend() {
  const t = useT();

  return (
    <div className="text-muted-foreground hidden items-center gap-3 text-[11px] sm:flex">
      <span className="flex items-center gap-1">
        <span className="size-2 rounded-full bg-emerald-500" />
        {t("Holidays")}
      </span>
      <span className="flex items-center gap-1">
        <span className="size-2 rounded-full bg-rose-500" />
        {t("Blackouts")}
      </span>
    </div>
  );
}

function MonthCard({
  name,
  year,
  month,
  byDate,
  isCurrentYear,
  onDayClick,
}: {
  name: string;
  year: number;
  month: number;
  byDate: Map<string, Occurrence[]>;
  isCurrentYear: boolean;
  onDayClick: (isoDate: string, month: number, day: number) => void;
}) {
  const leading = new Date(Date.UTC(year, month, 1)).getUTCDay();
  const total = daysInMonth(year, month);
  const today = new Date();
  const todayKey = isCurrentYear
    ? isoDateKey(today.getFullYear(), today.getMonth(), today.getDate())
    : null;

  return (
    <div className="bg-card rounded-lg border p-3">
      <h3 className="mb-2 text-sm font-semibold">{name}</h3>
      <div className="grid grid-cols-7 gap-y-0.5 text-center">
        {WEEKDAY_LETTERS.map((letter, index) => (
          <span key={`${letter}-${index}`} className="text-muted-foreground text-[10px]">
            {letter}
          </span>
        ))}
        {Array.from({ length: leading }, (_, index) => (
          <span key={`pad-${index}`} />
        ))}
        {Array.from({ length: total }, (_, index) => {
          const day = index + 1;
          const key = isoDateKey(year, month, day);
          const marked = byDate.get(key);
          const kind = marked?.[0]?.entry.kind;
          const label = marked?.map((occurrence) => occurrence.entry.name).join(", ");
          return (
            <button
              key={key}
              type="button"
              aria-label={label ? `${label} on ${formatUtcDate(marked![0].date)}` : undefined}
              title={label}
              onClick={() => onDayClick(key, month, day)}
              className={cn(
                "mx-auto flex size-6 items-center justify-center rounded-full text-[11px] tabular-nums transition-colors",
                kind === "Holiday" &&
                  "bg-emerald-500 font-semibold text-white hover:bg-emerald-600",
                kind === "Blackout" && "bg-rose-500 font-semibold text-white hover:bg-rose-600",
                !kind && "text-foreground/80 hover:bg-muted",
                todayKey === key && !kind && "ring-primary ring-1",
              )}
            >
              {day}
            </button>
          );
        })}
      </div>
    </div>
  );
}

function EntryRow({
  occurrence,
  canUpdate,
  canDelete,
  onEdit,
  onRemove,
}: {
  occurrence: Occurrence;
  canUpdate: boolean;
  canDelete: boolean;
  onEdit: () => void;
  onRemove: () => void;
}) {
  const t = useT();

  const { entry } = occurrence;
  const isBlackout = entry.kind === "Blackout";
  return (
    <div className="bg-card group flex items-center gap-3 rounded-lg border px-3 py-2">
      <span
        className={cn(
          "flex size-9 shrink-0 flex-col items-center justify-center rounded-md text-[10px] font-semibold uppercase",
          isBlackout
            ? "bg-rose-500/10 text-rose-700 dark:text-rose-400"
            : "bg-emerald-500/10 text-emerald-700 dark:text-emerald-400",
        )}
      >
        <span className="text-sm leading-none tabular-nums">{occurrence.day}</span>
        {formatUtcDate(occurrence.date, { month: "short", day: undefined, year: undefined })}
      </span>
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-medium">{entry.name}</p>
        <p className="text-muted-foreground truncate text-[11px]">
          {formatUtcDate(occurrence.date, { weekday: "short" })}
          {entry.description ? ` · ${entry.description}` : ""}
        </p>
      </div>
      <div className="flex shrink-0 items-center gap-1">
        {entry.recursAnnually ? (
          <Badge variant="outline" className="gap-1 px-1.5 py-0 text-[10px]">
            <RepeatIcon className="size-3" />
            {t("Every year")}
          </Badge>
        ) : null}
        <Badge
          variant="outline"
          className={cn(
            "px-1.5 py-0 text-[10px]",
            isBlackout
              ? "border-rose-500/40 text-rose-700 dark:text-rose-400"
              : "border-emerald-500/40 text-emerald-700 dark:text-emerald-400",
          )}
        >
          {ORG_HOLIDAY_KIND_LABELS[entry.kind]}
        </Badge>
        {canUpdate ? (
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="size-7"
            aria-label={`Edit ${entry.name}`}
            onClick={onEdit}
          >
            <PencilIcon className="size-3.5" />
          </Button>
        ) : null}
        {canDelete ? (
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="text-destructive size-7"
            aria-label={`Remove ${entry.name}`}
            onClick={onRemove}
          >
            <Trash2Icon className="size-3.5" />
          </Button>
        ) : null}
      </div>
    </div>
  );
}
