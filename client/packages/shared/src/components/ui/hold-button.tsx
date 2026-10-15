import { CheckIcon } from "@trenova/shared/components/icons";
import { Button, type ButtonProps } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import {
  holdButtonVariants,
  holdFillVariants,
  type HoldTone,
} from "@trenova/shared/lib/variants/hold-button";
import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type KeyboardEvent,
  type ReactNode,
} from "react";

export type HoldButtonProps = Omit<ButtonProps, "onClick" | "onPointerDown" | "onKeyDown"> & {
  /** Runs once the button has been held down until it fills. */
  onDone: () => void;
  /** How long the hold takes, in milliseconds. */
  ms?: number;
  /** What the fill warns of: warning for a stop, success for something finished. */
  tone?: HoldTone;
  /** The action has happened: the fill stays full, the button no longer holds, and it shows `doneLabel`. */
  done?: boolean;
  /** What a done button reads, after a check. Falls back to its children. */
  doneLabel?: ReactNode;
  /** Says "hold" beside the label, and "keep holding" once the press has started. */
  hint?: boolean;
};

/**
 * A button that acts only once held down until it fills, for an action worth a
 * moment's pause. Releasing, leaving or Escape before it fills cancels it. Enter
 * or Space held on the focused button works the same way.
 *
 * The fill is a CSS transition over the hold's length, not a per-frame state, so
 * a hold renders the button twice — at its start and its end — however long it is.
 */
export function HoldButton({
  onDone,
  ms = 900,
  tone,
  done = false,
  doneLabel,
  hint = false,
  disabled,
  className,
  children,
  ...props
}: HoldButtonProps) {
  const t = useT();
  const [holding, setHolding] = useState(false);
  const timer = useRef<number | null>(null);
  const doneRef = useRef(onDone);
  useEffect(() => {
    doneRef.current = onDone;
  });

  const stop = useCallback(() => {
    if (timer.current !== null) {
      window.clearTimeout(timer.current);
      timer.current = null;
    }
    setHolding(false);
  }, []);

  const start = useCallback(() => {
    if (disabled || done) return;
    stop();
    setHolding(true);
    timer.current = window.setTimeout(() => {
      timer.current = null;
      setHolding(false);
      doneRef.current();
    }, ms);
  }, [disabled, done, ms, stop]);

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
      aria-disabled={done || undefined}
      data-holding={holding || undefined}
      data-done={done || undefined}
      className={cn(holdButtonVariants({ tone }), className)}
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
          holdFillVariants({ tone }),
          holding || done ? "scale-x-100" : "scale-x-0 duration-0",
        )}
        style={holding ? { transitionDuration: `${ms}ms` } : undefined}
      />
      <span className="relative flex items-center justify-center gap-2">
        {done ? (
          <>
            <CheckIcon className="size-3.5" />
            {doneLabel ?? children}
          </>
        ) : (
          <>
            {children}
            {hint && (
              <em className="text-muted-foreground text-xs not-italic">
                {holding ? t("keep holding") : t("hold")}
              </em>
            )}
          </>
        )}
      </span>
    </Button>
  );
}
