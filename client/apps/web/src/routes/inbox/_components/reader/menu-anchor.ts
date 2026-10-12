import { openContextMenu } from "@trenova/shared/components/context-menu/context-menu-host";
import type { ContextMenuSpec } from "@trenova/shared/components/context-menu/menu-spec";

/** The menu's narrowest width, so an end-aligned menu never starts past its button. */
const MENU_MIN_WIDTH = 208;

/**
 * Opens the one shared menu from a button, as a dropdown: under it (or above
 * it, for a bar at the foot of the screen), lined up with its start or its
 * end. The menu flips at the viewport's edges on its own; focus returns to the
 * button when it closes.
 */
export function openMenuFrom(
  button: HTMLElement,
  spec: ContextMenuSpec,
  { align = "start", side = "below" }: { align?: "start" | "end"; side?: "below" | "above" } = {},
): void {
  const rect = button.getBoundingClientRect();
  openContextMenu(
    {
      x: align === "start" ? rect.left : Math.max(0, rect.right - MENU_MIN_WIDTH),
      y: side === "below" ? rect.bottom + 4 : rect.top - 4,
    },
    spec,
    button,
  );
}
