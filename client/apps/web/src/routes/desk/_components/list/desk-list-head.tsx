import { DeskIcon } from "@/components/desk-chat/desk-icons";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useLayoutEffect, useRef, useState, type ComponentProps, type ReactNode } from "react";

/**
 * The head of a list pane: what the list is, the views on it, a search over
 * it and what it is narrowed to. Each part is one row, and a page stacks the
 * ones it has. The parts differ on purpose, so what they do is plain at a
 * glance: a view is a quiet word that holds the active surface while it is
 * chosen, and a filter is a bordered token that is taken off.
 */
export function DeskListHead({ className, ...props }: ComponentProps<"header">) {
  return <header className={cn("flex flex-none flex-col pb-2", className)} {...props} />;
}

/** The list's name, with what sits beside it (a scope) and what trails it (how far along). */
export function DeskListTitle({
  title,
  children,
  trailing,
}: {
  title: string;
  /** Beside the name: whose list it is. */
  children?: ReactNode;
  /** At the row's end: progress, a count. */
  trailing?: ReactNode;
}) {
  return (
    <div className="flex items-center gap-2.5 px-4 pt-4 pb-2.5">
      <h2 className="flex-none text-base font-semibold text-dsk-fg">{title}</h2>
      {children}
      {trailing !== undefined && (
        <span className="ml-auto min-w-0 truncate text-xs text-dsk-subtle">{trailing}</span>
      )}
    </div>
  );
}

/** How much of the list's work is behind the person, as a hairline under its name. */
export function DeskListProgress({
  label,
  done,
  total,
}: {
  label: string;
  done: number;
  total: number;
}) {
  return (
    <div
      className="mx-4 h-0.5 overflow-hidden rounded-[2px] bg-dsk-b-sub"
      role="progressbar"
      aria-label={label}
      aria-valuemin={0}
      aria-valuemax={total}
      aria-valuenow={Math.min(done, total)}
    >
      <i
        className="block h-full origin-left bg-dsk-fg2 transition-transform duration-400 ease-settle"
        style={{ transform: `scaleX(${total === 0 ? 0 : Math.min(1, done / total)})` }}
      />
    </div>
  );
}

export type DeskListTab<TValue extends string> = {
  value: TValue;
  label: string;
  /** What the view holds. */
  count?: number;
  /** The key that picks it, for the tooltip. */
  hint?: string;
};

type TabMark = { left: number; width: number };

/**
 * The views on a list, each with what it holds. One marker holds the Desk's
 * active surface, as the chosen row under it does, and glides to the view
 * that is picked rather than being lit on each: the views are one control,
 * and the page under it changes in place. What trails the tabs (the filter)
 * keeps to the row's end. `onIntent` hears that a view is about to be picked
 * (the pointer is on it, or the focus), so a page can read it ahead.
 */
export function DeskListTabs<TValue extends string>({
  label,
  tabs,
  value,
  onChange,
  onIntent,
  children,
}: {
  /** What the tabs choose between, for assistive tech. */
  label: string;
  tabs: readonly DeskListTab<TValue>[];
  value: TValue;
  onChange: (value: TValue) => void;
  onIntent?: (value: TValue) => void;
  children?: ReactNode;
}) {
  const listRef = useRef<HTMLDivElement>(null);
  const [mark, setMark] = useState<TabMark | null>(null);
  // What decides where the marker sits: the view in front, and every tab's
  // words, since a count that grows moves the tabs after it.
  const layoutKey = tabs.map((tab) => `${tab.value}:${tab.label}:${tab.count ?? ""}`).join("|");

  useLayoutEffect(() => {
    const list = listRef.current;
    if (list === null) {
      return;
    }
    const measure = () => {
      const chosen = list.querySelector<HTMLElement>('[role="tab"][aria-selected="true"]');
      setMark((current) => {
        if (chosen === null || chosen.offsetWidth === 0) {
          return null;
        }
        const next = { left: chosen.offsetLeft, width: chosen.offsetWidth };
        return current?.left === next.left && current.width === next.width ? current : next;
      });
    };
    measure();
    if (typeof ResizeObserver === "undefined") {
      return;
    }
    // A tab's width also changes without a render: a font arriving, a pane
    // shown again after a narrow window hid it.
    const observer = new ResizeObserver(measure);
    for (const tab of list.querySelectorAll('[role="tab"]')) {
      observer.observe(tab);
    }
    return () => observer.disconnect();
  }, [layoutKey, value]);

  return (
    <div className="flex items-center gap-2 pt-2.5 pr-3 pl-2">
      <div
        ref={listRef}
        role="tablist"
        aria-label={label}
        className="relative flex min-w-0 flex-1 [scrollbar-width:none] gap-0.5 overflow-x-auto"
      >
        {mark !== null && (
          <span
            aria-hidden
            data-slot="desk-list-tab-mark"
            className="pointer-events-none absolute top-0 left-0 h-6.5 rounded-md bg-(--dsk-active) transition-[translate,width] duration-200 ease-settle in-[.dk-calm]:transition-none motion-reduce:transition-none"
            style={{ width: mark.width, translate: `${mark.left}px 0` }}
          />
        )}
        {tabs.map((tab) => (
          <button
            key={tab.value}
            type="button"
            role="tab"
            aria-selected={tab.value === value}
            title={tab.hint}
            onClick={() => onChange(tab.value)}
            onPointerEnter={onIntent && (() => onIntent(tab.value))}
            onFocus={onIntent && (() => onIntent(tab.value))}
            className={cn(
              "relative flex h-6.5 flex-none items-center gap-1.5 rounded-md ui-focus-ring px-2 text-sm whitespace-nowrap transition-colors",
              tab.value === value ? "font-medium text-dsk-fg" : "text-dsk-subtle hover:text-dsk-fg",
            )}
          >
            {tab.label}
            {tab.count !== undefined && (
              <span
                className={cn(
                  "text-xs font-normal tabular-nums",
                  tab.value === value ? "text-dsk-subtle" : "text-dsk-faint",
                )}
              >
                {tab.count}
              </span>
            )}
          </button>
        ))}
      </div>
      {children}
    </div>
  );
}

/** A choice between two or three ways to read the list: whose it is, the order it is in. */
export function DeskListToggle<TValue extends string>({
  label,
  options,
  value,
  onChange,
}: {
  /** What is being chosen, for assistive tech. */
  label: string;
  options: readonly { value: TValue; label: string }[];
  value: TValue;
  onChange: (value: TValue) => void;
}) {
  return (
    <div role="group" aria-label={label} className="flex flex-none gap-0.5">
      {options.map((option) => (
        <button
          key={option.value}
          type="button"
          aria-pressed={option.value === value}
          onClick={() => onChange(option.value)}
          className={cn(
            "h-5.5 rounded-badge ui-focus-ring px-2 text-xs transition-colors",
            option.value === value
              ? "bg-surface-hover text-foreground"
              : "text-foreground-subtle hover:text-foreground",
          )}
        >
          {option.label}
        </button>
      ))}
    </div>
  );
}

/**
 * The classes of the filter's button. What can be filtered is the page's to
 * say, so the page brings the menu and its trigger wears these; with filters
 * on it holds the active surface, as a chosen view does.
 */
export function deskListFilterClass(active: boolean): string {
  return cn(
    // i18n-ignore: class names, never shown to a person
    "flex h-6.5 flex-none items-center gap-1.5 rounded-md ui-focus-ring px-2 text-sm transition-colors data-popup-open:bg-dsk-hover data-popup-open:text-dsk-fg",
    active
      ? "bg-(--dsk-active) font-medium text-dsk-fg"
      : "text-dsk-subtle hover:bg-dsk-hover hover:text-dsk-fg",
  );
}

/** What the filter's button says: the mark, the word, and how many filters are on. */
export function DeskListFilterLabel({ count }: { count: number }) {
  const t = useT();

  return (
    <>
      <DeskIcon name="filter" size={13} />
      {t("Filter")}
      {count > 0 && (
        <span className="text-xs font-normal text-dsk-subtle tabular-nums">{count}</span>
      )}
    </>
  );
}

export type DeskListToken = {
  key: string;
  label: string;
  /** A mark before the label: a severity's square, an unopened row's dot. */
  mark?: ReactNode;
  onRemove: () => void;
};

/**
 * What the list is narrowed to, one token per filter, each taken off on its
 * own, and all of them at once. A page with nothing filtered draws no row.
 */
export function DeskListTokens({
  tokens,
  onClear,
}: {
  tokens: readonly DeskListToken[];
  onClear: () => void;
}) {
  const t = useT();
  if (tokens.length === 0) {
    return null;
  }

  return (
    <div
      role="group"
      aria-label={t("Filters")}
      className="flex flex-wrap items-center gap-1 px-3 pt-2"
    >
      {tokens.map((token) => (
        <span
          key={token.key}
          className="flex h-6 min-w-0 items-center gap-1.5 rounded-md border border-dsk-b bg-dsk-card pr-0.5 pl-2 text-sm text-dsk-fg2"
        >
          {token.mark}
          <span className="min-w-0 truncate">{token.label}</span>
          <button
            type="button"
            aria-label={t("Remove the filter {0}", token.label)}
            onClick={token.onRemove}
            className="flex size-5 flex-none items-center justify-center rounded-sm ui-focus-ring text-dsk-faint hover:bg-dsk-hover hover:text-dsk-fg"
          >
            <DeskIcon name="x" size={10} stroke={2} />
          </button>
        </span>
      ))}
      <button
        type="button"
        onClick={onClear}
        className="ml-auto h-6 flex-none rounded-md ui-focus-ring px-1.5 text-xs text-dsk-subtle hover:text-dsk-fg"
      >
        {t("Clear")}
      </button>
    </div>
  );
}
