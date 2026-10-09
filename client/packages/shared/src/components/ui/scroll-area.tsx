"use client";

import * as React from "react";

import { ScrollArea as ScrollAreaPrimitive } from "@base-ui/react/scroll-area";

import { cn } from "@trenova/shared/lib/utils";

import { useTouchPrimary } from "@trenova/shared/components/ui/use-has-primary-touch";

type Mask = {
  top: boolean;
  bottom: boolean;
  left: boolean;
  right: boolean;
};

export type ScrollAreaMaskVariant =
  | "background"
  | "card"
  | "field"
  | "muted"
  | "popover"
  | "sidebar";

export type ScrollAreaContextProps = {
  isTouch: boolean;
  type: "auto" | "always" | "scroll" | "hover";
};

const ScrollAreaContext = React.createContext<ScrollAreaContextProps>({
  isTouch: false,
  type: "hover",
});

const scrollMaskVariantClassNames: Record<ScrollAreaMaskVariant, string> = {
  background: "before:from-background after:from-background",
  card: "before:from-card after:from-card",
  field: "before:from-field after:from-field",
  muted: "before:from-muted after:from-muted",
  popover: "before:from-popover after:from-popover",
  sidebar: "before:from-sidebar after:from-sidebar",
};

/** How far a mouse must move before a press becomes a drag rather than a click. */
const DRAG_THRESHOLD = 4;

type DragState = {
  pointerId: number;
  x: number;
  y: number;
  left: number;
  top: number;
  moved: boolean;
};

/**
 * Press-and-drag scrolling for a mouse. It moves the viewport directly and marks it
 * with a data attribute, so a drag never re-renders anything; a drag that moved
 * swallows the click that ends it, so dragging never presses what it started on.
 */
function useDragToScroll(
  viewportRef: React.RefObject<HTMLDivElement | null>,
  enabled: boolean,
) {
  const state = React.useRef<DragState | null>(null);
  const swallowClick = React.useRef(false);

  const handlers = React.useMemo(() => {
    const end = () => {
      const drag = state.current;
      const element = viewportRef.current;
      state.current = null;
      if (!drag?.moved || !element) return;
      delete element.dataset.dragging;
      if (element.hasPointerCapture(drag.pointerId)) {
        element.releasePointerCapture(drag.pointerId);
      }
      swallowClick.current = true;
      window.setTimeout(() => {
        swallowClick.current = false;
      }, 0);
    };

    return {
      onPointerDown(event: React.PointerEvent<HTMLDivElement>) {
        const element = viewportRef.current;
        if (!enabled || !element || event.pointerType !== "mouse" || event.button !== 0) return;
        state.current = {
          pointerId: event.pointerId,
          x: event.clientX,
          y: event.clientY,
          left: element.scrollLeft,
          top: element.scrollTop,
          moved: false,
        };
      },
      onPointerMove(event: React.PointerEvent<HTMLDivElement>) {
        const drag = state.current;
        const element = viewportRef.current;
        if (!drag || !element || event.pointerId !== drag.pointerId) return;
        const dx = event.clientX - drag.x;
        const dy = event.clientY - drag.y;
        if (!drag.moved) {
          if (Math.abs(dx) < DRAG_THRESHOLD && Math.abs(dy) < DRAG_THRESHOLD) return;
          drag.moved = true;
          element.setPointerCapture(drag.pointerId);
          element.dataset.dragging = "";
        }
        element.scrollLeft = drag.left - dx;
        element.scrollTop = drag.top - dy;
      },
      onPointerUp: end,
      onPointerCancel: end,
      onClickCapture(event: React.MouseEvent<HTMLDivElement>) {
        if (!swallowClick.current) return;
        swallowClick.current = false;
        event.preventDefault();
        event.stopPropagation();
      },
    };
  }, [enabled, viewportRef]);

  return { handlers };
}

const ScrollArea = React.forwardRef<
  React.ComponentRef<typeof ScrollAreaPrimitive.Root>,
  React.ComponentPropsWithoutRef<typeof ScrollAreaPrimitive.Root> & {
    type?: "auto" | "always" | "scroll" | "hover";
    viewportClassName?: string;
    /**
     * `maskHeight` is the height of the mask in pixels.
     * pass `0` to disable the mask
     * @default 30
     */
    maskHeight?: number;
    maskClassName?: string;
    /**
     * Background token used by the scroll mask fade.
     * @default "background"
     */
    maskVariant?: ScrollAreaMaskVariant;
    /**
     * Lets a mouse scroll the area by pressing and dragging it, as touch already
     * does; a click that does not move still reaches what was clicked.
     */
    dragToScroll?: boolean;
  }
>(
  (
    {
      className,
      children,
      type = "hover",
      maskHeight = 30,
      maskClassName,
      maskVariant = "background",
      viewportClassName,
      dragToScroll = false,
      style,
      ...props
    },
    ref,
  ) => {
    const [showMask, setShowMask] = React.useState<Mask>({
      top: false,
      bottom: false,
      left: false,
      right: false,
    });

    const viewportRef = React.useRef<HTMLDivElement>(null);
    const drag = useDragToScroll(viewportRef, dragToScroll);
    const isTouch = useTouchPrimary();
    const touchStyle = typeof style === "function" ? undefined : style;

    const checkScrollability = React.useCallback(() => {
      const element = viewportRef.current;
      if (!element) return;

      const { scrollTop, scrollLeft, scrollWidth, clientWidth, scrollHeight, clientHeight } =
        element;
      setShowMask((prev) => ({
        ...prev,
        top: scrollTop > 0,
        bottom: scrollTop + clientHeight < scrollHeight - 1,
        left: scrollLeft > 0,
        right: scrollLeft + clientWidth < scrollWidth - 1,
      }));
    }, []);

    React.useEffect(() => {
      if (typeof window === "undefined") return;

      const element = viewportRef.current;
      if (!element) return;

      const controller = new AbortController();
      const { signal } = controller;

      const resizeObserver = new ResizeObserver(checkScrollability);
      resizeObserver.observe(element);

      element.addEventListener("scroll", checkScrollability, {
        passive: true,
        signal,
      });
      window.addEventListener("resize", checkScrollability, {
        passive: true,
        signal,
      });

      // Run an initial check whenever dependencies change (including pointer mode)
      checkScrollability();

      return () => {
        controller.abort();
        resizeObserver.disconnect();
      };
    }, [checkScrollability, isTouch]);

    return (
      <ScrollAreaContext.Provider value={{ isTouch, type }}>
        {isTouch ? (
          <div
            ref={ref}
            {...props}
            style={touchStyle}
            role="group"
            data-slot="scroll-area"
            aria-roledescription="scroll area"
            className={cn("relative overflow-hidden", className)}
          >
            <div
              ref={viewportRef}
              className={cn("size-full overflow-auto", viewportClassName)}
              tabIndex={0}
            >
              {children}
            </div>
            {maskHeight > 0 && (
              <ScrollMask
                showMask={showMask}
                className={maskClassName}
                maskHeight={maskHeight}
                variant={maskVariant}
              />
            )}
          </div>
        ) : (
          <ScrollAreaPrimitive.Root
            ref={ref}
            data-slot="scroll-area"
            className={cn("relative overflow-hidden", viewportClassName, className)}
            {...props}
          >
            <ScrollAreaPrimitive.Viewport
              ref={viewportRef}
              data-slot="scroll-area-viewport"
              className={cn(
                "ui-focus-ring size-full rounded-[inherit]",
                dragToScroll && "cursor-grab data-dragging:cursor-grabbing data-dragging:select-none",
                viewportClassName,
              )}
              {...(dragToScroll ? drag.handlers : undefined)}
            >
              {children}
            </ScrollAreaPrimitive.Viewport>
            <ScrollBar />
            <ScrollAreaPrimitive.Corner />
            {maskHeight > 0 && (
              <ScrollMask
                showMask={showMask}
                className={maskClassName}
                maskHeight={maskHeight}
                variant={maskVariant}
              />
            )}
          </ScrollAreaPrimitive.Root>
        )}
      </ScrollAreaContext.Provider>
    );
  },
);

ScrollArea.displayName = ScrollAreaPrimitive.Root.displayName;

const ScrollBar = React.forwardRef<
  React.ComponentRef<typeof ScrollAreaPrimitive.Scrollbar>,
  React.ComponentPropsWithoutRef<typeof ScrollAreaPrimitive.Scrollbar>
>(({ className, orientation = "vertical", ...props }, ref) => {
  const { isTouch, type } = React.useContext(ScrollAreaContext);

  if (isTouch) return null;

  return (
    <ScrollAreaPrimitive.Scrollbar
      ref={ref}
      orientation={orientation}
      data-slot="scroll-area-scrollbar"
      className={cn(
        "flex touch-none p-px transition-[colors,opacity] duration-150 ease-out select-none hover:bg-muted dark:hover:bg-muted/50",
        orientation === "vertical" && "h-full w-2.5 border-l border-l-transparent",
        orientation === "horizontal" && "h-2.5 flex-col border-t border-t-transparent px-1 pr-1.25",
        type === "hover" && "opacity-0 data-[hovering]:opacity-100",
        type === "scroll" && "opacity-0 data-[scrolling]:opacity-100",
        className,
      )}
      {...props}
    >
      <ScrollAreaPrimitive.Thumb
        data-slot="scroll-area-thumb"
        className={cn(
          "relative flex-1 rounded-full bg-border transition-[scale]",
          orientation === "vertical" && "my-1 active:scale-y-95",
          orientation === "horizontal" && "active:scale-x-98",
        )}
      />
    </ScrollAreaPrimitive.Scrollbar>
  );
});

ScrollBar.displayName = ScrollAreaPrimitive.Scrollbar.displayName;

const ScrollMask = ({
  showMask,
  maskHeight,
  variant,
  className,
  ...props
}: React.ComponentProps<"div"> & {
  showMask: Mask;
  maskHeight: number;
  variant: ScrollAreaMaskVariant;
}) => {
  const variantClassName = scrollMaskVariantClassNames[variant];

  return (
    <>
      <div
        {...props}
        aria-hidden="true"
        style={
          {
            "--top-fade-height": showMask.top ? `${maskHeight}px` : "0px",
            "--bottom-fade-height": showMask.bottom ? `${maskHeight}px` : "0px",
          } as React.CSSProperties
        }
        className={cn(
          "pointer-events-none absolute inset-0 z-10",
          "before:absolute before:inset-x-0 before:top-0 before:transition-[height,opacity] before:duration-300 before:content-['']",
          "after:absolute after:inset-x-0 after:bottom-0 after:transition-[height,opacity] after:duration-300 after:content-['']",
          "before:h-(--top-fade-height) after:h-(--bottom-fade-height)",
          showMask.top ? "before:opacity-100" : "before:opacity-0",
          showMask.bottom ? "after:opacity-100" : "after:opacity-0",
          "before:bg-gradient-to-b before:to-transparent",
          "after:bg-gradient-to-t after:to-transparent",
          variantClassName,
          className,
        )}
      />
      <div
        {...props}
        aria-hidden="true"
        style={
          {
            "--left-fade-width": showMask.left ? `${maskHeight}px` : "0px",
            "--right-fade-width": showMask.right ? `${maskHeight}px` : "0px",
          } as React.CSSProperties
        }
        className={cn(
          "pointer-events-none absolute inset-0 z-10",
          "before:absolute before:inset-y-0 before:left-0 before:transition-[width,opacity] before:duration-300 before:content-['']",
          "after:absolute after:inset-y-0 after:right-0 after:transition-[width,opacity] after:duration-300 after:content-['']",
          "before:w-(--left-fade-width) after:w-(--right-fade-width)",
          showMask.left ? "before:opacity-100" : "before:opacity-0",
          showMask.right ? "after:opacity-100" : "after:opacity-0",
          "before:bg-gradient-to-r before:to-transparent",
          "after:bg-gradient-to-l after:to-transparent",
          variantClassName,
          className,
        )}
      />
    </>
  );
};

export { ScrollArea, ScrollBar };
