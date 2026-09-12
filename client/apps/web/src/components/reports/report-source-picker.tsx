import { useT } from "@trenova/shared/i18n/use-t";
import {
  useCannedReports,
  useReportDefinition,
  useReportDefinitionOptions,
  type ReportDefinitionOption,
} from "@/hooks/use-reports";
import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { useDebounce } from "@trenova/shared/hooks/use-debounce";
import { cn } from "@trenova/shared/lib/utils";
import { CheckIcon, SearchIcon, SparklesIcon, XIcon } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";

const SEARCH_DEBOUNCE_MS = 250;
/** How close to the bottom of the list a scroll gets before the next page is asked for. */
const SCROLL_BUFFER_PX = 64;

export type ReportSource = {
  definitionId: string | null;
  cannedKey: string | null;
};

type Tab = "saved" | "gallery";

export type ReportSourcePickerProps = {
  value: ReportSource;
  onChange: (next: ReportSource) => void;
  /** Ties the picker to an external <Label>. */
  labelledBy?: string;
  className?: string;
};

type Row = {
  id: string;
  name: string;
  description: string | null;
  meta: string | null;
  selected: boolean;
  onSelect: () => void;
};

function initialTab(value: ReportSource): Tab {
  return value.cannedKey ? "gallery" : "saved";
}

/**
 * Chooses the report behind a tile. The saved library is paged and searched by
 * the server — an organization's report library is unbounded, and the previous
 * picker asked for the first hundred rows with their whole definition blobs and
 * offered no way to search at all, so any report outside that hundred was
 * simply unreachable. The gallery is a fixed, code-resident catalog, so it is
 * held whole and filtered here.
 */
export function ReportSourcePicker({
  value,
  onChange,
  labelledBy,
  className,
}: ReportSourcePickerProps) {
  const t = useT();

  const [tab, setTab] = useState<Tab>(() => initialTab(value));
  const [search, setSearch] = useState("");
  const debouncedSearch = useDebounce(search, SEARCH_DEBOUNCE_MS);
  const listRef = useRef<HTMLDivElement>(null);

  const savedTab = tab === "saved";
  const term = search.trim().toLowerCase();

  const definitions = useReportDefinitionOptions(debouncedSearch.trim(), savedTab);
  const canned = useCannedReports(!savedTab);

  // A report chosen on an earlier page must still show as chosen. The library
  // is paged, so a selection the loaded pages do not carry is fetched by id and
  // pinned above them rather than waiting for the user to scroll back to where
  // it lives. A selection already on the page needs no second read.
  const onLoadedPage =
    value.definitionId != null &&
    (definitions.data ?? []).some((entry) => entry.id === value.definitionId);
  const selectedDefinition = useReportDefinition(
    savedTab && value.definitionId && !onLoadedPage ? value.definitionId : undefined,
  );

  // Switching source kind is a decision about which report, so the other kind's
  // key is cleared as part of choosing — a config carrying both is rejected by
  // the server.
  const chooseDefinition = useCallback(
    (id: string) => onChange({ definitionId: id, cannedKey: null }),
    [onChange],
  );
  const chooseCanned = useCallback(
    (key: string) => onChange({ definitionId: null, cannedKey: key }),
    [onChange],
  );

  const savedRows = useMemo<Row[]>(() => {
    const pages: ReportDefinitionOption[] = definitions.data ?? [];
    const rows = pages.map((entry) => toSavedRow(entry, value.definitionId, chooseDefinition));

    const selected = selectedDefinition.data;
    if (selected && !rows.some((row) => row.id === selected.id)) {
      rows.unshift(toSavedRow(selected, value.definitionId, chooseDefinition));
    }

    return rows;
  }, [definitions.data, selectedDefinition.data, value.definitionId, chooseDefinition]);

  const galleryRows = useMemo<Row[]>(
    () =>
      (canned.data ?? [])
        .filter(
          (entry) =>
            term === "" ||
            entry.name.toLowerCase().includes(term) ||
            (entry.description ?? "").toLowerCase().includes(term) ||
            (entry.category ?? "").toLowerCase().includes(term),
        )
        .map((entry) => ({
          id: entry.key,
          name: entry.name,
          description: entry.description ?? null,
          meta: entry.category || null,
          selected: value.cannedKey === entry.key,
          onSelect: () => chooseCanned(entry.key),
        })),
    [canned.data, term, value.cannedKey, chooseCanned],
  );

  const rows = savedTab ? savedRows : galleryRows;
  const loading = savedTab ? definitions.isLoading : canned.isLoading;
  const error = savedTab ? definitions.error : canned.error;
  const { hasNextPage, isFetchingNextPage, fetchNextPage } = definitions;

  // A fresh search scrolls back to the top, or the first result sits above the
  // viewport and the list reads as though nothing changed.
  useEffect(() => {
    listRef.current?.scrollTo({ top: 0 });
  }, [debouncedSearch, tab]);

  const handleScroll = useCallback(() => {
    if (!savedTab || !hasNextPage || isFetchingNextPage) return;
    const list = listRef.current;
    if (!list) return;
    if (list.scrollHeight - (list.scrollTop + list.clientHeight) <= SCROLL_BUFFER_PX) {
      void fetchNextPage();
    }
  }, [savedTab, hasNextPage, isFetchingNextPage, fetchNextPage]);

  return (
    <div
      className={cn("border-border bg-background flex flex-col rounded-md border", className)}
      aria-labelledby={labelledBy}
    >
      <div className="border-border/70 flex flex-col gap-2 border-b p-2">
        <SegmentedControl
          aria-label={t("Where the report comes from")}
          fullWidth
          value={tab}
          onValueChange={setTab}
          items={[
            { value: "saved", label: "Saved reports" },
            { value: "gallery", label: "Report gallery", icon: SparklesIcon },
          ]}
        />
        <Input
          aria-label={savedTab ? "Search saved reports" : "Search the report gallery"}
          placeholder={savedTab ? "Search saved reports…" : "Search the gallery…"}
          value={search}
          onChange={(event) => setSearch(event.target.value)}
          leftElement={<SearchIcon className="text-muted-foreground size-3.5" />}
          rightElement={
            search ? (
              <Button
                variant="ghost"
                size="icon-sm"
                aria-label={t("Clear search")}
                className="size-6"
                onClick={() => setSearch("")}
              >
                <XIcon className="size-3.5" />
              </Button>
            ) : undefined
          }
        />
      </div>

      <div
        ref={listRef}
        onScroll={handleScroll}
        role="listbox"
        aria-label={savedTab ? "Saved reports" : "Report gallery"}
        className="flex max-h-56 min-h-32 flex-col gap-1 overflow-y-auto overscroll-contain p-1.5"
      >
        {error ? (
          <PickerMessage>
            <p className="text-destructive text-xs">{t("Could not load reports.")}</p>
            <Button
              size="sm"
              variant="outline"
              onClick={() => void (savedTab ? definitions.refetch() : canned.refetch())}
            >
              {t("Retry")}
            </Button>
          </PickerMessage>
        ) : loading ? (
          <div className="flex flex-col gap-1">
            {[0, 1, 2, 3].map((row) => (
              <Skeleton key={row} className="h-9 rounded" />
            ))}
          </div>
        ) : rows.length === 0 ? (
          <PickerMessage>
            <p className="text-xs font-medium">
              {search.trim() === "" ? t("Nothing here yet") : t("Nothing matches")}
            </p>
            <p className="text-muted-foreground text-center text-xs">
              {search.trim() === ""
                ? savedTab
                  ? t("You have no saved reports yet. Build one in Reports, or pick from the gallery.")
                  : t("The report gallery is empty on this deployment.")
                : t("No report matches “{0}”.", search.trim())}
            </p>
            {search.trim() !== "" && (
              <Button size="sm" variant="outline" onClick={() => setSearch("")}>
                {t("Clear search")}
              </Button>
            )}
          </PickerMessage>
        ) : (
          <>
            {rows.map((row) => (
              <PickerRow key={row.id} row={row} />
            ))}

            {savedTab && hasNextPage && (
              <Button
                variant="ghost"
                size="sm"
                className="text-muted-foreground h-7 w-full"
                disabled={isFetchingNextPage}
                onClick={() => void fetchNextPage()}
              >
                {isFetchingNextPage ? (
                  <>
                    <Spinner className="size-3" />
                    {t("Loading more")}
                  </>
                ) : (
                  t("Load more")
                )}
              </Button>
            )}
          </>
        )}
      </div>
    </div>
  );
}

function toSavedRow(
  entry: { id: string; name: string; description: string; category: string; status: string },
  selectedId: string | null,
  onSelect: (id: string) => void,
): Row {
  return {
    id: entry.id,
    name: entry.name,
    description: entry.description || null,
    meta: entry.category || (entry.status === "draft" ? "Draft" : null),
    selected: selectedId === entry.id,
    onSelect: () => onSelect(entry.id),
  };
}

function PickerRow({ row }: { row: Row }) {
  const t = useT();

  return (
    <button
      type="button"
      role="option"
      aria-selected={row.selected}
      onClick={row.onSelect}
      className={cn(
        "flex w-full items-start gap-2 rounded px-2 py-1.5 text-left transition-colors",
        "focus-visible:ring-brand/40 focus-visible:ring-2 focus-visible:outline-none",
        row.selected
          ? "border-brand/50 bg-brand/10 border"
          : "hover:bg-muted/70 border border-transparent",
      )}
    >
      <CheckIcon
        className={cn(
          "text-brand mt-0.5 size-3 shrink-0 transition-opacity",
          row.selected ? "opacity-100" : "opacity-0",
        )}
      />
      <span className="flex min-w-0 flex-1 flex-col">
        <span className="truncate text-xs font-medium">{row.name}</span>
        {row.description && (
          <span className="text-muted-foreground truncate text-[11px]">{t(row.description)}</span>
        )}
      </span>
      {row.meta && (
        <span className="text-muted-foreground/80 mt-0.5 shrink-0 text-[9px] tracking-wide uppercase">
          {row.meta}
        </span>
      )}
    </button>
  );
}

function PickerMessage({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex flex-1 flex-col items-center justify-center gap-2 px-4 py-6">
      {children}
    </div>
  );
}
