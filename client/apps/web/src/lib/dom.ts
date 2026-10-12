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
 * True while the event originated inside a floating list (a command menu, a listbox or a
 * popover), which owns its own number and Enter keys.
 */
export function isWithinPopup(target: EventTarget | null): boolean {
  return (
    target instanceof Element &&
    target.closest(
      '[cmdk-root], [role="listbox"], [role="dialog"], [data-slot="popover-content"]',
    ) !== null
  );
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

/**
 * True when a window-level key belongs to `scope`: it was pressed with focus on nothing,
 * or on something inside the scope. A key pressed in another pane is that pane's.
 */
export function isKeyForScope(target: EventTarget | null, scope: Element | null): boolean {
  if (target === document.body) {
    return true;
  }
  return scope !== null && target instanceof Node && scope.contains(target);
}

/**
 * Whether focus may be moved within `scope` without interrupting the person: it is on
 * nothing, or inside the scope and not in a field holding words they are typing.
 */
export function mayMoveFocusWithin(scope: Element | null): boolean {
  const active = document.activeElement;
  if (active === null || active === document.body) {
    return true;
  }
  if (scope === null || !scope.contains(active)) {
    return false;
  }
  return (
    !isTypingTarget(active) ||
    ((active instanceof HTMLTextAreaElement || active instanceof HTMLInputElement) &&
      active.value === "")
  );
}

/**
 * The nearest ancestor of `element` that scrolls vertically, or null when only the page
 * does. An ancestor whose overflow lets it scroll counts whether or not it overflows now,
 * since a pane about to hold taller content is still the one that will scroll; pass
 * `overflowing` to skip one that has nothing to scroll yet.
 */
export function scrollParent(
  element: Element | null,
  { overflowing = false }: { overflowing?: boolean } = {},
): HTMLElement | null {
  let candidate = element?.parentElement ?? null;
  while (candidate && candidate !== document.body) {
    const scrolls = /auto|scroll/u.test(getComputedStyle(candidate).overflowY);
    if (scrolls && (!overflowing || candidate.scrollHeight > candidate.clientHeight)) {
      return candidate;
    }
    candidate = candidate.parentElement;
  }
  return null;
}
