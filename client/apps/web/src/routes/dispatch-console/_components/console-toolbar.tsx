import { useT } from "@trenova/shared/i18n/use-t";
import { MultiSelectAutocomplete } from "@/components/fields/multi-select-field";
import { isTypingTarget, isWithinDialog } from "@/lib/dom";
import type {
  GraphQLSelectOptionsConfig,
  SelectOption as GraphQLSelectOption,
} from "@/lib/graphql/select-options";
import { formatRangeLabelForDays, isTodayAnchor } from "@/lib/timeline/time-scale";
import {
  CalendarIcon,
  ChevronLeftIcon,
  ChevronRightIcon,
  SearchLgIcon,
} from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { Calendar } from "@trenova/shared/components/ui/calendar";
import { Input } from "@trenova/shared/components/ui/input";
import { Kbd } from "@trenova/shared/components/ui/kbd";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { Switch } from "@trenova/shared/components/ui/switch";
import { useLocale } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useEffect, useRef, useState } from "react";
import { ConsoleStatusLine, type BoardFreshness } from "./console-status-line";
import { TIMELINE_ZOOM_OPTIONS } from "./timeline-metrics";
import { useDispatchRail, useDispatchView, useDispatchWindow } from "./url-state";

const FLEET_SELECT_OPTIONS = { resource: "FLEET_CODE" } satisfies GraphQLSelectOptionsConfig;
const CUSTOMER_SELECT_OPTIONS = { resource: "CUSTOMER" } satisfies GraphQLSelectOptionsConfig;

function selectedIds(values: readonly (string | GraphQLSelectOption)[]): string[] {
  return values
    .map((value) => (typeof value === "string" ? value : (value.id ?? "")))
    .filter(Boolean);
}

/**
 * The window the board is asked about and what narrows it. Everything here lives in the
 * URL, so the toolbar owns that state rather than being handed it; the status line on the
 * left says how fresh the answer is.
 */
export function ConsoleToolbar({ freshness }: { freshness: BoardFreshness }) {
  const t = useT();
  const locale = useLocale();
  const searchRef = useRef<HTMLInputElement>(null);

  const {
    anchor,
    zoom,
    includeCovered,
    search,
    fleetCodeIds,
    customerIds,
    setAnchor,
    shiftWindow,
    goToday,
    setZoom,
    setIncludeCovered,
    setSearch,
    setFleets,
    setCustomers,
  } = useDispatchWindow();
  const { tab } = useDispatchView();
  const { driverSearch, setDriverSearch } = useDispatchRail();
  // One search box: on the Drivers tab it finds drivers, everywhere else moves.
  const onDrivers = tab === "drivers";
  const boxSearch = onDrivers ? driverSearch : search;
  const setBoxSearch = onDrivers ? setDriverSearch : setSearch;

  // The box answers on the keystroke while the URL, and the query behind it, settles.
  const [draft, setDraft] = useState(boxSearch);
  const [prevSearch, setPrevSearch] = useState(boxSearch);
  if (prevSearch !== boxSearch) {
    setPrevSearch(boxSearch);
    setDraft(boxSearch);
  }

  // "/" reaches the search from anywhere on the console, as it does in the mockup's legend.
  useEffect(() => {
    function onKeyDown(event: KeyboardEvent) {
      if (event.key !== "/" || event.metaKey || event.ctrlKey || event.altKey) return;
      if (isTypingTarget(event.target) || isWithinDialog(event.target)) return;
      event.preventDefault();
      searchRef.current?.focus();
    }
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, []);

  return (
    // One line on a wide screen, the search giving up width first; it wraps only when it must.
    <div className="flex flex-wrap items-center gap-x-2 gap-y-1.5 border-b border-border bg-background px-4 py-2 xl:flex-nowrap">
      <ConsoleStatusLine freshness={freshness} />
      <span className="h-4 w-px shrink-0 bg-border" aria-hidden />

      <div className="flex shrink-0 items-center gap-1">
        <Button
          type="button"
          variant="ghost"
          size="icon-xs"
          aria-label={t("Previous window")}
          title={t("Previous window (←)")}
          onClick={() => shiftWindow(-1)}
        >
          <ChevronLeftIcon className="size-3.5" />
        </Button>
        <Button
          type="button"
          variant="outline"
          size="xxs"
          title={t("Jump to today (T)")}
          onClick={goToday}
          disabled={isTodayAnchor(anchor)}
        >
          {t("Today")}
        </Button>
        <Button
          type="button"
          variant="ghost"
          size="icon-xs"
          aria-label={t("Next window")}
          title={t("Next window (→)")}
          onClick={() => shiftWindow(1)}
        >
          <ChevronRightIcon className="size-3.5" />
        </Button>
      </div>

      <Popover>
        <PopoverTrigger
          className="flex shrink-0 items-center gap-1.5 rounded-md ui-focus-ring px-1.5 py-0.5 text-xs font-medium whitespace-nowrap transition-colors hover:bg-surface-hover"
          aria-label={t("Jump to date")}
        >
          <CalendarIcon className="size-3 text-muted-foreground" />
          {formatRangeLabelForDays(anchor, zoom, locale)}
        </PopoverTrigger>
        <PopoverContent align="start" className="w-auto p-0">
          <Calendar
            mode="single"
            selected={anchor}
            defaultMonth={anchor}
            onSelect={(date) => date && setAnchor(date)}
          />
        </PopoverContent>
      </Popover>

      <div
        role="group"
        aria-label={t("Window length")}
        className="inline-flex shrink-0 overflow-hidden rounded-md border border-border whitespace-nowrap"
      >
        {TIMELINE_ZOOM_OPTIONS.map((option, index) => (
          <button
            key={option.id}
            type="button"
            onClick={() => setZoom(option.id)}
            aria-pressed={zoom === option.id}
            className={cn(
              "px-2 py-1 text-xs ui-inset-focus-ring transition-colors",
              index > 0 && "border-l border-border",
              zoom === option.id
                ? "bg-muted text-foreground"
                : "bg-background text-muted-foreground hover:text-foreground",
            )}
          >
            {t(option.label)}
          </button>
        ))}
      </div>

      <span className="h-4 w-px shrink-0 bg-border" aria-hidden />

      <Input
        ref={searchRef}
        value={draft}
        onChange={(event) => {
          setDraft(event.target.value);
          setBoxSearch(event.target.value);
        }}
        onKeyDown={(event) => {
          if (event.key === "Escape" && draft) {
            event.preventDefault();
            setDraft("");
            setBoxSearch("");
          }
        }}
        placeholder={onDrivers ? t("Search drivers") : t("Search PRO, customer or city")}
        aria-label={
          onDrivers
            ? t("Search drivers by name, tractor code, or fleet")
            : t("Search moves by PRO, customer or city")
        }
        leftElement={<SearchLgIcon className="size-3.5 text-muted-foreground" />}
        rightElement={<Kbd className="h-4 min-w-4 text-3xs">/</Kbd>}
        inputContainerClassName="w-60 min-w-40 shrink"
        className="h-7 text-xs md:text-xs"
      />

      <div className="shrink-0">
        <MultiSelectAutocomplete<GraphQLSelectOption>
          graphql={FLEET_SELECT_OPTIONS}
          label={t("Fleet")}
          placeholder={t("Fleet: all")}
          values={[...fleetCodeIds]}
          onChange={(values) => setFleets(selectedIds(values))}
          getOptionValue={(option) => option.id}
          getDisplayValue={(option) => option.label}
          renderOption={(option) => <span className="text-xs">{option.label}</span>}
          triggerClassName="h-7 w-32 text-xs"
          maxCount={1}
        />
      </div>
      <div className="shrink-0">
        <MultiSelectAutocomplete<GraphQLSelectOption>
          graphql={CUSTOMER_SELECT_OPTIONS}
          label={t("Customer")}
          placeholder={t("Customer: all")}
          values={[...customerIds]}
          onChange={(values) => setCustomers(selectedIds(values))}
          getOptionValue={(option) => option.id}
          getDisplayValue={(option) => option.label}
          renderOption={(option) => <span className="text-xs">{option.label}</span>}
          triggerClassName="h-7 w-36 text-xs"
          maxCount={1}
        />
      </div>

      <label className="ml-auto flex shrink-0 items-center gap-1.5 text-xs whitespace-nowrap text-muted-foreground">
        <Switch checked={includeCovered} onCheckedChange={setIncludeCovered} />
        {t("Show covered")}
      </label>
    </div>
  );
}
