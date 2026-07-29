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
