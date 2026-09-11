import { useT } from "@trenova/shared/i18n/use-t";
import type {
  HomeWidgetCatalog,
  HomeWidgetCategory,
  HomeWidgetOption,
} from "@/lib/graphql/home-layout";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Input } from "@trenova/shared/components/ui/input";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { ScrollArea } from "@trenova/shared/components/ui/scroll-area";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { cn } from "@trenova/shared/lib/utils";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { CheckIcon, PlusIcon, SearchIcon, XIcon } from "lucide-react";
import type React from "react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { widgetVisualFor, WidgetSketch } from "./widget-gallery-visuals";

/** How long a card keeps its "Added" confirmation before settling back. */
const ADDED_FEEDBACK_MS = 1200;

const ALL_CATEGORIES = "__all__";

export type AddWidgetDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  catalog: HomeWidgetCatalog | undefined;
  loading: boolean;
  /** How many of each widget the canvas already carries, keyed by widget key. */
  usedCounts: Map<string, number>;
  /** Widgets currently on the canvas. */
  used: number;
  /** The most widgets this home screen may hold. */
  max: number;
  onAdd: (option: HomeWidgetOption) => void;
};

type CategoryTab = {
  key: string;
  label: string;
  description: string;
  count: number;
};

type FocusDirection = "next" | "previous" | "up" | "down";

const ARROW_DIRECTIONS: Record<string, FocusDirection | undefined> = {
  ArrowRight: "next",
  ArrowLeft: "previous",
  ArrowDown: "down",
  ArrowUp: "up",
};

/**
 * How many cards share a row with the one at `index`, read off their laid-out
 * positions. Cards in the same CSS grid row share an offsetTop, and a section
 * heading between two groups restarts the count — which is what makes an
 * arrow-down from the last row of Work land on the first row of Pulse.
 */
function columnsAround(cards: HTMLButtonElement[], index: number): number {
  const top = cards[index]?.offsetTop;
  if (top == null) return 1;

  let columns = 0;
  for (const card of cards) {
    if (card.offsetTop === top) columns += 1;
  }
  return Math.max(columns, 1);
}

function matches(option: HomeWidgetOption, term: string): boolean {
  if (term === "") return true;
  return (
    option.label.toLowerCase().includes(term) ||
    option.description.toLowerCase().includes(term) ||
    option.key.toLowerCase().includes(term)
  );
}

/**
 * The widget gallery. It is a browsing surface rather than a list: a person
 * adding a tile is deciding what their morning looks like, and the thing that
 * decision turns on — is this a queue, a number, or a picture — is carried by
 * the silhouette on each card, not by its name.
 *
 * Adding a widget that needs no configuration leaves the gallery open, because
 * the common case is picking three queues in a row; a widget that cannot draw
 * anything until it is told what to show hands over to its editor instead, and
 * the caller closes the gallery for that.
 */
export function AddWidgetDialog({ open, onOpenChange, ...gallery }: AddWidgetDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="flex h-[min(40rem,88vh)] w-full flex-col gap-0 overflow-hidden p-0 sm:max-w-4xl"
        showCloseButton={false}
      >
        {/* The body holds the search term and the open category. It lives one
            level down so that closing the dialog, which unmounts the portal,
            is what clears them — a filter left over from the last visit reads
            as an empty catalog rather than as a filter still in force. */}
        <WidgetGallery {...gallery} onClose={() => onOpenChange(false)} />
      </DialogContent>
    </Dialog>
  );
}

type WidgetGalleryProps = Omit<AddWidgetDialogProps, "open" | "onOpenChange"> & {
  onClose: () => void;
};

function WidgetGallery({
  catalog,
  loading,
  usedCounts,
  used,
  max,
  onAdd,
  onClose,
}: WidgetGalleryProps) {
  const t = useT();

  const [search, setSearch] = useState("");
  const [category, setCategory] = useState(ALL_CATEGORIES);
  const [justAdded, setJustAdded] = useState<string | null>(null);

  const searchRef = useRef<HTMLInputElement>(null);
  const gridRef = useRef<HTMLDivElement>(null);
  const addedTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const remaining = Math.max(max - used, 0);
  const full = remaining <= 0;
  const term = search.trim().toLowerCase();

  useEffect(
    () => () => {
      if (addedTimer.current) clearTimeout(addedTimer.current);
    },
    [],
  );

  const widgets = useMemo(() => catalog?.widgets ?? [], [catalog?.widgets]);

  const tabs = useMemo<CategoryTab[]>(() => {
    const searched = widgets.filter((widget) => matches(widget, term));
    const categories: HomeWidgetCategory[] = catalog?.categories ?? [];

    return [
      {
        key: ALL_CATEGORIES,
        label: "All widgets",
        description: "Everything you can put on this home screen",
        count: searched.length,
      },
      ...categories.map((entry) => ({
        key: entry.key,
        label: entry.label,
        description: entry.description,
        count: searched.filter((widget) => widget.category === entry.key).length,
      })),
    ];
  }, [catalog?.categories, widgets, term]);

  // A search that empties the open category would otherwise read as "no widgets
  // exist". Widening to All is derived rather than stored, so clearing the
  // search puts the person back in the category they chose.
  const activeCategory = useMemo(() => {
    if (category === ALL_CATEGORIES) return ALL_CATEGORIES;
    const tab = tabs.find((entry) => entry.key === category);
    return tab && tab.count > 0 ? category : ALL_CATEGORIES;
  }, [tabs, category]);

  const groups = useMemo(() => {
    const categories: HomeWidgetCategory[] = catalog?.categories ?? [];
    const visible = categories.filter(
      (entry) => activeCategory === ALL_CATEGORIES || entry.key === activeCategory,
    );

    return visible
      .map((entry) => ({
        category: entry,
        widgets: widgets.filter((widget) => widget.category === entry.key && matches(widget, term)),
      }))
      .filter((group) => group.widgets.length > 0);
  }, [catalog?.categories, widgets, activeCategory, term]);

  const resultCount = groups.reduce((total, group) => total + group.widgets.length, 0);

  const add = useCallback(
    (option: HomeWidgetOption) => {
      if (full) return;
      onAdd(option);
      setJustAdded(option.key);
      if (addedTimer.current) clearTimeout(addedTimer.current);
      addedTimer.current = setTimeout(() => setJustAdded(null), ADDED_FEEDBACK_MS);
    },
    [full, onAdd],
  );

  /**
   * Moves focus between cards so the grid is usable without a pointer. A row is
   * measured rather than assumed: the grid is one, two, or three columns
   * depending on the width, so a fixed vertical step lands on the wrong card at
   * every size but the widest.
   */
  const moveFocus = useCallback((from: HTMLElement, direction: FocusDirection) => {
    const grid = gridRef.current;
    if (!grid) return;

    const cards = Array.from(grid.querySelectorAll<HTMLButtonElement>("[data-widget-card]"));
    const index = cards.indexOf(from as HTMLButtonElement);
    if (index === -1) return;

    const step =
      direction === "next"
        ? 1
        : direction === "previous"
          ? -1
          : columnsAround(cards, index) * (direction === "down" ? 1 : -1);

    cards[Math.min(Math.max(index + step, 0), cards.length - 1)]?.focus();
  }, []);

  const focusFirstCard = useCallback(() => {
    gridRef.current?.querySelector<HTMLButtonElement>("[data-widget-card]")?.focus();
  }, []);

  return (
    <>
      <GalleryHeader used={used} max={max} onClose={onClose} />

      <div className="border-border/70 flex items-center gap-2 border-b px-4 py-2.5">
        <Input
          ref={searchRef}
          autoFocus
          aria-label={t("Search widgets")}
          placeholder={t("Search widgets by name or what they show…")}
          value={search}
          onChange={(event) => setSearch(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === "ArrowDown") {
              event.preventDefault();
              focusFirstCard();
            }
          }}
          leftElement={<SearchIcon className="text-muted-foreground size-3.5" />}
          inputContainerClassName="flex-1"
          className="h-8"
          rightElement={
            search ? (
              <Button
                variant="ghost"
                size="icon-sm"
                aria-label={t("Clear search")}
                className="size-6"
                onClick={() => {
                  setSearch("");
                  searchRef.current?.focus();
                }}
              >
                <XIcon className="size-3.5" />
              </Button>
            ) : undefined
          }
        />
        {term !== "" && (
          <span className="text-2xs text-muted-foreground shrink-0 tabular-nums">
            {resultCount} {resultCount === 1 ? "match" : "matches"}
          </span>
        )}
      </div>

      <div className="flex min-h-0 flex-1">
        <CategoryRail tabs={tabs} value={activeCategory} onChange={setCategory} />

        <ScrollArea className="min-h-0 flex-1" viewportClassName="px-4 py-3.5">
          {loading ? (
            <GallerySkeleton />
          ) : resultCount === 0 ? (
            <GalleryEmpty
              search={search}
              onClear={() => {
                setSearch("");
                setCategory(ALL_CATEGORIES);
                searchRef.current?.focus();
              }}
            />
          ) : (
            <div ref={gridRef} className="flex flex-col gap-5">
              {groups.map(({ category: entry, widgets: options }) => (
                <section key={entry.key} className="flex flex-col gap-2">
                  <header className="flex items-baseline gap-2">
                    <h3 className="cc-label text-foreground">{entry.label}</h3>
                    <p className="text-2xs text-muted-foreground truncate">{entry.description}</p>
                  </header>
                  <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
                    {options.map((option) => (
                      <WidgetCard
                        key={option.key}
                        option={option}
                        onCanvas={usedCounts.get(option.key) ?? 0}
                        added={justAdded === option.key}
                        disabled={full}
                        onAdd={() => add(option)}
                        onMove={moveFocus}
                      />
                    ))}
                  </div>
                </section>
              ))}
            </div>
          )}
        </ScrollArea>
      </div>

      <GalleryFooter full={full} remaining={remaining} onDone={onClose} />
    </>
  );
}

function GalleryHeader({ used, max, onClose }: { used: number; max: number; onClose: () => void }) {
  const t = useT();

  return (
    <DialogHeader className="border-border/70 flex-row items-start gap-4 border-b px-4 py-3">
      <div className="flex min-w-0 flex-1 flex-col gap-1">
        <DialogTitle>{t("Add a widget")}</DialogTitle>
        <DialogDescription className="text-xs">
          {t("Pick what this home screen opens on. Everything here is already scoped to what you are allowed to see.")}
        </DialogDescription>
      </div>
      <SlotMeter used={used} max={max} />
      <Button variant="ghost" size="icon-sm" aria-label={t("Close")} onClick={onClose}>
        <XIcon />
      </Button>
    </DialogHeader>
  );
}

/**
 * How much of the home screen is spoken for. The limit is the server's, and a
 * person who has nearly reached it should see that before they pick, not after
 * the card they wanted refuses to be added.
 */
function SlotMeter({ used, max }: { used: number; max: number }) {
  const t = useT();

  const share = max > 0 ? Math.min(used / max, 1) : 0;
  const nearlyFull = max - used <= 3;

  return (
    <div className="hidden shrink-0 flex-col items-end gap-1.5 pt-0.5 sm:flex">
      <span className="text-2xs text-muted-foreground tabular-nums">
        <span
          className={cn("font-medium", nearlyFull ? "text-warning-foreground" : "text-foreground")}
        >
          {used}
        </span>{t("of {0} widgets", max)}
      </span>
      <span className="bg-muted h-1 w-28 overflow-hidden rounded-full">
        <span
          className={cn(
            "block h-full rounded-full transition-[width] duration-300",
            nearlyFull ? "bg-warning" : "bg-brand",
          )}
          style={{ width: `${share * 100}%` }}
        />
      </span>
    </div>
  );
}

function CategoryRail({
  tabs,
  value,
  onChange,
}: {
  tabs: CategoryTab[];
  value: string;
  onChange: (key: string) => void;
}) {
  const t = useT();

  return (
    <nav
      aria-label={t("Widget categories")}
      className="border-border/70 bg-muted/25 hidden w-44 shrink-0 flex-col gap-0.5 border-r p-2 sm:flex"
    >
      {tabs.map((tab) => {
        const active = tab.key === value;
        return (
          <button
            key={tab.key}
            type="button"
            aria-current={active ? "true" : undefined}
            disabled={tab.count === 0}
            onClick={() => onChange(tab.key)}
            title={tab.description}
            className={cn(
              "flex items-center gap-2 rounded-md px-2 py-1.5 text-left text-xs transition-colors",
              active
                ? "bg-background text-foreground ring-border/70 font-medium ring-1"
                : "text-muted-foreground hover:bg-background/60 hover:text-foreground",
              tab.count === 0 && "pointer-events-none opacity-40",
            )}
          >
            <span className="min-w-0 flex-1 truncate">{tab.label}</span>
            <span
              className={cn(
                "shrink-0 text-[10px] tabular-nums",
                active ? "text-muted-foreground" : "text-muted-foreground/70",
              )}
            >
              {tab.count}
            </span>
          </button>
        );
      })}
    </nav>
  );
}

function WidgetCard({
  option,
  onCanvas,
  added,
  disabled,
  onAdd,
  onMove,
}: {
  option: HomeWidgetOption;
  onCanvas: number;
  added: boolean;
  disabled: boolean;
  onAdd: () => void;
  onMove: (from: HTMLElement, direction: FocusDirection) => void;
}) {
  const t = useT();

  const { icon: Icon, shape } = widgetVisualFor(option.key);
  const needsSetup = option.configKind !== "none" && option.configKind !== "queue";

  const handleKeyDown = (event: React.KeyboardEvent<HTMLButtonElement>) => {
    const direction = ARROW_DIRECTIONS[event.key];
    if (!direction) return;
    event.preventDefault();
    onMove(event.currentTarget, direction);
  };

  return (
    <Tooltip delay={500}>
      <TooltipTrigger
        render={
          <button
            type="button"
            data-widget-card
            disabled={disabled}
            onClick={onAdd}
            onKeyDown={handleKeyDown}
            className={cn(
              "group/widget-card border-border bg-card relative flex flex-col gap-2 rounded-lg border p-2.5 text-left transition-all",
              "focus-visible:ring-brand/40 focus-visible:border-brand focus-visible:ring-4 focus-visible:outline-none",
              disabled
                ? "cursor-not-allowed opacity-45"
                : "hover:border-brand/45 hover:bg-accent/30 hover:shadow-sm",
              added && "border-success/60 bg-success/5",
            )}
          />
        }
      >
        <span className="relative block">
          <WidgetSketch shape={shape} />
          <AddAffordance added={added} hidden={disabled} />
        </span>

        <span className="flex min-w-0 items-start gap-2">
          <span className="bg-muted text-muted-foreground group-hover/widget-card:bg-brand/10 group-hover/widget-card:text-brand mt-px flex size-6 shrink-0 items-center justify-center rounded-md transition-colors">
            <Icon className="size-3.5" />
          </span>
          <span className="flex min-w-0 flex-1 flex-col gap-0.5">
            <span className="flex min-w-0 items-center gap-1.5">
              <span className="min-w-0 flex-1 truncate text-xs font-medium">{option.label}</span>
              {onCanvas > 0 && (
                <Badge
                  variant="outline"
                  className="border-border/70 h-4 shrink-0 border px-1 text-[9px]"
                >
                  {onCanvas > 1 ? `${onCanvas}× on canvas` : "On canvas"}
                </Badge>
              )}
            </span>
            <span className="text-muted-foreground line-clamp-2 text-[11px] leading-snug">
              {option.description}
            </span>
          </span>
        </span>
      </TooltipTrigger>
      <TooltipContent side="top" className="max-w-56">
        <span className="flex flex-col gap-0.5">
          <span>
            {t("Lands {0} columns wide and {1} rows tall. Resize it on the canvas.", option.defaultW, option.defaultH)}
          </span>
          {needsSetup && (
            <span className="text-background/70">{t("You choose what it shows before it lands.")}</span>
          )}
        </span>
      </TooltipContent>
    </Tooltip>
  );
}

/**
 * The one thing the card is for, drawn over the corner of the preview so the
 * card body stays nothing but the widget's own identity. It is decoration: the
 * whole card is the button, and it carries the accessible name.
 */
function AddAffordance({ added, hidden }: { added: boolean; hidden: boolean }) {
  if (hidden) return null;

  return (
    <span
      aria-hidden
      className={cn(
        "absolute top-1 right-1 flex size-5 items-center justify-center rounded-full shadow-sm transition-all duration-150",
        added
          ? "bg-success text-background scale-100 opacity-100"
          : "bg-brand text-brand-foreground scale-90 opacity-0 group-hover/widget-card:scale-100 group-hover/widget-card:opacity-100 group-focus-visible/widget-card:scale-100 group-focus-visible/widget-card:opacity-100",
      )}
    >
      {added ? <CheckIcon className="size-3" /> : <PlusIcon className="size-3" />}
    </span>
  );
}

function GalleryFooter({
  full,
  remaining,
  onDone,
}: {
  full: boolean;
  remaining: number;
  onDone: () => void;
}) {
  const t = useT();

  return (
    <div className="border-border/70 bg-muted/40 flex items-center gap-3 border-t px-4 py-2.5">
      <p className="text-2xs text-muted-foreground min-w-0 flex-1">
        {full ? (
          <span className="text-warning-foreground">
            {t("This home screen is full. Remove a widget to make room for another.")}
          </span>
        ) : (
          <>
            {t("Room for {0} more.", remaining)}
            <span className="hidden sm:inline">
              {t("Use")} <Kbd className="h-4 px-1 text-[10px]">&darr;</Kbd> {t("to reach the cards and")}{" "}
              <Kbd className="h-4 px-1 text-[10px]">&crarr;</Kbd> {t("to add one.")}
            </span>
          </>
        )}
      </p>
      <Button size="sm" variant="outline" onClick={onDone}>
        {t("Done")}
      </Button>
    </div>
  );
}

function GalleryEmpty({ search, onClear }: { search: string; onClear: () => void }) {
  const t = useT();

  return (
    <div className="flex flex-col items-center justify-center gap-2 py-16 text-center">
      <SearchIcon className="text-muted-foreground/40 size-5" />
      <p className="text-sm font-medium">{t("Nothing matches")}</p>
      <p className="text-muted-foreground max-w-xs text-xs">
        {search.trim() === ""
          ? "No widgets are available to you in this category."
          : `No widget matches “${search.trim()}”.`}
      </p>
      <Button variant="outline" size="sm" className="mt-1" onClick={onClear}>
        {t("Clear filters")}
      </Button>
    </div>
  );
}

function GallerySkeleton() {
  return (
    <div className="flex flex-col gap-5">
      {[0, 1].map((group) => (
        <div key={group} className="flex flex-col gap-2">
          <Skeleton className="h-2.5 w-24 rounded-full" />
          <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
            {[0, 1, 2, 3, 4, 5].map((card) => (
              <Skeleton key={card} className="h-30 rounded-lg" />
            ))}
          </div>
        </div>
      ))}
    </div>
  );
}
