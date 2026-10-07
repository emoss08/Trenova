import { useT } from "@trenova/shared/i18n/use-t";
import { CheckIcon, type IconComponent } from "@trenova/shared/components/icons";
import { cn } from "@trenova/shared/lib/utils";
import { useEffect, useRef, useState, type ReactNode } from "react";

const DEFAULT_HOLD_MS = 900;

type HoldButtonProps = {
  label: ReactNode;
  doneLabel: ReactNode;
  done: boolean;
  disabled?: boolean;
  holdMs?: number;
  onConfirm: () => void;
  /** What the fill warns of: success for something done, warning for a stop. */
  tone?: "success" | "warning";
  icon?: IconComponent;
};

const FILL_TONE = { success: "bg-success/25", warning: "bg-warning/25" } as const;

/**
 * A button for an action too consequential for a click: the press has to be
 * held while it fills, and letting go early cancels it. Enter or Space held
 * on the focused button works the same way.
 */
export function HoldButton({
  label,
  doneLabel,
  done,
  disabled = false,
  holdMs = DEFAULT_HOLD_MS,
  onConfirm,
  tone = "success",
  icon: Icon,
}: HoldButtonProps) {
  const t = useT();
  const [progress, setProgress] = useState(0);
  const frame = useRef<number | null>(null);
  const confirmRef = useRef(onConfirm);
  confirmRef.current = onConfirm;

  const stop = () => {
    if (frame.current != null) cancelAnimationFrame(frame.current);
    frame.current = null;
    if (!done) setProgress(0);
  };

  const start = () => {
    if (done || disabled || frame.current != null) return;
    const started = performance.now();
    const tick = (now: number) => {
      const value = Math.min(1, (now - started) / holdMs);
      setProgress(value);
      if (value >= 1) {
        frame.current = null;
        confirmRef.current();
        return;
      }
      frame.current = requestAnimationFrame(tick);
    };
    frame.current = requestAnimationFrame(tick);
  };

  useEffect(() => () => {
    if (frame.current != null) cancelAnimationFrame(frame.current);
  }, []);

  return (
    <button
      type="button"
      disabled={disabled}
      aria-disabled={done || undefined}
      onPointerDown={start}
      onPointerUp={stop}
      onPointerLeave={stop}
      onKeyDown={(event) => {
        if ((event.key === "Enter" || event.key === " ") && !event.repeat) {
          event.preventDefault();
          start();
        }
      }}
      onKeyUp={(event) => {
        if (event.key === "Enter" || event.key === " ") stop();
      }}
      className={cn(
        "ui-focus-ring border-input bg-card relative flex h-8 w-full items-center justify-center overflow-hidden rounded-md border text-sm font-medium select-none disabled:opacity-50",
        done && "border-success-border text-success",
      )}
    >
      <span
        aria-hidden
        className={cn("absolute inset-y-0 left-0", FILL_TONE[tone])}
        style={{ width: `${(done ? 1 : progress) * 100}%` }}
      />
      <span className="relative inline-flex items-center gap-1.5">
        {done ? (
          <>
            <CheckIcon className="size-3.5" />
            {doneLabel}
          </>
        ) : (
          <>
            {Icon && <Icon className="size-3.5" />}
            {label}
            <em className="text-muted-foreground text-xs not-italic">
              {progress > 0 ? t("keep holding") : t("hold")}
            </em>
          </>
        )}
      </span>
    </button>
  );
}
