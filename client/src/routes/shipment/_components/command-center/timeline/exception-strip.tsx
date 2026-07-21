import { cn } from "@/lib/utils";
import {
  ChevronLeftIcon,
  ChevronRightIcon,
  CircleCheckIcon,
  EyeIcon,
  InboxIcon,
  LayersIcon,
  TimerIcon,
  TriangleAlertIcon,
  XIcon,
} from "lucide-react";
import type { TimelineExceptionSummary, TimelineFocus } from "./use-timeline-data";

type ChipTone = "destructive" | "warning";

const CHIP_CONFIG: readonly {
  id: TimelineFocus;
  label: string;
  tone: ChipTone;
  Icon: typeof TimerIcon;
}[] = [
  { id: "late", label: "Late", tone: "destructive", Icon: TriangleAlertIcon },
  { id: "dwelling", label: "Dwelling", tone: "warning", Icon: TimerIcon },
  { id: "overlaps", label: "Overlaps", tone: "warning", Icon: LayersIcon },
  { id: "unassigned", label: "Unassigned", tone: "warning", Icon: InboxIcon },
  { id: "watch", label: "Watch", tone: "warning", Icon: EyeIcon },
] as const;

const CHIP_TONE_CLASS: Record<ChipTone, { idle: string; active: string }> = {
  destructive: {
    idle: "border-destructive/35 text-destructive hover:bg-destructive/10",
    active: "border-destructive bg-destructive/15 text-destructive",
  },
  warning: {
    idle: "border-warning/40 text-warning hover:bg-warning/10",
    active: "border-warning bg-warning/15 text-warning",
  },
};

type ExceptionStripProps = {
  exceptions: TimelineExceptionSummary;
  focus: TimelineFocus | null;
  matchCount: number;
  matchIndex: number;
  onFocusChange: (focus: TimelineFocus | null) => void;
  onStep: (direction: 1 | -1) => void;
};

/**
 * Triage strip for the dispatch timeline: one glance tells the operator what
 * needs attention in the visible window, one click isolates it, and the
 * stepper walks match-by-match so nothing gets missed.
 */
export function ExceptionStrip({
  exceptions,
  focus,
  matchCount,
  matchIndex,
  onFocusChange,
  onStep,
}: ExceptionStripProps) {
  const visibleChips = CHIP_CONFIG.filter(
    (chip) => exceptions[chip.id] > 0 || chip.id === focus,
  );
  const allClear = visibleChips.length === 0;

  return (
    <div className="flex min-h-7 flex-wrap items-center gap-1.5 border-b border-border bg-muted/30 px-3 py-1">
      <span className="text-[9.5px] font-semibold tracking-wide text-muted-foreground uppercase">
        Attention
      </span>
      {allClear ? (
        <span className="inline-flex items-center gap-1 text-[10.5px] text-muted-foreground">
          <CircleCheckIcon className="size-3 text-success" />
          All clear in this window
        </span>
      ) : (
        visibleChips.map(({ id, label, tone, Icon }) => {
          const isActive = focus === id;
          return (
            <button
              key={id}
              type="button"
              aria-pressed={isActive}
              onClick={() => onFocusChange(isActive ? null : id)}
              className={cn(
                "inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-[10.5px] font-medium transition-colors",
                isActive ? CHIP_TONE_CLASS[tone].active : CHIP_TONE_CLASS[tone].idle,
              )}
            >
              <Icon className="size-3" />
              {label}
              <span className="font-table text-[10px] font-semibold tabular-nums">
                {exceptions[id]}
              </span>
            </button>
          );
        })
      )}
      {focus && (
        <div className="ml-auto flex items-center gap-0.5">
          <span className="mr-1 font-table text-[10px] text-muted-foreground tabular-nums">
            {matchCount === 0 ? "No matches" : `${matchIndex + 1} / ${matchCount}`}
          </span>
          <button
            type="button"
            aria-label="Previous match"
            disabled={matchCount === 0}
            onClick={() => onStep(-1)}
            className="flex size-5 items-center justify-center rounded text-muted-foreground transition-colors hover:bg-muted hover:text-foreground disabled:pointer-events-none disabled:opacity-40"
          >
            <ChevronLeftIcon className="size-3.5" />
          </button>
          <button
            type="button"
            aria-label="Next match"
            disabled={matchCount === 0}
            onClick={() => onStep(1)}
            className="flex size-5 items-center justify-center rounded text-muted-foreground transition-colors hover:bg-muted hover:text-foreground disabled:pointer-events-none disabled:opacity-40"
          >
            <ChevronRightIcon className="size-3.5" />
          </button>
          <button
            type="button"
            aria-label="Clear focus"
            onClick={() => onFocusChange(null)}
            className="ml-0.5 flex size-5 items-center justify-center rounded text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
          >
            <XIcon className="size-3.5" />
          </button>
        </div>
      )}
    </div>
  );
}
