import { formatRangeLabelForDays, isTodayAnchor } from "@/lib/timeline/time-scale";
import { Button } from "@trenova/shared/components/ui/button";
import { Calendar } from "@trenova/shared/components/ui/calendar";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { Switch } from "@trenova/shared/components/ui/switch";
import { cn } from "@trenova/shared/lib/utils";
import {
  CalendarIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
  Columns3Icon,
  GanttChartIcon,
  SparklesIcon,
  Undo2Icon,
} from "lucide-react";
import { preloadCenterView } from "./console-center-pane";
import { TIMELINE_ZOOM_OPTIONS } from "./timeline-metrics";
import { useDispatchView, useDispatchWindow, type CenterMode } from "./url-state";

const CENTER_MODE_OPTIONS: readonly { id: CenterMode; label: string; Icon: typeof Columns3Icon }[] =
  [
    { id: "board", label: "Board", Icon: Columns3Icon },
    { id: "timeline", label: "Timeline", Icon: GanttChartIcon },
  ];

/**
 * The window the board is asked about, the shape it is drawn in, and the two actions that
 * write to it. Everything the toolbar changes lives in the URL, so it owns that state
 * directly rather than being handed it.
 */
export function ConsoleToolbar({
  canUndo,
  isAssigning,
  isPlanning,
  onUndo,
  onPlan,
}: {
  canUndo: boolean;
  isAssigning: boolean;
  isPlanning: boolean;
  onUndo: () => void;
  onPlan: () => void;
}) {
  const {
    anchor,
    zoom,
    includeCovered,
    setAnchor,
    shiftWindow,
    goToday,
    setZoom,
    setIncludeCovered,
  } = useDispatchWindow();
  const { mode, setMode, timelineAvailable } = useDispatchView();

  return (
    <div className="flex flex-wrap items-center gap-2">
      <div className="flex items-center gap-1">
        <Button
          type="button"
          variant="ghost"
          size="icon-xs"
          aria-label="Previous window"
          title="Previous window (←)"
          onClick={() => shiftWindow(-1)}
        >
          <ChevronLeftIcon className="size-3.5" />
        </Button>
        <Button
          type="button"
          variant="outline"
          size="xxs"
          title="Jump to today (T)"
          onClick={goToday}
          disabled={isTodayAnchor(anchor)}
        >
          Today
        </Button>
        <Button
          type="button"
          variant="ghost"
          size="icon-xs"
          aria-label="Next window"
          title="Next window (→)"
          onClick={() => shiftWindow(1)}
        >
          <ChevronRightIcon className="size-3.5" />
        </Button>
      </div>

      <Popover>
        <PopoverTrigger
          className="flex items-center gap-1.5 rounded-md px-1.5 py-0.5 text-[11.5px] font-medium transition-colors hover:bg-muted"
          aria-label="Jump to date"
        >
          <CalendarIcon className="size-3 text-muted-foreground" />
          {formatRangeLabelForDays(anchor, zoom)}
        </PopoverTrigger>
        <PopoverContent align="start" className="w-auto p-0">
          <Calendar
            mode="single"
            selected={anchor}
            defaultMonth={anchor}
            onSelect={(date) => date && setAnchor(date)}
          />
        </PopoverContent>
      </Popover>

      <div
        role="group"
        aria-label="Window length"
        className="inline-flex overflow-hidden rounded-md border border-border"
      >
        {TIMELINE_ZOOM_OPTIONS.map((option, index) => (
          <button
            key={option.id}
            type="button"
            onClick={() => setZoom(option.id)}
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

      {/* With the driver timeline withheld the board is the only centre there is,
          so the toggle goes rather than offering a choice of one. */}
      {timelineAvailable && (
        <div
          role="group"
          aria-label="Center view"
          className="inline-flex overflow-hidden rounded-md border border-border"
        >
          {CENTER_MODE_OPTIONS.map((option, index) => (
            <button
              key={option.id}
              type="button"
              onClick={() => setMode(option.id)}
              // Reaching for the toggle is enough warning to fetch the view behind it.
              onPointerEnter={() => preloadCenterView(option.id)}
              onFocus={() => preloadCenterView(option.id)}
              aria-pressed={mode === option.id}
              className={cn(
                "flex items-center gap-1 px-2 py-1 text-[11px] transition-colors",
                index > 0 && "border-l border-border",
                mode === option.id
                  ? "bg-muted text-foreground"
                  : "bg-background text-muted-foreground hover:text-foreground",
              )}
            >
              <option.Icon className="size-3" aria-hidden />
              {option.label}
            </button>
          ))}
        </div>
      )}

      <label className="flex items-center gap-1.5 text-[11px] text-muted-foreground">
        <Switch checked={includeCovered} onCheckedChange={setIncludeCovered} />
        Show covered
      </label>

      <div className="ml-auto flex items-center gap-1.5">
        <Button
          size="sm"
          variant="outline"
          className="h-7 gap-1 px-2 text-[11px]"
          disabled={!canUndo || isAssigning}
          onClick={onUndo}
        >
          <Undo2Icon className="size-3" aria-hidden />
          Undo
          <Kbd className="h-4 min-w-4 text-[9px]">u</Kbd>
        </Button>
        <Button
          size="sm"
          className="h-7 gap-1 px-2 text-[11px]"
          disabled={isPlanning}
          isLoading={isPlanning}
          onClick={onPlan}
        >
          <SparklesIcon className="size-3" aria-hidden />
          Auto-assign
        </Button>
      </div>
    </div>
  );
}
