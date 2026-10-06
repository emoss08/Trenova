import { useT } from "@trenova/shared/i18n/use-t";
import { CheckIcon } from "@trenova/shared/components/icons";
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
};

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
        className="bg-success/25 absolute inset-y-0 left-0"
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
