import {
  clampPanelSize,
  isLeftDock,
  isTopDock,
  resizePanel,
  resizePanelByKey,
  type AssistantDock,
  type AssistantPanelSize,
} from "@/lib/assistant-dock";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useRef, type KeyboardEvent, type PointerEvent } from "react";

type AssistantResizeHandleProps = {
  dock: AssistantDock;
  size: AssistantPanelSize;
  /** Every step of a drag, so the panel follows the pointer. */
  onResize: (size: AssistantPanelSize) => void;
  /** The size to keep once the drag or key press is done. */
  onResizeEnd: (size: AssistantPanelSize) => void;
  onReset: () => void;
};

type DragStart = { pointerX: number; pointerY: number; size: AssistantPanelSize };

function viewport() {
  return { width: window.innerWidth, height: window.innerHeight };
}

/**
 * The panel's free corner. The panel is anchored in the corner the launcher
 * sits in, so it grows from the opposite one: a panel in the bottom right is
 * pulled up and to the left.
 */
export function AssistantResizeHandle({
  dock,
  size,
  onResize,
  onResizeEnd,
  onReset,
}: AssistantResizeHandleProps) {
  const t = useT();
  const start = useRef<DragStart | null>(null);
  const last = useRef(size);
  const top = isTopDock(dock);
  const left = isLeftDock(dock);
  const diagonal = top === left ? "cursor-nwse-resize" : "cursor-nesw-resize";

  const handlePointerDown = (event: PointerEvent<HTMLButtonElement>) => {
    if (event.button !== 0) {
      return;
    }
    event.preventDefault();
    event.currentTarget.setPointerCapture(event.pointerId);
    start.current = { pointerX: event.clientX, pointerY: event.clientY, size };
    last.current = size;
  };

  const handlePointerMove = (event: PointerEvent<HTMLButtonElement>) => {
    const origin = start.current;
    if (!origin) {
      return;
    }
    const next = clampPanelSize(
      resizePanel(dock, origin.size, {
        x: event.clientX - origin.pointerX,
        y: event.clientY - origin.pointerY,
      }),
      viewport(),
    );
    last.current = next;
    onResize(next);
  };

  const finish = (event: PointerEvent<HTMLButtonElement>) => {
    if (!start.current) {
      return;
    }
    start.current = null;
    if (event.currentTarget.hasPointerCapture(event.pointerId)) {
      event.currentTarget.releasePointerCapture(event.pointerId);
    }
    onResizeEnd(last.current);
  };

  const handleKeyDown = (event: KeyboardEvent<HTMLButtonElement>) => {
    const next = resizePanelByKey(dock, size, event.key);
    if (!next) {
      return;
    }
    event.preventDefault();
    onResizeEnd(clampPanelSize(next, viewport()));
  };

  return (
    <button
      type="button"
      aria-label={t("Resize the assistant. Use the arrow keys, or double-click to reset.")}
      title={t("Drag to resize, double-click to reset")}
      onPointerDown={handlePointerDown}
      onPointerMove={handlePointerMove}
      onPointerUp={finish}
      onPointerCancel={finish}
      onKeyDown={handleKeyDown}
      onDoubleClick={onReset}
      className={cn(
        "ui-focus-ring text-muted-foreground absolute z-20 flex size-4 touch-none items-center justify-center outline-none",
        "opacity-0 transition-opacity group-hover/panel:opacity-100 focus-visible:opacity-100",
        diagonal,
        top ? "bottom-0" : "top-0",
        left ? "right-0" : "left-0",
      )}
    >
      <svg viewBox="0 0 10 10" className={cn("size-2.5", resizeGlyphRotation(dock))} aria-hidden>
        <path
          d="M1 9 L9 1 M5 9 L9 5"
          stroke="currentColor"
          strokeWidth="1.2"
          strokeLinecap="round"
        />
      </svg>
    </button>
  );
}

/** The glyph's lines are drawn for the bottom-right corner and turned to face the others. */
function resizeGlyphRotation(dock: AssistantDock): string {
  switch (dock) {
    case "bottom-right":
      return "rotate-180";
    case "bottom-left":
      return "-rotate-90";
    case "top-right":
      return "rotate-90";
    case "top-left":
      return "";
  }
}
