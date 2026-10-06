import { useEffect, type KeyboardEvent as ReactKeyboardEvent, type RefObject } from "react";

const FOCUSABLE = [
  "a[href]",
  "button:not([disabled])",
  "input:not([disabled]):not([type='hidden'])",
  "select:not([disabled])",
  "textarea:not([disabled])",
  "[tabindex]:not([tabindex='-1'])",
  "[contenteditable='true']",
].join(",");

/** What Tab can reach inside `root`, in order. */
export function focusableIn(root: HTMLElement): HTMLElement[] {
  return [...root.querySelectorAll<HTMLElement>(FOCUSABLE)].filter(
    (element) =>
      !element.closest("[inert]") &&
      (typeof element.checkVisibility !== "function" || element.checkVisibility()),
  );
}

/**
 * A dialog's hold on focus: whatever had it when the dialog opened gets it
 * back when the dialog goes, and Tab and Shift+Tab go round inside it rather
 * than out to the page behind.
 */
export function useModalFocus(ref: RefObject<HTMLElement | null>, trap = true) {
  useEffect(() => {
    const previous = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    return () => {
      if (previous?.isConnected) {
        previous.focus({ preventScroll: true });
      }
    };
  }, []);

  useEffect(() => {
    const root = ref.current;
    if (!trap || !root) {
      return;
    }
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key !== "Tab" || event.defaultPrevented) {
        return;
      }
      const items = focusableIn(root);
      if (items.length === 0) {
        event.preventDefault();
        root.focus();
        return;
      }
      const first = items[0];
      const last = items[items.length - 1];
      const active = document.activeElement;
      if (event.shiftKey && (active === first || active === root || !root.contains(active))) {
        event.preventDefault();
        last.focus();
      } else if (!event.shiftKey && (active === last || !root.contains(active))) {
        event.preventDefault();
        first.focus();
      }
    };
    root.addEventListener("keydown", onKeyDown);
    return () => root.removeEventListener("keydown", onKeyDown);
  }, [ref, trap]);
}

/**
 * Arrow keys over a radio group drawn as buttons: the next or previous choice
 * is chosen and focused, wrapping at the ends, as a native radio group does.
 */
export function onRadioArrows<V>(
  event: ReactKeyboardEvent<HTMLElement>,
  values: readonly V[],
  current: V,
  choose: (value: V) => void,
) {
  const step =
    event.key === "ArrowRight" || event.key === "ArrowDown"
      ? 1
      : event.key === "ArrowLeft" || event.key === "ArrowUp"
        ? -1
        : 0;
  if (step === 0 || values.length === 0) {
    return;
  }
  event.preventDefault();
  const at = Math.max(0, values.indexOf(current));
  const next = (at + step + values.length) % values.length;
  choose(values[next]);
  const group = event.currentTarget.closest<HTMLElement>("[role='radiogroup']");
  group?.querySelectorAll<HTMLElement>("[role='radio']")[next]?.focus();
}
