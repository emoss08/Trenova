"use client";

import { Slider as SliderPrimitive } from "@base-ui/react/slider";
import type { ReactNode } from "react";

import { cn } from "@trenova/shared/lib/utils";

export type SliderMark = { value: number; label: ReactNode };

type SliderProps = Omit<SliderPrimitive.Root.Props<number>, "children"> & {
  /** Labels drawn under the track at their values, on the same scale the thumb moves along. */
  marks?: readonly SliderMark[];
};

function percentOf(value: number, min: number, max: number) {
  return max === min ? 0 : ((value - min) / (max - min)) * 100;
}

/**
 * One value on a track. The thumb's centre sits on its value, and each mark is centred on its own,
 * so a label reads exactly where the thumb stops for it.
 */
function Slider({ className, marks, min = 0, max = 100, ...props }: SliderProps) {
  return (
    <SliderPrimitive.Root
      data-slot="slider"
      min={min}
      max={max}
      className={cn("flex w-full flex-col gap-1 px-2 data-disabled:opacity-50", className)}
      {...props}
    >
      <SliderPrimitive.Control className="relative flex h-4 w-full cursor-pointer touch-none items-center select-none data-disabled:cursor-not-allowed">
        <SliderPrimitive.Track className="bg-border-strong relative h-1 w-full rounded-full">
          <SliderPrimitive.Indicator className="bg-foreground rounded-full" />
          {marks
            ?.filter((mark) => mark.value > min && mark.value < max)
            .map((mark) => (
              <span
                key={mark.value}
                aria-hidden
                className="bg-card absolute top-1/2 h-1 w-px -translate-x-1/2 -translate-y-1/2"
                style={{ left: `${percentOf(mark.value, min, max)}%` }}
              />
            ))}
          <SliderPrimitive.Thumb className="ui-container-focus-ring bg-card border-foreground size-4 rounded-full border-2 transition-transform data-dragging:scale-110" />
        </SliderPrimitive.Track>
      </SliderPrimitive.Control>
      {marks?.length ? (
        <div aria-hidden className="text-foreground-subtle relative h-3.5 text-2xs tabular-nums">
          {marks.map((mark) => (
            <span
              key={mark.value}
              className="absolute top-0 -translate-x-1/2 whitespace-nowrap"
              style={{ left: `${percentOf(mark.value, min, max)}%` }}
            >
              {mark.label}
            </span>
          ))}
        </div>
      ) : null}
    </SliderPrimitive.Root>
  );
}

export { Slider };
