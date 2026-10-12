import { Button, type ButtonProps } from "@trenova/shared/components/ui/button";
import { cn } from "@trenova/shared/lib/utils";
import { useCallback, useEffect, useRef, useState, type KeyboardEvent } from "react";

export type HoldButtonProps = Omit<ButtonProps, "onClick" | "onPointerDown" | "onKeyDown"> & {
  /** Runs once the button has been held down until it fills. */
  onDone: () => void;
  /** How long the hold takes, in milliseconds. */
  ms?: number;
};

/**
 * A button that acts only once held down until it fills, for an action worth a
 * moment's pause. Releasing, leaving or Escape before it fills cancels it.
 *
 * The fill is a CSS transition over the hold's length, not a per-frame state, so
 * a hold renders the button twice — at its start and its end — however long it is.
 */
export function HoldButton({
  onDone,
  ms = 900,
  disabled,
  className,
  children,
  ...props
}: HoldButtonProps) {
  const [holding, setHolding] = useState(false);
  const timer = useRef<number | null>(null);
  const done = useRef(onDone);
  useEffect(() => {
    done.current = onDone;
  });

  const stop = useCallback(() => {
    if (timer.current !== null) {
      window.clearTimeout(timer.current);
      timer.current = null;
    }
    setHolding(false);
  }, []);

  const start = useCallback(() => {
    if (disabled) return;
    stop();
    setHolding(true);
    timer.current = window.setTimeout(() => {
      timer.current = null;
      setHolding(false);
      done.current();
    }, ms);
  }, [disabled, ms, stop]);

  useEffect(() => stop, [stop]);

  const onKeyDown = (event: KeyboardEvent<HTMLButtonElement>) => {
    if ((event.key === " " || event.key === "Enter") && !event.repeat) {
      event.preventDefault();
      start();
    } else if (event.key === "Escape") {
      stop();
    }
  };

  return (
    <Button
      type="button"
      variant="outline"
      {...props}
      disabled={disabled}
      data-holding={holding || undefined}
      className={cn(
        "hover:border-warning relative touch-none overflow-hidden transition-[border-color,scale] select-none data-holding:scale-[0.985]",
        className,
      )}
      onPointerDown={start}
      onPointerUp={stop}
      onPointerLeave={stop}
      onPointerCancel={stop}
      onKeyDown={onKeyDown}
      onKeyUp={stop}
    >
      <span
        aria-hidden
        className={cn(
          "bg-warning/28 pointer-events-none absolute inset-0 origin-left transition-transform ease-linear",
          holding ? "scale-x-100" : "scale-x-0 duration-0",
        )}
        style={holding ? { transitionDuration: `${ms}ms` } : undefined}
      />
      <span className="relative flex items-center justify-center gap-2">{children}</span>
    </Button>
  );
}
