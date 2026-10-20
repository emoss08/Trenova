import { clampInteger, cn } from "@trenova/shared/lib/utils";
import {
  useRef,
  useState,
  type KeyboardEvent as ReactKeyboardEvent,
  type PointerEvent as ReactPointerEvent,
} from "react";

/** How far one arrow key moves the edge. */
const KEY_STEP = 16;

/** How far past its narrowest a panel that folds is dragged before letting go folds it. */
export const PANEL_FOLD_DRAG = 72;

export type PanelResizeHandleProps = {
  /** The panel's width now, in pixels. */
  width: number;
  min: number;
  max: number;
  /** What a double click puts the width back to. */
  defaultWidth: number;
  onResize: (width: number) => void;
  /**
   * Folds the panel away. Given it, dragging the edge well past the panel's
   * narrowest and letting go folds it, at the width it had before the drag,
   * and Enter on the focused edge folds it too.
   */
  onFold?: () => void;
  label: string;
  className?: string;
};

type DragOrigin = { x: number; width: number };

/**
 * A panel's right edge, which drags. It is a line the pointer can take hold
 * of, with a little more to grab than it shows; a double click puts the width
 * back, and the arrow keys, Home and End move it from the keyboard. While a
 * fold is on offer it carries `data-folding`, so the panel can show it.
 */
export function PanelResizeHandle({
  width,
  min,
  max,
  defaultWidth,
  onResize,
  onFold,
  label,
  className,
}: PanelResizeHandleProps) {
  const [dragging, setDragging] = useState(false);
  const [folding, setFolding] = useState(false);
  const origin = useRef<DragOrigin | null>(null);

  const onPointerDown = (event: ReactPointerEvent<HTMLDivElement>) => {
    if (event.button !== 0) return;
    event.preventDefault();
    origin.current = { x: event.clientX, width };
    event.currentTarget.setPointerCapture(event.pointerId);
    setDragging(true);
  };

  const onPointerMove = (event: ReactPointerEvent<HTMLDivElement>) => {
    const start = origin.current;
    if (!start) return;
    const next = start.width + event.clientX - start.x;
    setFolding(onFold !== undefined && next < min - PANEL_FOLD_DRAG);
    onResize(clampInteger(next, min, max, defaultWidth));
  };

  const endDrag = (event: ReactPointerEvent<HTMLDivElement>, fold: boolean) => {
    const start = origin.current;
    if (!start) return;
    origin.current = null;
    if (event.currentTarget.hasPointerCapture(event.pointerId)) {
      event.currentTarget.releasePointerCapture(event.pointerId);
    }
    setDragging(false);
    setFolding(false);
    if (fold && onFold) {
      onResize(start.width);
      onFold();
    }
  };

  const onKeyDown = (event: ReactKeyboardEvent<HTMLDivElement>) => {
    const next =
      event.key === "ArrowLeft"
        ? width - KEY_STEP
        : event.key === "ArrowRight"
          ? width + KEY_STEP
          : event.key === "Home"
            ? min
            : event.key === "End"
              ? max
              : null;
    if (next !== null) {
      event.preventDefault();
      onResize(clampInteger(next, min, max, defaultWidth));
    } else if (event.key === "Enter" && onFold) {
      event.preventDefault();
      onFold();
    }
  };

  return (
    <div
      role="separator"
      aria-orientation="vertical"
      aria-label={label}
      aria-valuemin={min}
      aria-valuemax={max}
      aria-valuenow={width}
      tabIndex={0}
      data-dragging={dragging || undefined}
      data-folding={folding || undefined}
      onPointerDown={onPointerDown}
      onPointerMove={onPointerMove}
      onPointerUp={(event) => endDrag(event, folding)}
      onPointerCancel={(event) => endDrag(event, false)}
      onDoubleClick={() => onResize(defaultWidth)}
      onKeyDown={onKeyDown}
      className={cn(
        "ui-focus-ring group absolute inset-y-0 -right-1 z-10 w-2 cursor-col-resize touch-none outline-none select-none",
        className,
      )}
    >
      <span
        aria-hidden
        className={cn(
          "absolute inset-y-0 left-[3px] w-0.5 transition-colors",
          dragging ? "bg-border-strong" : "bg-transparent group-hover:bg-border-strong",
        )}
      />
    </div>
  );
}
