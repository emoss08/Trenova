import { useT } from "@trenova/shared/i18n/use-t";
import { SearchLgIcon, XCloseIcon } from "@trenova/shared/components/icons";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { cn } from "@trenova/shared/lib/utils";
import { isTypingTarget, isWithinDialog } from "@/lib/dom";
import { queries } from "@/lib/queries";
import {
  addQuickFilter,
  QUICK_FILTER_MENU,
  quickFilterKey,
  quickFilterLabel,
  removeQuickFilter,
} from "@/lib/shipment-board/quick-filters";
import { useQuery } from "@tanstack/react-query";
import { useEffect, useRef, useState } from "react";
import { useBoardScope } from "../use-board-scope";
import { useShipmentBoardUrl } from "../url-state";

const SEARCH_DEBOUNCE_MS = 300;

type BoardSearchProps = {
  query: string;
  onSearchChange: (query: string) => void;
};

/**
 * The board's one search field. Quick filters sit in it as removable tokens
 * ahead of the free text, and an empty, focused field offers them with how
 * many shipments each would leave. "/" focuses it from anywhere on the page.
 */
export function BoardSearch({ query, onSearchChange }: BoardSearchProps) {
  const t = useT();
  const [{ qf }, setUrl] = useShipmentBoardUrl();
  const [text, setText] = useState(query);
  const [focused, setFocused] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const scope = useBoardScope();
  const counts = useQuery({
    ...queries.shipmentBoard.quickFilterCounts(scope),
    enabled: focused,
    staleTime: 15_000,
  });

  useEffect(() => setText(query), [query]);

  useEffect(() => {
    if (text === query) return;
    const timer = window.setTimeout(() => onSearchChange(text), SEARCH_DEBOUNCE_MS);
    return () => window.clearTimeout(timer);
  }, [text, query, onSearchChange]);

  useEffect(() => {
    const focusOnSlash = (event: KeyboardEvent) => {
      if (event.key !== "/" || event.metaKey || event.ctrlKey || event.altKey) return;
      if (isTypingTarget(event.target) || isWithinDialog(event.target)) return;
      event.preventDefault();
      inputRef.current?.focus();
    };
    window.addEventListener("keydown", focusOnSlash);
    return () => window.removeEventListener("keydown", focusOnSlash);
  }, []);

  const setQuickFilters = (next: typeof qf) => void setUrl({ qf: next, expanded: null });
  const countOf = (filter: string) =>
    counts.data?.find((entry) => entry.filter === filter)?.count;

  return (
    <div className="relative min-w-30 flex-[0_1_300px]">
      <div
        className={cn(
          "ui-field flex h-7 min-w-0 items-center gap-1 px-2",
          focused && "border-border-strong",
        )}
        onClick={() => inputRef.current?.focus()}
      >
        <SearchLgIcon className="text-muted-foreground size-3.5 shrink-0" aria-hidden />
        <div className="flex min-w-0 flex-1 items-center gap-1 overflow-hidden">
          {qf.map((token) => (
            <span
              key={quickFilterKey(token)}
              className="bg-brand-subtle text-brand-subtle-foreground inline-flex h-5 shrink-0 items-center gap-1 rounded px-1.5 text-xs font-medium"
            >
              {quickFilterLabel(token, t)}
              <button
                type="button"
                aria-label={t("Remove {0}", quickFilterLabel(token, t))}
                className="ui-focus-ring rounded-sm opacity-70 hover:opacity-100"
                onClick={(event) => {
                  event.stopPropagation();
                  setQuickFilters(removeQuickFilter(qf, token));
                }}
              >
                <XCloseIcon className="size-3" />
              </button>
            </span>
          ))}
          <input
            ref={inputRef}
            value={text}
            aria-label={t("Search shipments")}
            placeholder={qf.length ? t("Add filter") : t("Search shipments…")}
            className="placeholder:text-muted-foreground min-w-12 flex-1 bg-transparent text-sm outline-none"
            onChange={(event) => setText(event.target.value)}
            onFocus={() => setFocused(true)}
            onBlur={() => window.setTimeout(() => setFocused(false), 120)}
            onKeyDown={(event) => {
              if (event.key === "Enter") onSearchChange(text);
              if (event.key === "Backspace" && !text && qf.length) setQuickFilters(qf.slice(0, -1));
              if (event.key === "Escape") event.currentTarget.blur();
            }}
          />
        </div>
        {qf.length || text ? (
          <button
            type="button"
            aria-label={t("Clear search")}
            className="ui-focus-ring text-muted-foreground hover:text-foreground grid size-5 shrink-0 place-items-center rounded-sm"
            onClick={(event) => {
              event.stopPropagation();
              setText("");
              onSearchChange("");
              setQuickFilters([]);
            }}
          >
            <XCloseIcon className="size-3" />
          </button>
        ) : (
          <Kbd>/</Kbd>
        )}
      </div>
      {focused && !text ? (
        <div
          role="listbox"
          aria-label={t("Quick filters")}
          className="dark bg-popover text-popover-foreground animate-pop-in ring-foreground/10 absolute top-[calc(100%+4px)] left-0 z-40 w-60 rounded-lg p-1 ring-1"
        >
          <div className="text-muted-foreground px-2 py-1 text-xs">{t("Quick filters")}</div>
          {QUICK_FILTER_MENU.map((entry) => (
            <button
              key={entry.filter}
              type="button"
              role="option"
              aria-selected={qf.some((token) => token.filter === entry.filter)}
              className="hover:bg-surface-hover flex h-7 w-full items-center gap-2 rounded-md px-2 text-left text-sm"
              onMouseDown={(event) => event.preventDefault()}
              onClick={() => setQuickFilters(addQuickFilter(qf, { filter: entry.filter }))}
            >
              <span aria-hidden className={cn("size-1.5 rounded-full", entry.dotClassName)} />
              <span className="flex-1">{t(entry.label)}</span>
              <span className="text-muted-foreground font-mono text-xs tabular-nums">
                {countOf(entry.filter)?.toLocaleString() ?? ""}
              </span>
            </button>
          ))}
        </div>
      ) : null}
    </div>
  );
}
