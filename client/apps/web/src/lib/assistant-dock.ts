export const ASSISTANT_DOCKS = ["bottom-right", "bottom-left", "top-right", "top-left"] as const;

export type AssistantDock = (typeof ASSISTANT_DOCKS)[number];

export const DEFAULT_ASSISTANT_DOCK: AssistantDock = "bottom-right";

export type AssistantPanelSize = { width: number; height: number };

export const DEFAULT_PANEL_SIZE: AssistantPanelSize = { width: 380, height: 560 };
export const MIN_PANEL_SIZE: AssistantPanelSize = { width: 320, height: 360 };

/** One keyboard press on the resize handle. */
export const PANEL_RESIZE_STEP = 24;

export function isAssistantDock(value: unknown): value is AssistantDock {
  return typeof value === "string" && (ASSISTANT_DOCKS as readonly string[]).includes(value);
}

export function isPanelSize(value: unknown): value is AssistantPanelSize {
  if (typeof value !== "object" || value === null) {
    return false;
  }
  const { width, height } = value as Record<string, unknown>;

  return (
    typeof width === "number" &&
    typeof height === "number" &&
    Number.isFinite(width) &&
    Number.isFinite(height)
  );
}

export function isTopDock(dock: AssistantDock): boolean {
  return dock === "top-right" || dock === "top-left";
}

export function isLeftDock(dock: AssistantDock): boolean {
  return dock === "bottom-left" || dock === "top-left";
}

/**
 * The corner nearest a point, which is where a dragged launcher lands. The
 * viewport is split in half each way, so a drop anywhere in a quadrant means
 * that quadrant's corner.
 */
export function nearestDock(
  point: { x: number; y: number },
  viewport: { width: number; height: number },
): AssistantDock {
  const top = point.y < viewport.height / 2;
  const left = point.x < viewport.width / 2;
  if (top) {
    return left ? "top-left" : "top-right";
  }

  return left ? "bottom-left" : "bottom-right";
}

/**
 * Fixed-position classes for something anchored in a corner, written out in
 * full so the stylesheet contains them. The top corners sit below the app
 * header rather than over it.
 */
const DOCK_POSITION_CLASSES: Record<"launcher" | "panel" | "tab", Record<AssistantDock, string>> = {
  launcher: {
    "bottom-right": "right-5 bottom-5",
    "bottom-left": "bottom-5 left-5",
    "top-right": "top-14 right-5",
    "top-left": "top-14 left-5",
  },
  panel: {
    "bottom-right": "right-4 bottom-4",
    "bottom-left": "bottom-4 left-4",
    "top-right": "top-14 right-4",
    "top-left": "top-14 left-4",
  },
  tab: {
    "bottom-right": "right-0 bottom-5",
    "bottom-left": "bottom-5 left-0",
    "top-right": "top-14 right-0",
    "top-left": "top-14 left-0",
  },
};

export function dockPositionClass(
  dock: AssistantDock,
  surface: "launcher" | "panel" | "tab",
): string {
  return DOCK_POSITION_CLASSES[surface][dock];
}

/**
 * The panel's size, never smaller than it can be used at and never larger
 * than the window it sits in.
 */
export function clampPanelSize(
  size: AssistantPanelSize,
  viewport: { width: number; height: number },
): AssistantPanelSize {
  const maxWidth = Math.max(MIN_PANEL_SIZE.width, viewport.width - 32);
  const maxHeight = Math.max(MIN_PANEL_SIZE.height, viewport.height - 72);

  return {
    width: Math.round(Math.min(Math.max(size.width, MIN_PANEL_SIZE.width), maxWidth)),
    height: Math.round(Math.min(Math.max(size.height, MIN_PANEL_SIZE.height), maxHeight)),
  };
}

/**
 * The panel is anchored in its corner, so its handle is on the opposite
 * corner and dragging away from the anchor makes it bigger: left for a panel
 * on the right, down for a panel at the top.
 */
export function resizePanel(
  dock: AssistantDock,
  start: AssistantPanelSize,
  delta: { x: number; y: number },
): AssistantPanelSize {
  const widthDelta = isLeftDock(dock) ? delta.x : -delta.x;
  const heightDelta = isTopDock(dock) ? delta.y : -delta.y;

  return { width: start.width + widthDelta, height: start.height + heightDelta };
}

/**
 * What an arrow key does to the panel from its handle. The arrows point the
 * way the handle moves, the same as dragging it, so the rule is the drag's.
 */
export function resizePanelByKey(
  dock: AssistantDock,
  start: AssistantPanelSize,
  key: string,
): AssistantPanelSize | null {
  const step = PANEL_RESIZE_STEP;
  switch (key) {
    case "ArrowLeft":
      return resizePanel(dock, start, { x: -step, y: 0 });
    case "ArrowRight":
      return resizePanel(dock, start, { x: step, y: 0 });
    case "ArrowUp":
      return resizePanel(dock, start, { x: 0, y: -step });
    case "ArrowDown":
      return resizePanel(dock, start, { x: 0, y: step });
    default:
      return null;
  }
}

/**
 * The panel's size as inline styles. The saved size is a preference; the
 * window it opens in may since have shrunk, so the browser caps it again.
 */
export function panelSizeStyle(
  dock: AssistantDock,
  size: AssistantPanelSize,
): { width: string; height: string } {
  const verticalMargin = isTopDock(dock) ? "4.5rem" : "2rem";

  return {
    width: `min(${size.width}px, calc(100vw - 2rem))`,
    height: `min(${size.height}px, calc(100dvh - ${verticalMargin}))`,
  };
}
