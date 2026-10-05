const TYPING_TAGS = new Set(["INPUT", "TEXTAREA", "SELECT"]);

/**
 * Window-level hotkeys have to stay out of the way of typing. Reading the event target
 * rather than tracking focus keeps the check correct for any listener bound to the window.
 */
export function isTypingTarget(target: EventTarget | null): boolean {
  if (!(target instanceof HTMLElement)) return false;
  return TYPING_TAGS.has(target.tagName) || target.isContentEditable;
}

/** True while the event originated inside a dialog, which owns its own key handling. */
export function isWithinDialog(target: EventTarget | null): boolean {
  return target instanceof Element && target.closest('[role="dialog"]') !== null;
}

/**
 * The 0-based index of a number key from 1 to `count`, pressed without a modifier and
 * outside a text field, or -1. Pickers that answer to "press 1, 2 or 3" read it.
 */
export function numberKeyIndex(event: KeyboardEvent, count: number): number {
  if (event.metaKey || event.ctrlKey || event.altKey || isTypingTarget(event.target)) {
    return -1;
  }
  const index = Number(event.key) - 1;
  return Number.isInteger(index) && index >= 0 && index < count ? index : -1;
}
