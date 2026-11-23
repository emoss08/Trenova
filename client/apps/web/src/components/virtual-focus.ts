"use no memo";
import type { FocusEvent, KeyboardEvent, RefObject } from "react";
import { useRef } from "react";

/*
 * Keeping keyboard focus alive in a windowed list. A windowed list unmounts the rows that scroll
 * out of view, and a focused element that unmounts drops focus to the page. These pieces keep the
 * focused row in the window wherever it scrolls, and move focus into a row that is not mounted
 * by scrolling it into the window first.
 */

const ROW = "[data-index]";
const FOCUSABLE =
  'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])';

/** The index of the windowed row an element sits in, or null. */
function rowIndexOf(element: Element | null): number | null {
  const row = element?.closest(ROW);
  if (!row) return null;
  const index = Number(row.getAttribute("data-index"));
  return Number.isInteger(index) ? index : null;
}

/** The row element itself when it takes focus, else the first focusable thing in it. */
function focusTargetIn(row: Element): HTMLElement | null {
  if (row instanceof HTMLElement && row.tabIndex >= 0) return row;
  return row.querySelector<HTMLElement>(FOCUSABLE);
}

/**
 * Which row holds focus, by its key, for a range extractor to keep mounted. Spread `handlers` on
 * the scroll container; read `focusedKey.current` inside the range extractor.
 */
export function useFocusedRow(keyAt: (index: number) => string | undefined) {
  const focusedKey = useRef<string | null>(null);
  const handlers = {
    onFocusCapture: (event: FocusEvent<HTMLElement>) => {
      const index = rowIndexOf(event.target as Element);
      focusedKey.current = index === null ? null : (keyAt(index) ?? null);
    },
    onBlurCapture: (event: FocusEvent<HTMLElement>) => {
      const next = event.relatedTarget as Element | null;
      if (!next || !event.currentTarget.contains(next)) focusedKey.current = null;
    },
  };
  return { focusedKey, handlers };
}

/** `indexes` with `extra` added in order, for a range extractor; unchanged when null or present. */
export function withIndex(indexes: number[], extra: number | null): number[] {
  if (extra === null || extra < 0 || indexes.includes(extra)) return indexes;
  return [...indexes, extra].sort((a, b) => a - b);
}

/** Focuses row `index` as soon as it is mounted (it may take a frame or two after a scroll). */
function focusRowWhenMounted(container: HTMLElement, index: number, attempts = 6) {
  const row = container.querySelector(`${ROW.slice(0, -1)}="${index}"]`);
  const target = row ? focusTargetIn(row) : null;
  if (target) {
    target.focus({ preventScroll: false });
    return;
  }
  if (attempts > 0) {
    requestAnimationFrame(() => focusRowWhenMounted(container, index, attempts - 1));
  }
}

/**
 * Arrow Up and Down move focus to the row above or below, and Tab or Shift+Tab out of the last or
 * first mounted row goes on to the next row rather than leaving the list: a row that is not
 * mounted is scrolled into the window first, then focused. Spread on the scroll container of a
 * list whose rows take focus.
 */
export function rowNavigationKeyDown({
  count,
  containerRef,
  scrollToIndex,
}: {
  count: number;
  containerRef: RefObject<HTMLElement | null>;
  scrollToIndex: (index: number) => void;
}) {
  return (event: KeyboardEvent<HTMLElement>) => {
    const container = containerRef.current;
    if (!container || event.defaultPrevented || event.altKey || event.ctrlKey || event.metaKey) {
      return;
    }
    const target = event.target as Element;
    const index = rowIndexOf(target);
    if (index === null) return;
    const row = target.closest(ROW);
    if (!row) return;

    const typing = target.matches(
      'input, textarea, select, [role="combobox"], [contenteditable="true"]',
    );
    let next: number | null = null;
    if ((event.key === "ArrowDown" || event.key === "ArrowUp") && typing) return;
    if (event.key === "ArrowDown") next = index + 1;
    else if (event.key === "ArrowUp") next = index - 1;
    else if (event.key === "Tab") {
      const focusables = [...row.querySelectorAll<HTMLElement>(FOCUSABLE)];
      if (row instanceof HTMLElement && row.tabIndex >= 0) focusables.unshift(row);
      const position = focusables.indexOf(target as HTMLElement);
      const leaving = event.shiftKey ? position <= 0 : position === focusables.length - 1;
      if (!leaving) return;
      const neighbour = event.shiftKey ? index - 1 : index + 1;
      // A neighbour already in the DOM is the browser's own Tab; only an unmounted one is ours.
      if (container.querySelector(`${ROW.slice(0, -1)}="${neighbour}"]`)) return;
      next = neighbour;
    } else {
      return;
    }
    if (next < 0 || next >= count) return;
    event.preventDefault();
    scrollToIndex(next);
    focusRowWhenMounted(container, next);
  };
}
