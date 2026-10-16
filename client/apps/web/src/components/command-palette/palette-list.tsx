import { Button } from "@trenova/shared/components/ui/button";
import { CommandGroup, CommandList } from "@trenova/shared/components/ui/command";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { ChevronRightIcon, RefreshCw01Icon } from "@trenova/shared/components/icons";
import { useLayoutEffect, useRef } from "react";
import type { PaletteItem, PaletteScope, PaletteSection } from "./palette-model";
import { PaletteRow } from "./palette-row";

function SectionHeading({
  section,
  onSeeAll,
}: {
  section: PaletteSection;
  onSeeAll: (scope: PaletteScope) => void;
}) {
  const t = useT();
  const count =
    section.count ?? (section.loading || section.error ? undefined : section.items.length);
  return (
    <span className="flex h-7.5 items-center gap-1.5">
      <span className="truncate">{section.heading}</span>
      {count !== undefined && count > 1 && (
        <span className="text-foreground-faint tabular-nums">{count.toLocaleString()}</span>
      )}
      {section.seeAll && count !== undefined && (
        <button
          type="button"
          tabIndex={-1}
          onMouseDown={(event) => event.preventDefault()}
          onClick={() => section.seeAll && onSeeAll(section.seeAll)}
          className="text-foreground-subtle hover:text-foreground ml-auto flex items-center gap-0.5 font-sans text-xs normal-case transition-colors"
        >
          {t("See all {0}", count.toLocaleString())}
          <ChevronRightIcon className="size-3" />
        </button>
      )}
    </span>
  );
}

function SkeletonRows({ rows }: { rows: number }) {
  return (
    <div aria-hidden className="flex flex-col">
      {Array.from({ length: rows }, (_, index) => (
        <div key={index} className="flex min-h-11.5 items-center gap-3 px-2.5 py-1.5">
          <Skeleton className="rounded-control size-7 shrink-0" />
          <div className="flex flex-1 flex-col gap-1.5">
            <Skeleton className="h-3 w-2/5" />
            <Skeleton className="h-2.5 w-3/5" />
          </div>
        </div>
      ))}
    </div>
  );
}

function rowFor(list: HTMLElement, key: string): HTMLElement | null {
  for (const row of list.querySelectorAll<HTMLElement>("[data-palette-row]")) {
    if (row.dataset.paletteKey === key) {
      return row;
    }
  }
  return null;
}

/**
 * Keeps one fill under whichever row is selected and slides it there, rather
 * than painting each row, so moving through the list reads as one thing
 * moving. It slides only when the selection moves through a list that stayed
 * put; when the list itself changes (a key typed, a scope picked, a section
 * loading above, an answer streaming in) the fill lands under its row at once,
 * since a slide there would be the fill chasing rows that moved without it.
 *
 * The row is found by its key rather than by cmdk's selected mark, which cmdk
 * writes after this effect runs in the commit that changes the selection.
 */
function useSelectionFill(
  listRef: React.RefObject<HTMLDivElement | null>,
  fillRef: React.RefObject<HTMLDivElement | null>,
  selectedKey: string,
  sections: readonly PaletteSection[],
) {
  const placed = useRef<{
    sections: readonly PaletteSection[] | null;
    key: string;
    top: number;
    height: number;
  }>({ sections: null, key: "", top: Number.NaN, height: Number.NaN });

  useLayoutEffect(() => {
    const list = listRef.current;
    const fill = fillRef.current;
    if (!list || !fill) {
      return;
    }

    const place = (slide: boolean) => {
      const last = placed.current;
      const row = rowFor(list, selectedKey);
      if (!row) {
        fill.style.opacity = "0";
        placed.current = { sections, key: selectedKey, top: Number.NaN, height: Number.NaN };
        return;
      }
      const listBox = list.getBoundingClientRect();
      const rowBox = row.getBoundingClientRect();
      const top = rowBox.top - listBox.top;
      if (fill.style.opacity === "1" && top === last.top && rowBox.height === last.height) {
        return;
      }
      if (!slide) {
        fill.style.transition = "none";
      }
      fill.style.opacity = "1";
      fill.style.transform = `translateY(${top}px)`;
      fill.style.height = `${rowBox.height}px`;
      if (!slide) {
        void fill.offsetHeight;
        fill.style.transition = "";
      }
      placed.current = { sections, key: selectedKey, top, height: rowBox.height };
    };

    const listChanged = placed.current.sections !== sections;
    const keyChanged = placed.current.key !== selectedKey;
    place(!listChanged && keyChanged);
    if (listChanged && keyChanged) {
      rowFor(list, selectedKey)?.scrollIntoView?.({ block: "nearest" });
    }

    const observer = new ResizeObserver(() => place(false));
    observer.observe(list);
    const sizer = list.firstElementChild;
    if (sizer) {
      observer.observe(sizer);
    }
    return () => observer.disconnect();
  }, [fillRef, listRef, sections, selectedKey]);
}

/**
 * The result list, section by section, under whatever sits at its head (an
 * answer, an empty state). Loading and failure are drawn inside the section
 * they belong to, so one slow or failing source never blanks the rest.
 */
export function PaletteList({
  sections,
  selectedKey,
  onSelect,
  onRetry,
  onSeeAll,
  head,
}: {
  sections: readonly PaletteSection[];
  selectedKey: string;
  onSelect: (item: PaletteItem) => void;
  onRetry: () => void;
  onSeeAll: (scope: PaletteScope) => void;
  head?: React.ReactNode;
}) {
  const t = useT();
  const listRef = useRef<HTMLDivElement | null>(null);
  const fillRef = useRef<HTMLDivElement | null>(null);
  useSelectionFill(listRef, fillRef, selectedKey, sections);

  return (
    <ScrollArea className="min-h-0 flex-1" maskHeight={0}>
      <CommandList ref={listRef} className="relative max-h-none overflow-visible px-2 pt-1 pb-2">
        <div
          ref={fillRef}
          aria-hidden
          className="bg-surface-hover rounded-surface ease-settle pointer-events-none absolute inset-x-2 top-0 z-0 opacity-0 transition-[transform,height] duration-170 motion-reduce:transition-none"
        />
        {head}
        {sections.map((section, index) => (
          <CommandGroup
            key={section.id}
            heading={<SectionHeading section={section} onSeeAll={onSeeAll} />}
            className={
              "p-0 [&_[cmdk-group-heading]]:text-foreground-faint [&_[cmdk-group-heading]]:px-2.5 [&_[cmdk-group-heading]]:py-0 [&_[cmdk-group-heading]]:font-mono [&_[cmdk-group-heading]]:text-2xs [&_[cmdk-group-heading]]:font-normal [&_[cmdk-group-heading]]:uppercase" +
              (index === 0 && !head ? "" : " [&_[cmdk-group-heading]]:pt-1")
            }
          >
            {section.items.map((item) => (
              <PaletteRow key={item.key} item={item} onSelect={onSelect} />
            ))}
            {section.loading && section.items.length === 0 && (
              <SkeletonRows rows={section.placeholderRows ?? 3} />
            )}
            {section.error && (
              <div className="rounded-surface border-border-subtle mx-0.5 flex items-center justify-between gap-3 border px-3 py-2 text-sm">
                <span className="text-foreground-muted">
                  {t("Record search is unavailable right now.")}
                </span>
                <Button
                  type="button"
                  size="xs"
                  variant="ghost"
                  tabIndex={-1}
                  onMouseDown={(event) => event.preventDefault()}
                  onClick={onRetry}
                >
                  <RefreshCw01Icon className="size-3" />
                  {t("Try again")}
                </Button>
              </div>
            )}
          </CommandGroup>
        ))}
      </CommandList>
    </ScrollArea>
  );
}
