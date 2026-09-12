import { cn } from "@trenova/shared/lib/utils";
import { CheckIcon } from "lucide-react";
import { m, useReducedMotion } from "motion/react";
import type { ReactNode } from "react";

export type StepperStepState = "done" | "active" | "pending";

export type StepperStep = {
  id: string;
  label: ReactNode;
  detail?: ReactNode;
  state: StepperStepState;
};

type StepperProps = {
  steps: readonly StepperStep[];
  /** Whether the connectors fill in sequence on mount. Answers reduced motion regardless. */
  animate?: boolean;
  className?: string;
  "aria-label"?: string;
};

const STEP_DELAY = 0.11;

/**
 * A fixed sequence read top to bottom: what is done, what is next, what is
 * still to come. The connectors fill in order on first paint, so the eye is
 * led to the active step rather than asked to find it. One hue carries the
 * whole track; the states differ by weight and fill, not by colour.
 */
export function Stepper({
  steps,
  animate = true,
  className,
  "aria-label": ariaLabel,
}: StepperProps) {

  const reduceMotion = useReducedMotion();
  const animated = animate && !reduceMotion;

  return (
    <ol aria-label={ariaLabel} className={cn("flex flex-col", className)}>
      {steps.map((step, index) => {
        const last = index === steps.length - 1;
        const done = step.state === "done";
        const active = step.state === "active";
        return (
          <li
            key={step.id}
            data-state={step.state}
            aria-current={active ? "step" : undefined}
            className="grid grid-cols-[1rem_minmax(0,1fr)] gap-x-3"
          >
            <div className="flex flex-col items-center">
              <m.span
                aria-hidden
                initial={animated ? { scale: 0.6, opacity: 0 } : false}
                animate={{ scale: 1, opacity: 1 }}
                transition={{ duration: 0.25, delay: index * STEP_DELAY, ease: "easeOut" }}
                className={cn(
                  "mt-0.5 flex size-4 shrink-0 items-center justify-center rounded-full border",
                  done && "border-brand bg-brand text-brand-foreground",
                  active && "border-brand bg-background ring-4 ring-brand/15",
                  step.state === "pending" && "border-border bg-background",
                )}
              >
                {done ? (
                  <CheckIcon className="size-2.5" strokeWidth={3} />
                ) : active ? (
                  <span className="size-1.5 rounded-full bg-brand" />
                ) : null}
              </m.span>
              {last ? null : (
                <span aria-hidden className="relative my-1 w-px flex-1 bg-border">
                  {done ? (
                    <m.span
                      className="absolute inset-0 origin-top bg-brand"
                      initial={animated ? { scaleY: 0 } : false}
                      animate={{ scaleY: 1 }}
                      transition={{
                        duration: 0.3,
                        delay: index * STEP_DELAY + 0.1,
                        ease: "easeOut",
                      }}
                    />
                  ) : null}
                </span>
              )}
            </div>
            <div className={cn("flex min-w-0 flex-col gap-0.5", last ? "pb-0" : "pb-4")}>
              <span
                className={cn(
                  "text-xs leading-4",
                  active ? "font-medium text-foreground" : "font-medium",
                  step.state === "pending" && "text-muted-foreground",
                )}
              >
                {step.label}
              </span>
              {step.detail !== undefined ? (
                <span className="text-muted-foreground text-2xs leading-snug">{step.detail}</span>
              ) : null}
            </div>
          </li>
        );
      })}
    </ol>
  );
}
