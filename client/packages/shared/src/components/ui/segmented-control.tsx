import { cn } from "@trenova/shared/lib/utils";
import { m } from "motion/react";
import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
  type ComponentType,
  type KeyboardEvent,
  type ReactNode,
} from "react";

export type SegmentedControlItem<TValue extends string> = {
  value: TValue;
  label: ReactNode;
  /** Secondary line rendered under the label — a metric, count, or hint. */
  caption?: ReactNode;
  icon?: ComponentType<{ className?: string }>;
  disabled?: boolean;
};

type SegmentedControlProps<TValue extends string> = {
  items: SegmentedControlItem<TValue>[];
  value: TValue;
  onValueChange: (value: TValue) => void;
  /** Stretches the segments to fill the available width. */
  fullWidth?: boolean;
  "aria-label"?: string;
  className?: string;
};

const INDICATOR_TRANSITION = {
  type: "spring",
  stiffness: 420,
  damping: 38,
  mass: 0.6,
} as const;

/**
 * A radio group rendered as a single track with a sliding indicator.
 *
 * The indicator is measured rather than laid out so it survives the panel being
 * mounted while hidden — a tab that is `display: none` reports zero-width
 * children, and the resize observer re-measures the moment it is revealed.
 */
export function SegmentedControl<TValue extends string>({
  items,
  value,
  onValueChange,
  fullWidth = false,
  className,
  ...props
}: SegmentedControlProps<TValue>) {
  const trackRef = useRef<HTMLDivElement>(null);
  const segmentRefs = useRef(new Map<string, HTMLButtonElement>());
  const [indicator, setIndicator] = useState<{ left: number; width: number } | null>(null);

  const measure = useCallback(() => {
    const segment = segmentRefs.current.get(value);

    if (!segment || segment.offsetWidth === 0) {
      return;
    }

    setIndicator((current) =>
      current?.left === segment.offsetLeft && current.width === segment.offsetWidth
        ? current
        : { left: segment.offsetLeft, width: segment.offsetWidth },
    );
  }, [value]);

  useLayoutEffect(() => {
    measure();
  }, [measure, items]);

  useEffect(() => {
    const track = trackRef.current;

    if (!track || typeof ResizeObserver === "undefined") {
      return;
    }

    const observer = new ResizeObserver(measure);
    observer.observe(track);

    for (const segment of segmentRefs.current.values()) {
      observer.observe(segment);
    }

    return () => observer.disconnect();
  }, [measure, items]);

  const focusSegment = useCallback(
    (index: number) => {
      const enabled = items.filter((item) => !item.disabled);

      if (enabled.length === 0) {
        return;
      }

      const next = enabled[((index % enabled.length) + enabled.length) % enabled.length];
      segmentRefs.current.get(next.value)?.focus();
      onValueChange(next.value);
    },
    [items, onValueChange],
  );

  const handleKeyDown = useCallback(
    (event: KeyboardEvent<HTMLDivElement>) => {
      const enabled = items.filter((item) => !item.disabled);
      const current = enabled.findIndex((item) => item.value === value);

      switch (event.key) {
        case "ArrowLeft":
        case "ArrowUp":
          event.preventDefault();
          focusSegment(current - 1);
          break;
        case "ArrowRight":
        case "ArrowDown":
          event.preventDefault();
          focusSegment(current + 1);
          break;
        case "Home":
          event.preventDefault();
          focusSegment(0);
          break;
        case "End":
          event.preventDefault();
          focusSegment(enabled.length - 1);
          break;
        default:
          break;
      }
    },
    [focusSegment, items, value],
  );

  return (
    <div
      ref={trackRef}
      role="radiogroup"
      aria-label={props["aria-label"]}
      onKeyDown={handleKeyDown}
      className={cn(
        "relative isolate flex items-stretch gap-0.5 rounded-lg bg-muted/60 p-0.5",
        fullWidth ? "w-full" : "w-fit",
        className,
      )}
    >
      {indicator && (
        <m.div
          aria-hidden
          className="absolute inset-y-0.5 left-0 -z-10 rounded-md bg-background shadow-xs ring-1 ring-border/60"
          initial={false}
          animate={{ x: indicator.left, width: indicator.width }}
          transition={INDICATOR_TRANSITION}
        />
      )}

      {items.map((item) => {
        const isActive = item.value === value;
        const Icon = item.icon;

        return (
          <button
            key={item.value}
            ref={(node) => {
              if (node) {
                segmentRefs.current.set(item.value, node);
              } else {
                segmentRefs.current.delete(item.value);
              }
            }}
            type="button"
            role="radio"
            aria-checked={isActive}
            tabIndex={isActive ? 0 : -1}
            disabled={item.disabled}
            onClick={() => onValueChange(item.value)}
            className={cn(
              "flex min-w-0 flex-col items-center justify-center rounded-md px-2.5 py-1 text-xs font-medium transition-colors",
              "focus-visible:ring-ring/60 outline-none focus-visible:ring-2",
              "disabled:pointer-events-none disabled:opacity-50",
              fullWidth && "flex-1",
              isActive ? "text-foreground" : "text-muted-foreground hover:text-foreground",
            )}
          >
            <span className="flex min-w-0 items-center gap-1.5">
              {Icon && <Icon className="size-3.5 shrink-0" />}
              <span className="truncate">{item.label}</span>
            </span>
            {item.caption !== undefined && (
              <span
                className={cn(
                  "mt-0.5 truncate text-2xs tabular-nums transition-colors",
                  isActive ? "text-muted-foreground" : "text-muted-foreground/60",
                )}
              >
                {item.caption}
              </span>
            )}
          </button>
        );
      })}
    </div>
  );
}
