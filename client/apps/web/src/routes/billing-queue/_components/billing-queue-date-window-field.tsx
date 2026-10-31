import { LAST_DAYS, type DateWindow } from "@/lib/billing-queue-date-window";
import { CalendarIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { Calendar } from "@trenova/shared/components/ui/calendar";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixMonthDay, fromUserWallClock, toUserWallClock } from "@trenova/shared/lib/date";
import { cn } from "@trenova/shared/lib/utils";
import { useState } from "react";

const ANY = "any";
const RANGE = "range";

/**
 * One of the queue's date filters: any time, the last 7, 30 or 90 days, or a range of
 * days picked on a calendar.
 */
export function BillingQueueDateWindowField({
  label,
  value,
  onChange,
}: {
  label: string;
  value: DateWindow | null;
  onChange: (value: DateWindow | null) => void;
}) {
  const t = useT();
  // "Pick dates…" chosen but no range picked yet: the window in force is cleared, so
  // the select says what the list is filtered by while the calendar is open.
  const [choosingRange, setChoosingRange] = useState(false);
  const choice =
    value?.kind === "range"
      ? RANGE
      : value?.kind === "last"
        ? String(value.days)
        : choosingRange
          ? RANGE
          : ANY;
  const items = [
    { value: ANY, label: t("Any time") },
    ...LAST_DAYS.map((days) => ({
      value: String(days),
      label: t("{0, plural, one {Last day} other {Last # days}}", days),
    })),
    { value: RANGE, label: t("Pick dates…") },
  ];

  return (
    <div className="grid grid-cols-[5.5rem_minmax(0,1fr)] items-center gap-3">
      <p className="text-muted-foreground text-xs">{label}</p>
      <div className="flex min-w-0 gap-1.5">
        <Select
          value={choice}
          items={items}
          onValueChange={(next) => {
            if (next === RANGE) {
              setChoosingRange(true);
              if (value?.kind !== "range") onChange(null);
              return;
            }
            setChoosingRange(false);
            onChange(next === ANY || next === null ? null : { kind: "last", days: Number(next) });
          }}
        >
          <SelectTrigger aria-label={label} className="h-7 w-full min-w-0 flex-1 text-xs">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {items.map((item) => (
              <SelectItem key={item.value} value={item.value}>
                {item.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        {choice === RANGE ? (
          <DateRangePicker
            label={label}
            value={value?.kind === "range" ? value : null}
            onChange={(range) => {
              setChoosingRange(false);
              onChange(range);
            }}
          />
        ) : null}
      </div>
    </div>
  );
}

/** The days of a range, picked on a calendar; the window is set once both ends are. */
function DateRangePicker({
  label,
  value,
  onChange,
}: {
  label: string;
  value: Extract<DateWindow, { kind: "range" }> | null;
  onChange: (value: DateWindow) => void;
}) {
  const t = useT();
  const [open, setOpen] = useState(value === null);

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        render={
          <Button
            type="button"
            variant="outline"
            aria-label={t("{0} dates", label)}
            className={cn(
              "h-7 min-w-0 flex-1 justify-start gap-1.5 px-2 text-xs font-normal",
              !value && "text-muted-foreground",
            )}
          >
            <CalendarIcon className="size-3.5 shrink-0" />
            <span className="truncate">
              {value
                ? value.from === value.to
                  ? formatUnixMonthDay(value.from)
                  : t("{0} – {1}", formatUnixMonthDay(value.from), formatUnixMonthDay(value.to))
                : t("Pick dates")}
            </span>
          </Button>
        }
      />
      <PopoverContent className="w-auto p-0" align="end">
        <Calendar
          mode="range"
          selected={
            value ? { from: toUserWallClock(value.from), to: toUserWallClock(value.to) } : undefined
          }
          onSelect={(range) => {
            const from = fromUserWallClock(range?.from);
            const to = fromUserWallClock(range?.to);
            if (from != null && to != null) {
              onChange({ kind: "range", from, to });
            }
          }}
        />
      </PopoverContent>
    </Popover>
  );
}
