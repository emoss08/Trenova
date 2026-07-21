import { Button } from "@/components/ui/button";
import { Calendar } from "@/components/ui/calendar";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Spinner } from "@/components/ui/spinner";
import { cn } from "@/lib/utils";
import { CalendarIcon, ChevronLeftIcon, ChevronRightIcon, TriangleAlertIcon } from "lucide-react";
import { useState } from "react";
import { formatRangeLabel, isTodayAnchor, ZOOM_OPTIONS } from "./time-scale";
import type { TimelineZoom } from "../url-state";

const LEGEND_ITEMS: readonly { label: string; dotClass: string }[] = [
  { label: "On time", dotClass: "bg-brand" },
  { label: "Watch", dotClass: "bg-warning" },
  { label: "Late", dotClass: "bg-destructive" },
  { label: "Delivered", dotClass: "bg-success" },
] as const;

type TimelineToolbarProps = {
  anchor: Date;
  zoom: TimelineZoom;
  barCount: number;
  shipmentCount: number;
  totalCount: number;
  truncated: boolean;
  isFetching: boolean;
  onShift: (direction: 1 | -1) => void;
  onToday: () => void;
  onAnchorSelect: (date: Date) => void;
  onZoomChange: (zoom: TimelineZoom) => void;
};

export function TimelineToolbar({
  anchor,
  zoom,
  barCount,
  shipmentCount,
  totalCount,
  truncated,
  isFetching,
  onShift,
  onToday,
  onAnchorSelect,
  onZoomChange,
}: TimelineToolbarProps) {
  const [calendarOpen, setCalendarOpen] = useState(false);

  return (
    <div className="flex flex-wrap items-center gap-2 border-b border-border px-3 py-1.5">
      <div className="flex items-center gap-1">
        <Button
          type="button"
          variant="ghost"
          size="icon-xs"
          aria-label="Previous period"
          onClick={() => onShift(-1)}
        >
          <ChevronLeftIcon className="size-3.5" />
        </Button>
        <Button
          type="button"
          variant="outline"
          size="xxs"
          onClick={onToday}
          disabled={isTodayAnchor(anchor)}
        >
          Today
        </Button>
        <Button
          type="button"
          variant="ghost"
          size="icon-xs"
          aria-label="Next period"
          onClick={() => onShift(1)}
        >
          <ChevronRightIcon className="size-3.5" />
        </Button>
      </div>

      <Popover open={calendarOpen} onOpenChange={setCalendarOpen}>
        <PopoverTrigger
          className="flex items-center gap-1.5 rounded-md px-1.5 py-0.5 text-[11.5px] font-medium transition-colors hover:bg-muted"
          aria-label="Jump to date"
        >
          <CalendarIcon className="size-3 text-muted-foreground" />
          {formatRangeLabel(anchor, zoom)}
        </PopoverTrigger>
        <PopoverContent align="start" className="w-auto p-0">
          <Calendar
            mode="single"
            selected={anchor}
            defaultMonth={anchor}
            onSelect={(date) => {
              if (date) {
                onAnchorSelect(date);
                setCalendarOpen(false);
              }
            }}
          />
        </PopoverContent>
      </Popover>

      <div
        role="group"
        aria-label="Timeline zoom"
        className="inline-flex overflow-hidden rounded-md border border-border"
      >
        {ZOOM_OPTIONS.map((option, index) => (
          <button
            key={option.id}
            type="button"
            onClick={() => onZoomChange(option.id)}
            aria-pressed={zoom === option.id}
            className={cn(
              "px-2 py-1 text-[11px] transition-colors",
              index > 0 && "border-l border-border",
              zoom === option.id
                ? "bg-muted text-foreground"
                : "bg-background text-muted-foreground hover:text-foreground",
            )}
          >
            {option.label}
          </button>
        ))}
      </div>

      <div className="ml-auto flex items-center gap-3">
        {isFetching && (
          <span className="inline-flex items-center gap-1 text-[10px] text-muted-foreground">
            <Spinner className="size-3" />
            Refreshing
          </span>
        )}
        {truncated && (
          <span
            className="inline-flex items-center gap-1 rounded border border-warning/30 bg-warning/10 px-1.5 py-0.5 text-[10px] text-warning"
            title="Narrow the window or filters to see everything at once."
          >
            <TriangleAlertIcon className="size-3" />
            Showing first {shipmentCount} of {totalCount} shipments
          </span>
        )}
        <div className="hidden items-center gap-2.5 md:flex">
          {LEGEND_ITEMS.map((item) => (
            <span
              key={item.label}
              className="inline-flex items-center gap-1 text-[10px] text-muted-foreground"
            >
              <span className={cn("size-1.5 rounded-full", item.dotClass)} />
              {item.label}
            </span>
          ))}
        </div>
        <p className="shrink-0 font-table text-[10.5px] text-muted-foreground tabular-nums">
          {barCount} {barCount === 1 ? "load" : "loads"} in view
        </p>
      </div>
    </div>
  );
}
