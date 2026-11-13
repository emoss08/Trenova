import type { VirtualRow, VirtualRowsHandle } from "@/components/virtual-rows";
import { DeskList, DeskListHeading, DeskListStill } from "@/routes/desk/_components/list/desk-list";
import { DeskListToggle } from "@/routes/desk/_components/list/desk-list-head";
import { DeskListSearch } from "@/routes/desk/_components/list/desk-list-search";
import type { InboundMessageRow, TriageSort } from "@/lib/graphql/inbox";
import { useTriageStore, type TriageRowMode } from "@/stores/triage-store";
import { openContextMenu } from "@trenova/shared/components/context-menu/context-menu-host";
import { CheckIcon, Rows01Icon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { useMemo, type ReactNode, type Ref, type RefObject } from "react";
import type { FolderSort } from "../folders";
import type { ListItem } from "./list-items";
import { ListSkeleton } from "./list-states";
import { RowHandlersContext, TriageRow, type RowHandlers } from "./triage-row";
import { BulkBar } from "./bulk-bar";

/** A row's height before it is measured: three lines and their padding. */
const ROW_ESTIMATE = 74;

export type TriageListProps = {
  /** "Search waiting on you". */
  searchPlaceholder: string;
  search: string;
  onSearch: (value: string) => void;
  onSaveView: (query: string) => void;
  searchRef: RefObject<HTMLInputElement | null>;
  /** The order the folder is read in, once the server's answer is known. */
  sort: TriageSort;
  /** Whether the folder offers "Due first / Newest": only when something in it waits. */
  canSort: boolean;
  onSort: (sort: FolderSort) => void;
  items: readonly ListItem[];
  messages: readonly InboundMessageRow[];
  isLoading: boolean;
  openId: string | null;
  now: number;
  handlers: RowHandlers;
  listRef: Ref<VirtualRowsHandle>;
  onEndReached: () => void;
  isFetchingNextPage: boolean;
  /** Notices above the rows: a source gone quiet, Nova off. */
  banner?: ReactNode;
  /** What the folder shows with nothing in it. */
  empty: ReactNode;
};

function SortToggle({ sort, onSort }: { sort: TriageSort; onSort: (sort: FolderSort) => void }) {
  const t = useT();

  return (
    <div className="px-0.5">
      <DeskListToggle<FolderSort>
        label={t("Order")}
        value={sort === "Due" ? "due" : "newest"}
        onChange={onSort}
        options={[
          { value: "due", label: t("Due first") },
          { value: "newest", label: t("Newest") },
        ]}
      />
    </div>
  );
}

function RowModeButton() {
  const t = useT();
  const mode = useTriageStore((state) => state.rowMode);
  const setMode = useTriageStore((state) => state.setRowMode);
  const choices: Array<[TriageRowMode, string]> = [
    ["next", t("Next step")],
    ["preview", t("Preview")],
  ];

  return (
    <Button
      size="icon-xs"
      variant="quiet"
      aria-label={t("Rows show")}
      onClick={(event) => {
        const rect = event.currentTarget.getBoundingClientRect();
        openContextMenu(
          { x: rect.left, y: rect.bottom + 4 },
          {
            title: t("Rows show"),
            entries: choices.map(([value, label]) => ({
              id: value,
              label,
              leading: mode === value ? <CheckIcon className="size-3.5" /> : undefined,
              onSelect: () => setMode(value),
            })),
          },
          event.currentTarget,
        );
      }}
    >
      <Rows01Icon aria-hidden className="size-3.5" />
    </Button>
  );
}

/**
 * The open folder's messages: a search over it, the order it is read in,
 * then the rows, grouped, drawn only near the viewport and paged in as the
 * end comes into view. Selecting a row touches that row alone; the bulk bar
 * floats over the foot of the list while anything is selected.
 */
export function TriageList({
  searchPlaceholder,
  search,
  onSearch,
  onSaveView,
  searchRef,
  sort,
  canSort,
  onSort,
  items,
  messages,
  isLoading,
  openId,
  now,
  handlers,
  listRef,
  onEndReached,
  isFetchingNextPage,
  banner,
  empty,
}: TriageListProps) {
  const t = useT();
  const mode = useTriageStore((state) => state.rowMode);

  const rows = useMemo<VirtualRow[]>(
    () =>
      items.map((item) =>
        item.type === "heading"
          ? {
              key: item.key,
              sticky: true,
              render: () => (
                <DeskListHeading tone={item.tone === "danger" ? "danger" : "quiet"}>
                  {item.label}
                </DeskListHeading>
              ),
            }
          : {
              key: item.key,
              render: () => (
                <TriageRow
                  message={item.message}
                  active={item.message.id === openId}
                  mode={mode}
                  now={now}
                />
              ),
            },
      ),
    [items, openId, mode, now],
  );

  return (
    <section aria-label={searchPlaceholder} className="relative flex min-h-0 flex-1 flex-col">
      <header className="flex flex-col gap-2 px-3 pt-2.5 pb-2">
        <DeskListSearch
          inputRef={searchRef}
          placeholder={searchPlaceholder}
          value={search}
          onCommit={onSearch}
          onSave={onSaveView}
        />
        <div className="flex min-h-5.5 items-center justify-between">
          {canSort ? <SortToggle sort={sort} onSort={onSort} /> : <span />}
          <RowModeButton />
        </div>
      </header>
      {banner}
      <RowHandlersContext.Provider value={handlers}>
        {isLoading ? (
          <ListSkeleton />
        ) : (
          <DeskList
            rows={rows}
            estimateSize={ROW_ESTIMATE}
            initialHeight={800}
            overscan={6}
            handleRef={listRef}
            onEndReached={onEndReached}
            rowClassName="pb-px"
            aria-label={t("Messages")}
            role="listbox"
            aria-multiselectable
            empty={<DeskListStill>{empty}</DeskListStill>}
            footer={
              // Room under the last row for the bar that acts on the picked ones.
              <div className="min-h-12">
                {isFetchingNextPage && (
                  <p className="py-3 text-center text-xs text-dsk-subtle">{t("Loading more…")}</p>
                )}
              </div>
            }
          />
        )}
      </RowHandlersContext.Provider>
      <BulkBar messages={messages} />
    </section>
  );
}
