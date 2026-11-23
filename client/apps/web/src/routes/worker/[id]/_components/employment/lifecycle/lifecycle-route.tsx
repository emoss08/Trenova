import { PlusIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useState } from "react";
import { CYCLE_STAGE_LABELS } from "../checklists/checklist-cycles";
import type { LifecycleStop } from "./lifecycle-model";

/** The stops arrive one after another this far apart, the first time the route is drawn. */
const STAGGER_MS = 140;

type LifecycleRouteProps = {
  stops: readonly LifecycleStop[];
  /** Null when the reader may not record events. */
  onRecord: (() => void) | null;
};

/**
 * The employment as stops on a route sheet: a solid line through the stops behind, a ringed
 * stop where the worker is, hollow stops and a dashed line ahead.
 */
export function LifecycleRoute({ stops, onRecord }: LifecycleRouteProps) {
  const t = useT();
  // Drawn once: after the entrance the classes go, so returning to the tab does not replay it.
  const [drawn, setDrawn] = useState(false);

  return (
    <section
      aria-label={t("Employment lifecycle")}
      className="flex flex-col gap-3 rounded-md border bg-card py-3 pr-3 pl-4 lg:flex-row lg:items-center lg:gap-6"
    >
      <ol className="grid min-w-0 flex-1 [scrollbar-width:none] auto-cols-[minmax(9rem,1fr)] grid-flow-col overflow-x-auto">
        {stops.map((stop, index) => (
          <li
            key={stop.stage}
            data-testid={`lifecycle-${stop.stage}`}
            data-state={stop.state}
            aria-current={stop.state === "current" ? "step" : undefined}
            className="flex min-w-0 flex-col pr-4"
          >
            <span aria-hidden className="relative block h-5">
              <span
                style={drawn ? undefined : { animationDelay: `${index * STAGGER_MS}ms` }}
                onAnimationEnd={index === stops.length - 1 ? () => setDrawn(true) : undefined}
                className={cn(
                  "absolute top-1/2 left-0 -translate-y-1/2 rounded-full",
                  !drawn && "animate-confirm motion-reduce:animate-none",
                  stop.state === "done" && "size-2.25 bg-foreground",
                  stop.state === "ahead" && "size-2.25 border-[1.5px] border-border-strong bg-card",
                  stop.state === "current" &&
                    "-left-px size-2.75 border-[1.5px] bg-card ring-1 ring-border ring-offset-2 ring-offset-card",
                  stop.state === "current" &&
                    (stop.danger
                      ? "border-danger-foreground ring-danger-border"
                      : "border-foreground"),
                )}
              >
                {stop.state === "current" ? (
                  <span
                    className={cn(
                      "absolute inset-0.5 rounded-full",
                      stop.danger ? "bg-danger-foreground" : "bg-foreground",
                    )}
                  />
                ) : null}
              </span>
              {index < stops.length - 1 ? (
                <span
                  className={cn(
                    "absolute top-1/2 right-1 left-3.5 h-0",
                    stop.state === "done"
                      ? "-mt-px border-t-[1.5px] border-foreground"
                      : "border-t border-dashed border-border-strong",
                  )}
                />
              ) : null}
            </span>
            <span
              className={cn(
                "mt-1 text-sm whitespace-nowrap",
                stop.state === "ahead" ? "text-muted-foreground" : "text-foreground",
                stop.state === "current" && "font-medium",
              )}
            >
              {CYCLE_STAGE_LABELS[stop.stage]}
            </span>
            <span
              className={cn(
                "truncate text-xs tabular-nums",
                stop.state === "ahead" && "text-foreground-subtle",
                stop.state !== "ahead" && "text-muted-foreground",
                stop.state === "current" && stop.danger && "text-danger-foreground",
              )}
            >
              {stop.meta}
            </span>
          </li>
        ))}
      </ol>
      {onRecord ? (
        <Button size="sm" className="self-start lg:self-center" onClick={onRecord}>
          <PlusIcon className="size-3.5" />
          {t("Record event")}
        </Button>
      ) : null}
    </section>
  );
}
